package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
)

func TestScannerConfigPolicyAndArguments(t *testing.T) {
	defaults := DefaultConfig()
	if defaults.TrivyBin != "trivy" || !defaults.BlockOnCritical || defaults.BlockOnHigh || defaults.Timeout != 5*time.Minute {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
	if !New(Config{BlockOnCritical: true}).IsBlocked(&ScanResult{Critical: 1}) {
		t.Fatal("critical result was not blocked")
	}
	if !New(Config{BlockOnHigh: true}).IsBlocked(&ScanResult{High: 1}) {
		t.Fatal("high result was not blocked in compliance mode")
	}
	if New(Config{BlockOnCritical: true}).IsBlocked(&ScanResult{High: 1}) || New(Config{BlockOnHigh: true}).IsBlocked(&ScanResult{Medium: 1}) {
		t.Fatal("scanner blocked a result below configured severity")
	}

	args := buildTrivyArgs("image:v1", Config{ServerURL: "http://trivy:4954", OfflineDB: "/cache"})
	joined := strings.Join(args, " ")
	for _, expected := range []string{"image --format json", "--server http://trivy:4954", "--skip-update --cache-dir /cache", "image:v1"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("buildTrivyArgs() missing %q: %v", expected, args)
		}
	}
}

func TestParseTrivyJSON(t *testing.T) {
	clean, err := parseTrivyJSON("clean:v1", nil)
	if err != nil || !clean.Clean || clean.ImageRef != "clean:v1" || clean.ScannedAt.IsZero() {
		t.Fatalf("empty report = (%+v, %v)", clean, err)
	}
	data := []byte(`{"Results":[{"Vulnerabilities":[` +
		`{"VulnerabilityID":"C1","PkgName":"openssl","InstalledVersion":"1","FixedVersion":"2","Severity":"CRITICAL","Title":"critical","PrimaryURL":"https://c1"},` +
		`{"VulnerabilityID":"H1","PkgName":"curl","InstalledVersion":"3","Severity":"HIGH"},` +
		`{"VulnerabilityID":"M1","PkgName":"lib","InstalledVersion":"4","Severity":"MEDIUM"},` +
		`{"VulnerabilityID":"L1","PkgName":"app","InstalledVersion":"5","Severity":"LOW"},` +
		`{"VulnerabilityID":"U1","PkgName":"other","InstalledVersion":"6","Severity":"UNKNOWN"}` +
		`]}]}`)
	result, err := parseTrivyJSON("app:v1", data)
	if err != nil {
		t.Fatalf("parseTrivyJSON() error = %v", err)
	}
	if result.Clean || result.Critical != 1 || result.High != 1 || result.Medium != 1 || result.Low != 1 || len(result.Findings) != 5 {
		t.Fatalf("unexpected parsed result: %+v", result)
	}
	if result.Findings[0].FixedIn != "2" || result.Findings[0].Description != "critical" || result.Findings[0].Link != "https://c1" {
		t.Fatalf("finding fields were not mapped: %+v", result.Findings[0])
	}
	if _, err := parseTrivyJSON("bad:v1", []byte("{")); err == nil {
		t.Fatal("parseTrivyJSON() accepted malformed JSON")
	}
}

func TestScanImageSuccessVulnerabilitiesAndFailures(t *testing.T) {
	binary := fakeTrivy(t, `printf '%s' "$FAKE_TRIVY_OUTPUT"; exit "${FAKE_TRIVY_EXIT:-0}"`)
	t.Setenv("FAKE_TRIVY_OUTPUT", `{"Results":[{"Vulnerabilities":[{"VulnerabilityID":"CVE-1","Severity":"CRITICAL"}]}]}`)
	t.Setenv("FAKE_TRIVY_EXIT", "1")
	scanner := New(Config{TrivyBin: binary, Timeout: 10 * time.Second})
	result, err := scanner.ScanImage(context.Background(), "app:v1")
	if err != nil || result.Critical != 1 || result.Clean {
		t.Fatalf("ScanImage(vulnerability exit) = (%+v, %v)", result, err)
	}

	t.Setenv("FAKE_TRIVY_OUTPUT", "not-json")
	t.Setenv("FAKE_TRIVY_EXIT", "0")
	if _, err := scanner.ScanImage(context.Background(), "app:v1"); err == nil || !strings.Contains(err.Error(), "parsing trivy output") {
		t.Fatalf("ScanImage(malformed output) error = %v", err)
	}

	if _, err := New(Config{TrivyBin: filepath.Join(t.TempDir(), "missing"), Timeout: 10 * time.Second}).ScanImage(context.Background(), "missing:v1"); err == nil {
		t.Fatal("ScanImage() succeeded for missing binary")
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scanner.ScanImage(canceled, "app:v1"); err == nil {
		t.Fatal("ScanImage() succeeded with a canceled context")
	}

	failing := fakeTrivy(t, `exit 2`)
	if _, err := New(Config{TrivyBin: failing, Timeout: 10 * time.Second}).ScanImage(context.Background(), "bad:v1"); err == nil {
		t.Fatal("ScanImage() accepted an unexpected Trivy exit code")
	}
}

func TestScanAppDeduplicatesAndReturnsPartialResults(t *testing.T) {
	binary := fakeTrivy(t, `case "$*" in *bad:v1*) exit 2;; esac; printf '{"Results":[]}'`)
	scanner := New(Config{TrivyBin: binary, Timeout: 10 * time.Second})
	app := &kttmv1.KttmApp{Spec: kttmv1.KttmAppSpec{WorkflowDAG: kttmv1.WorkflowDAGSpec{Nodes: []kttmv1.WorkflowNode{
		{ID: "built-in", Type: "connector/s3"},
		{ID: "first", Image: "good:v1"},
		{ID: "duplicate", Image: "good:v1"},
	}}}}
	results, err := scanner.ScanApp(context.Background(), app)
	if err != nil || len(results) != 1 || results[0].ImageRef != "good:v1" {
		t.Fatalf("ScanApp() = (%+v, %v), want one deduplicated result", results, err)
	}
	app.Spec.WorkflowDAG.Nodes = append(app.Spec.WorkflowDAG.Nodes, kttmv1.WorkflowNode{ID: "failing", Image: "bad:v1"})
	results, err = scanner.ScanApp(context.Background(), app)
	if err == nil || !strings.Contains(err.Error(), "scanning node failing") || len(results) != 1 {
		t.Fatalf("ScanApp() failure = (%+v, %v), want prior result and node error", results, err)
	}
	if results, err := scanner.ScanApp(context.Background(), &kttmv1.KttmApp{}); err != nil || len(results) != 0 {
		t.Fatalf("ScanApp(empty) = (%+v, %v)", results, err)
	}
}

func TestToStatusResult(t *testing.T) {
	clean := ToStatusResult(nil)
	if !clean.Clean || clean.Critical != 0 || clean.High != 0 || clean.Medium != 0 {
		t.Fatalf("empty status result = %+v", clean)
	}
	result := ToStatusResult([]*ScanResult{{Critical: 1, High: 2, Medium: 3}, {High: 1, Medium: 2}})
	if result.Clean || result.Critical != 1 || result.High != 3 || result.Medium != 5 {
		t.Fatalf("aggregated status result = %+v", result)
	}
	mediumOnly := ToStatusResult([]*ScanResult{{Medium: 1}})
	if !mediumOnly.Clean {
		t.Fatalf("medium-only result should remain clean: %+v", mediumOnly)
	}
}

func fakeTrivy(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trivy")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatalf("write fake trivy: %v", err)
	}
	return path
}

var _ = errors.Is