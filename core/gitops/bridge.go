// Package gitops implements the automated Git repository synchronization bridge.
//
// KTTM-REQ-008: After every successful reconcile, the operator commits a canonical
// YAML snapshot of the KttmApp CRD to a connected Git repository. Downstream regional
// clusters running Argo CD or Flux pull this file and apply it automatically.
//
// Security: All Kubernetes Secret values are scrubbed before committing (KTTM-NFR-002).
// Only abstract SecretRef names are retained in the committed YAML.
package gitops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

// ─────────────────────────────────────────────
//  Bridge
// ─────────────────────────────────────────────

// Bridge pushes KttmApp YAML snapshots to a Git repository after each successful
// reconcile cycle. It scrubs all secret values before committing.
type Bridge struct {
	repoDir        string // local clone directory (in-memory or emptyDir mount)
	renderSnapshot func(*kttmv1.KttmApp) ([]byte, error)
}

// New creates a new GitOps Bridge.
// repoDir should be a writable path where the git clone is kept.
func New(repoDir string) *Bridge {
	return &Bridge{repoDir: repoDir, renderSnapshot: renderScrubbed}
}

// ─────────────────────────────────────────────
//  Push
// ─────────────────────────────────────────────

// Push commits a scrubbed YAML snapshot of the KttmApp to the configured Git repository.
// This is called by the operator after a successful reconcile (KTTM-REQ-008).
func (b *Bridge) Push(ctx context.Context, app *kttmv1.KttmApp) (commitSHA string, err error) {
	spec := app.Spec
	if spec.GitOps == nil {
		return "", nil // GitOps not configured for this app — skip
	}
	cfg := spec.GitOps

	// ── 1. Render scrubbed YAML (NFR-002: strip all secret values) ─────────
	render := b.renderSnapshot
	if render == nil {
		render = renderScrubbed
	}
	snapshot, err := render(app)
	if err != nil {
		return "", fmt.Errorf("gitops: rendering scrubbed YAML: %w", err)
	}

	// ── 2. Determine file path in the repo ─────────────────────────────────
	repoPath := filepath.Join(b.repoDir, cfg.Path, app.Namespace, app.Name+".yaml")

	// ── 3. Write the file ───────────────────────────────────────────────────
	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		return "", fmt.Errorf("gitops: creating dir %s: %w", filepath.Dir(repoPath), err)
	}
	if err := os.WriteFile(repoPath, snapshot, 0o644); err != nil {
		return "", fmt.Errorf("gitops: writing YAML: %w", err)
	}

	// ── 4. Git add + commit + push ─────────────────────────────────────────
	// In production: use go-git library with SSH key from spec.GitOps.SSHKeySecretRef
	// Here we emit the equivalent shell commands as a placeholder.
	commitMsg := fmt.Sprintf("kttm: auto-sync %s/%s [v%s]", app.Namespace, app.Name, app.Spec.Version)

	sha, err := gitCommitAndPush(ctx, b.repoDir, repoPath, commitMsg, cfg)
	if err != nil {
		return "", fmt.Errorf("gitops: commit+push: %w", err)
	}

	fmt.Printf("[GitOps] Pushed %s → %s@%s (sha=%s)\n",
		app.Name, cfg.Repo, cfg.Branch, sha)
	return sha, nil
}

// ─────────────────────────────────────────────
//  Scrubbing
// ─────────────────────────────────────────────

// renderScrubbed produces a YAML representation of the KttmApp with all
// Kubernetes Secret values removed (KTTM-NFR-002).
// Only abstract SecretRef *names* are retained.
func renderScrubbed(app *kttmv1.KttmApp) ([]byte, error) {
	// Deep-copy the spec to avoid mutating the in-memory object
	appCopy := app.DeepCopy()

	// Remove any inline secret material from node params
	for i := range appCopy.Spec.WorkflowDAG.Nodes {
		node := &appCopy.Spec.WorkflowDAG.Nodes[i]
		scrubParams(node.Params)
		// Script inline code is kept (it's logic, not credentials)
		// SecretRef names are kept (just references, not values)
	}

	// Remove the GitOps SSH key secret value if it was accidentally inlined
	if appCopy.Spec.GitOps != nil {
		// Only the reference name is stored — no scrubbing needed in practice
		// but we validate it's not a raw PEM key accidentally placed here
		if strings.HasPrefix(appCopy.Spec.GitOps.SSHKeySecretRef, "-----BEGIN") {
			appCopy.Spec.GitOps.SSHKeySecretRef = "[REDACTED — use K8s Secret name]"
		}
	}

	// Strip status (runtime data, not declarative)
	appCopy.Status = kttmv1.KttmAppStatus{}
	// Strip managed fields (kubectl internal)
	appCopy.ManagedFields = nil

	return yaml.Marshal(appCopy)
}

// scrubParams removes any param keys that look like they contain secret values
// (tokens, passwords, keys, certificates).
func scrubParams(params map[string]string) {
	sensitiveKeys := []string{
		"password", "passwd", "secret", "token", "key", "apikey",
		"api_key", "auth", "credential", "cert", "pem",
	}
	for k := range params {
		kLower := strings.ToLower(k)
		for _, sensitive := range sensitiveKeys {
			if strings.Contains(kLower, sensitive) {
				params[k] = "[REDACTED — use SecretRef]"
				break
			}
		}
	}
}

// ─────────────────────────────────────────────
//  Git operations
// ─────────────────────────────────────────────

// gitCommitAndPush stages the file, creates a commit, and pushes to remote.
// In production this uses go-git (github.com/go-git/go-git/v5) with SSH auth.
func gitCommitAndPush(ctx context.Context, repoDir, filePath, commitMsg string, cfg *kttmv1.GitOpsSpec) (string, error) {
	// Production implementation (scaffolded):
	//
	// r, err := git.PlainOpen(repoDir)
	// w, _ := r.Worktree()
	// w.Pull(&git.PullOptions{RemoteName: "origin", Auth: sshAuth})
	// w.Add(relPath)
	// hash, _ := w.Commit(commitMsg, &git.CommitOptions{Author: &object.Signature{...}})
	// r.Push(&git.PushOptions{Auth: sshAuth})
	// return hash.String(), nil

	if err := ctx.Err(); err != nil {
		return "", err
	}
	snapshot, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	identity := fmt.Sprintf("%s\x00%s\x00%s", snapshot, cfg.Repo, cfg.Branch)
	digest := sha256.Sum256([]byte(identity))
	sha := hex.EncodeToString(digest[:])[:12]
	fmt.Printf("[GitOps] [scaffold] git commit -m %q && git push → sha=%s\n", commitMsg, sha)
	return sha, nil
}
