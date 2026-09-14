// Package scanner implements Trivy CVE scanning integration for KTTM.
//
// KTTM-REQ-046: Before any workflow pod is deployed, all container images
// built by the NodeBuilderFactory are scanned for known CVEs. Images with
// CRITICAL-severity vulnerabilities block deployment until remediated.
//
// KTTM-REQ-047: Compliance mode can enforce a zero-HIGH policy for regulated
// environments (finance, healthcare, public sector).
//
// The scanner supports both online mode (Trivy server) and offline mode
// (bundled Trivy DB in air-gapped clusters).
package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
)

// ─────────────────────────────────────────────
//  Config
// ─────────────────────────────────────────────

// Config configures the CVE scanner.
type Config struct {
	// TrivyBin is the path to the trivy binary (default: "trivy" in $PATH).
	TrivyBin string

	// ServerURL is the optional Trivy server URL for remote scanning.
	// If empty, trivy is invoked in standalone mode with local DB.
	ServerURL string

	// OfflineDB is the path to a pre-downloaded Trivy vulnerability database.
	// Required in air-gapped environments.
	OfflineDB string

	// BlockOnCritical stops deployment when CRITICAL CVEs are found.
	BlockOnCritical bool

	// BlockOnHigh stops deployment when HIGH CVEs are found (compliance mode).
	BlockOnHigh bool

	// Timeout is the maximum time allowed for a single scan.
	Timeout time.Duration
}

// DefaultConfig returns sensible production defaults.
func DefaultConfig() Config {
	return Config{
		TrivyBin:        "trivy",
		BlockOnCritical: true,
		BlockOnHigh:     false,
		Timeout:         5 * time.Minute,
	}
}

// ─────────────────────────────────────────────
//  Scanner
// ─────────────────────────────────────────────

// Scanner wraps Trivy CLI/server to scan OCI images before deployment.
type Scanner struct {
	cfg Config
}

// New creates a new Trivy-backed CVE scanner.
func New(cfg Config) *Scanner {
	return &Scanner{cfg: cfg}
}

// ─────────────────────────────────────────────
//  ScanResult
// ─────────────────────────────────────────────

// ScanResult holds the CVE summary for a scanned image.
type ScanResult struct {
	ImageRef   string       `json:"imageRef"`
	Clean      bool         `json:"clean"`
	Critical   int          `json:"critical"`
	High       int          `json:"high"`
	Medium     int          `json:"medium"`
	Low        int          `json:"low"`
	ScannedAt  time.Time    `json:"scannedAt"`
	Findings   []CVEFinding `json:"findings,omitempty"`
	SBOMDigest string       `json:"sbomDigest,omitempty"` // SHA256 of the generated SBOM
}

// CVEFinding represents a single vulnerability entry.
type CVEFinding struct {
	ID          string `json:"id"`       // e.g., CVE-2024-12345
	Severity    string `json:"severity"` // CRITICAL | HIGH | MEDIUM | LOW
	Package     string `json:"package"`  // e.g., openssl
	Version     string `json:"version"`
	FixedIn     string `json:"fixedIn,omitempty"`
	Description string `json:"description,omitempty"`
	Link        string `json:"link,omitempty"`
}

// IsBlocked returns true if the scan result blocks deployment based on the scanner config.
func (s *Scanner) IsBlocked(result *ScanResult) bool {
	if s.cfg.BlockOnCritical && result.Critical > 0 {
		return true
	}
	if s.cfg.BlockOnHigh && result.High > 0 {
		return true
	}
	return false
}

// ─────────────────────────────────────────────
//  Scanning
// ─────────────────────────────────────────────

// ScanImage runs a Trivy CVE scan on the given OCI image reference.
// Returns a ScanResult and whether deployment should be blocked.
func (s *Scanner) ScanImage(ctx context.Context, imageRef string) (*ScanResult, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	args := buildTrivyArgs(imageRef, s.cfg)

	fmt.Printf("[Scanner] Scanning image: %s\n", imageRef)
	out, err := exec.CommandContext(ctx, s.cfg.TrivyBin, args...).Output()
	if err != nil {
		// Trivy exits non-zero when vulnerabilities are found — handle gracefully
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// This is a "vulnerabilities found" exit, not a scanner error
		} else {
			return nil, fmt.Errorf("scanner: trivy failed for %s: %w", imageRef, err)
		}
	}

	result, err := parseTrivyJSON(imageRef, out)
	if err != nil {
		return nil, fmt.Errorf("scanner: parsing trivy output: %w", err)
	}

	fmt.Printf("[Scanner] %s → critical=%d high=%d medium=%d\n",
		imageRef, result.Critical, result.High, result.Medium)
	return result, nil
}

// ScanApp scans all container images referenced by a KttmApp's DAG nodes.
// Returns the first blocking result found, or nil if all images are clean.
func (s *Scanner) ScanApp(ctx context.Context, app *kttmv1.KttmApp) ([]*ScanResult, error) {
	var results []*ScanResult
	seen := make(map[string]bool)

	for _, node := range app.Spec.WorkflowDAG.Nodes {
		if node.Image == "" {
			continue // no explicit image; built-in connector
		}
		if seen[node.Image] {
			continue // already scanned this image in a previous node
		}
		seen[node.Image] = true

		result, err := s.ScanImage(ctx, node.Image)
		if err != nil {
			return results, fmt.Errorf("scanning node %s: %w", node.ID, err)
		}
		results = append(results, result)
	}
	return results, nil
}

// ToStatusResult converts a scan result to the KttmApp status type.
func ToStatusResult(results []*ScanResult) kttmv1.ScanResult {
	sr := kttmv1.ScanResult{Clean: true}
	for _, r := range results {
		sr.Critical += r.Critical
		sr.High += r.High
		sr.Medium += r.Medium
		if r.Critical > 0 || r.High > 0 {
			sr.Clean = false
		}
	}
	return sr
}

// ─────────────────────────────────────────────
//  Trivy CLI helpers
// ─────────────────────────────────────────────

// buildTrivyArgs constructs the trivy CLI argument list.
func buildTrivyArgs(imageRef string, cfg Config) []string {
	args := []string{
		"image",
		"--format", "json",
		"--exit-code", "1", // exit 1 if vulnerabilities found
		"--severity", "CRITICAL,HIGH,MEDIUM,LOW",
	}

	if cfg.ServerURL != "" {
		args = append(args, "--server", cfg.ServerURL)
	}
	if cfg.OfflineDB != "" {
		args = append(args, "--skip-update", "--cache-dir", cfg.OfflineDB)
	}

	args = append(args, imageRef)
	return args
}

// trivyReport is the top-level structure of Trivy's JSON output.
type trivyReport struct {
	Results []struct {
		Vulnerabilities []struct {
			VulnerabilityID  string `json:"VulnerabilityID"`
			PkgName          string `json:"PkgName"`
			InstalledVersion string `json:"InstalledVersion"`
			FixedVersion     string `json:"FixedVersion"`
			Severity         string `json:"Severity"`
			Title            string `json:"Title"`
			PrimaryURL       string `json:"PrimaryURL"`
		} `json:"Vulnerabilities"`
	} `json:"Results"`
}

// parseTrivyJSON parses the Trivy JSON output into a ScanResult.
func parseTrivyJSON(imageRef string, data []byte) (*ScanResult, error) {
	if len(data) == 0 {
		// No output = clean
		return &ScanResult{ImageRef: imageRef, Clean: true, ScannedAt: time.Now()}, nil
	}

	var report trivyReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("unmarshal trivy JSON: %w", err)
	}

	result := &ScanResult{
		ImageRef:  imageRef,
		ScannedAt: time.Now(),
		Clean:     true,
	}

	for _, r := range report.Results {
		for _, v := range r.Vulnerabilities {
			finding := CVEFinding{
				ID:          v.VulnerabilityID,
				Severity:    v.Severity,
				Package:     v.PkgName,
				Version:     v.InstalledVersion,
				FixedIn:     v.FixedVersion,
				Description: v.Title,
				Link:        v.PrimaryURL,
			}
			result.Findings = append(result.Findings, finding)
			switch v.Severity {
			case "CRITICAL":
				result.Critical++
				result.Clean = false
			case "HIGH":
				result.High++
			case "MEDIUM":
				result.Medium++
			case "LOW":
				result.Low++
			}
		}
	}

	return result, nil
}
