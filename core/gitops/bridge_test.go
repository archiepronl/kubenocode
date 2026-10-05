package gitops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPushIsRepeatableAndScrubsSecretsWithoutMutation(t *testing.T) {
	repoDir := t.TempDir()
	app := &kttmv1.KttmApp{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "team-a", ManagedFields: []metav1.ManagedFieldsEntry{{Manager: "kubectl"}}},
		Spec: kttmv1.KttmAppSpec{
			Version: "1.2.3",
			GitOps:  &kttmv1.GitOpsSpec{Repo: "https://example.invalid/flows", Branch: "main", Path: "workflows", SSHKeySecretRef: "-----BEGIN PRIVATE KEY-----"},
			WorkflowDAG: kttmv1.WorkflowDAGSpec{Nodes: []kttmv1.WorkflowNode{{
				ID: "source", SecretRef: "aws-creds", Script: "print('not a secret')",
				Params: map[string]string{"password": "do-not-commit", "api_key": "also-secret", "bucket": "reports"},
			}}},
		},
		Status: kttmv1.KttmAppStatus{Phase: "Running", NodeStatuses: map[string]string{"source": "Succeeded"}},
	}

	firstSHA, err := New(repoDir).Push(t.Context(), app)
	if err != nil {
		t.Fatalf("first Push() error = %v", err)
	}
	path := filepath.Join(repoDir, "workflows", "team-a", "orders.yaml")
	firstSnapshot, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read first snapshot: %v", err)
	}
	secondSHA, err := New(repoDir).Push(t.Context(), app)
	if err != nil {
		t.Fatalf("second Push() error = %v", err)
	}
	secondSnapshot, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read second snapshot: %v", err)
	}
	if firstSHA != secondSHA || string(firstSnapshot) != string(secondSnapshot) {
		t.Fatalf("repeated Push() was not idempotent: sha %q != %q", firstSHA, secondSHA)
	}
	for _, secret := range []string{"do-not-commit", "also-secret", "Running", "Succeeded", "kubectl"} {
		if strings.Contains(string(firstSnapshot), secret) {
			t.Errorf("snapshot contains sensitive or runtime value %q", secret)
		}
	}
	for _, retained := range []string{"aws-creds", "print('not a secret')", "reports", "[REDACTED"} {
		if !strings.Contains(string(firstSnapshot), retained) {
			t.Errorf("snapshot omitted declarative value %q", retained)
		}
	}
	if app.Spec.WorkflowDAG.Nodes[0].Params["password"] != "do-not-commit" || app.Status.Phase != "Running" || app.ManagedFields == nil {
		t.Fatal("rendering or pushing mutated the source KttmApp")
	}
}

func TestPushWithoutGitOpsDoesNothing(t *testing.T) {
	sha, err := New(t.TempDir()).Push(t.Context(), &kttmv1.KttmApp{})
	if err != nil || sha != "" {
		t.Fatalf("Push() without GitOps = (%q, %v), want empty success", sha, err)
	}
}

func TestZeroValueBridgeUsesDefaultRenderer(t *testing.T) {
	app := &kttmv1.KttmApp{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "team-a"},
		Spec:       kttmv1.KttmAppSpec{GitOps: &kttmv1.GitOpsSpec{Repo: "https://example.invalid/flows"}},
	}
	sha, err := (&Bridge{repoDir: t.TempDir()}).Push(context.Background(), app)
	if err != nil || len(sha) != 12 {
		t.Fatalf("zero-value renderer Push() = (%q, %v)", sha, err)
	}
}

func TestPushReturnsSnapshotRenderingError(t *testing.T) {
	wantErr := errors.New("render failed")
	bridge := &Bridge{
		repoDir: t.TempDir(),
		renderSnapshot: func(*kttmv1.KttmApp) ([]byte, error) {
			return nil, wantErr
		},
	}
	app := &kttmv1.KttmApp{Spec: kttmv1.KttmAppSpec{GitOps: &kttmv1.GitOpsSpec{Repo: "https://example.invalid/flows"}}}
	if _, err := bridge.Push(context.Background(), app); !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "rendering scrubbed YAML") {
		t.Fatalf("Push() error = %v, want wrapped render error", err)
	}
}

func TestPushFilesystemFailures(t *testing.T) {
	app := &kttmv1.KttmApp{
		ObjectMeta: metav1.ObjectMeta{Name: "blocked", Namespace: "team-a"},
		Spec: kttmv1.KttmAppSpec{GitOps: &kttmv1.GitOpsSpec{Repo: "https://example.invalid/flows", Path: "workflows"}},
	}
	t.Run("parent is a file", func(t *testing.T) {
		repoDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(repoDir, "workflows"), []byte("file"), 0o600); err != nil {
			t.Fatalf("create blocker: %v", err)
		}
		if _, err := New(repoDir).Push(context.Background(), app); err == nil {
			t.Fatal("Push() succeeded when workflow parent is a file")
		}
	})
	t.Run("snapshot path is a directory", func(t *testing.T) {
		repoDir := t.TempDir()
		path := filepath.Join(repoDir, "workflows", "team-a", "blocked.yaml")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("create snapshot directory: %v", err)
		}
		if _, err := New(repoDir).Push(context.Background(), app); err == nil {
			t.Fatal("Push() succeeded when snapshot path is a directory")
		}
	})
}

func TestGitCommitAndPushReadAndCancellationErrors(t *testing.T) {
	cfg := &kttmv1.GitOpsSpec{Repo: "https://example.invalid/flows", Branch: "main"}
	if _, err := gitCommitAndPush(context.Background(), t.TempDir(), filepath.Join(t.TempDir(), "missing"), "message", cfg); err == nil {
		t.Fatal("gitCommitAndPush() succeeded for missing snapshot")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := gitCommitAndPush(ctx, t.TempDir(), "", "message", cfg); err != context.Canceled {
		t.Fatalf("gitCommitAndPush(canceled) error = %v, want context.Canceled", err)
	}
	app := &kttmv1.KttmApp{
		ObjectMeta: metav1.ObjectMeta{Name: "canceled", Namespace: "team-a"},
		Spec:       kttmv1.KttmAppSpec{GitOps: cfg},
	}
	if _, err := New(t.TempDir()).Push(ctx, app); err == nil {
		t.Fatal("Push() succeeded with a canceled context")
	}
}

func TestScrubParamsBySensitiveKey(t *testing.T) {
	params := map[string]string{
		"password": "secret", "passwd": "secret", "clientSecret": "secret", "token": "secret",
		"key": "secret", "apiKey": "secret", "api_key": "secret", "auth": "secret",
		"credential": "secret", "certificate": "secret", "safe": "keep",
	}
	scrubParams(params)
	for key, value := range params {
		if key == "safe" {
			if value != "keep" {
				t.Errorf("safe parameter changed to %q", value)
			}
			continue
		}
		if value != "[REDACTED — use SecretRef]" {
			t.Errorf("%s was not scrubbed: %q", key, value)
		}
	}
}