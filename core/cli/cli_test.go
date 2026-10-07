package cli

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failAfterWriter struct {
	writes int
	failAt int
}

func (w *failAfterWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.writes >= w.failAt {
		return 0, errors.New("write failed")
	}
	return len(data), nil
}

func TestExecuteDispatchAndUsage(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{name: "no arguments", want: "USAGE:"},
		{name: "help", args: []string{"help"}, want: "COMMANDS:"},
		{name: "short help", args: []string{"-h"}, want: "FLAGS (run):"},
		{name: "version", args: []string{"version"}, want: "nextkube dev"},
		{name: "unknown command", args: []string{"mystery"}, wantErr: `unknown command "mystery"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCaptured(t, tt.args...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Execute() error = %v, want %q", err, tt.wantErr)
				}
			} else if err != nil || !strings.Contains(output, tt.want) {
				t.Fatalf("Execute() output/error = (%q, %v), want output containing %q", output, err, tt.want)
			}
		})
	}
}

func TestRunAndValidateCommands(t *testing.T) {
	workflow := filepath.Join(t.TempDir(), "workflow.yaml")
	content := "name: orders\n"
	if err := os.WriteFile(workflow, []byte(content), 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := runCmd(nil); err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("runCmd(no args) error = %v", err)
	}
	if err := runCmd([]string{"--unknown"}); err == nil {
		t.Fatal("runCmd() accepted an unknown flag")
	}
	if err := runCmd([]string{filepath.Join(t.TempDir(), "missing.yaml")}); err == nil || !strings.Contains(err.Error(), "reading workflow file") {
		t.Fatalf("runCmd(missing file) error = %v", err)
	}
	output, err := captureStdout(func() error {
		return runCmd([]string{"--dry-run", "--runtime", "podman", "--namespace", "team-a", workflow})
	})
	if err != nil || !strings.Contains(output, "[DRY RUN]") || !strings.Contains(output, content) || !strings.Contains(output, "Runtime:   podman") {
		t.Fatalf("runCmd(dry run) = (%q, %v)", output, err)
	}
	output, err = captureStdout(func() error { return runCmd([]string{workflow}) })
	if err != nil || !strings.Contains(output, "Workflow complete") {
		t.Fatalf("runCmd(normal) = (%q, %v)", output, err)
	}
	if err := validateCmd(nil); err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("validateCmd(no args) error = %v", err)
	}
	if err := validateCmd([]string{"--unknown"}); err == nil {
		t.Fatal("validateCmd() accepted an unknown flag")
	}
	if err := validateCmd([]string{filepath.Join(t.TempDir(), "missing.yaml")}); err == nil {
		t.Fatal("validateCmd() succeeded for missing workflow")
	}
	output, err = captureStdout(func() error { return validateCmd([]string{workflow}) })
	if err != nil || !strings.Contains(output, "is valid") {
		t.Fatalf("validateCmd(text) = (%q, %v)", output, err)
	}
	output, err = captureStdout(func() error { return validateCmd([]string{"--json", workflow}) })
	if err != nil {
		t.Fatalf("validateCmd(json) error = %v", err)
	}
	var report struct {
		File  string `json:"file"`
		Valid bool   `json:"valid"`
	}
	jsonStart := strings.IndexByte(output, '{')
	if jsonStart < 0 {
		t.Fatalf("validate output has no JSON object: %q", output)
	}
	if err := json.Unmarshal([]byte(output[jsonStart:]), &report); err != nil || report.File != workflow || !report.Valid {
		t.Fatalf("validate JSON = (%+v, %v), raw %q", report, err, output)
	}
	if _, err := executeCaptured(t, "run", "--dry-run", workflow); err != nil {
		t.Fatalf("Execute(run) error = %v", err)
	}
	if _, err := executeCaptured(t, "validate", workflow); err != nil {
		t.Fatalf("Execute(validate) error = %v", err)
	}
}

func TestExportImportAndVersionCommands(t *testing.T) {
	defaultArtifact := filepath.Join("missing-bundle.tar.gz")
	if _, err := os.Stat(defaultArtifact); err == nil {
		t.Cleanup(func() { _ = os.Remove(defaultArtifact) })
	}
	workflow := filepath.Join(t.TempDir(), "pipeline.yaml")
	if err := os.WriteFile(workflow, []byte("steps: []\n"), 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := exportCmd(nil); err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("exportCmd(no args) error = %v", err)
	}
	if err := exportCmd([]string{"--unknown"}); err == nil {
		t.Fatal("exportCmd() accepted an unknown flag")
	}
	missingOutput := filepath.Join(t.TempDir(), "missing-bundle.tar.gz")
	if err := exportCmd([]string{"-o", missingOutput, filepath.Join(t.TempDir(), "missing.yaml")}); err == nil || !strings.Contains(err.Error(), "reading workflow file") {
		t.Fatalf("exportCmd(missing file) error = %v", err)
	}
	if err := exportCmd([]string{"-o", t.TempDir(), workflow}); err == nil || !strings.Contains(err.Error(), "creating output file") {
		t.Fatalf("exportCmd(directory output) error = %v", err)
	}
	manifestError := errors.New("manifest write failed")
	if err := exportCmdWithManifest([]string{"-o", filepath.Join(t.TempDir(), "failed.tar.gz"), workflow}, func(*tar.Writer, string, string) error {
		return manifestError
	}, addStringToTar); !errors.Is(err, manifestError) || !strings.Contains(err.Error(), "adding bundle manifest") {
		t.Fatalf("exportCmdWithManifest() error = %v, want wrapped manifest error", err)
	}

	stringError := errors.New("string write failed")
	if err := exportCmdWithManifest([]string{"-o", filepath.Join(t.TempDir(), "failed-str.tar.gz"), workflow}, addStringToTar, func(*tar.Writer, string, string) error {
		return stringError
	}); !errors.Is(err, stringError) || !strings.Contains(err.Error(), "adding workflow to bundle") {
		t.Fatalf("exportCmdWithManifest() string error = %v, want wrapped string error", err)
	}

	archivePath := filepath.Join(t.TempDir(), "bundle.tar.gz")
	output, err := captureStdout(func() error { return exportCmd([]string{"--include-images", "-o", archivePath, workflow}) })
	if err != nil || !strings.Contains(output, "Including container images") {
		t.Fatalf("exportCmd() = (%q, %v)", output, err)
	}
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	defaultDir := t.TempDir()
	if err := os.Chdir(defaultDir); err != nil {
		t.Fatalf("change to temporary directory: %v", err)
	}
	if err := exportCmd([]string{workflow}); err != nil {
		t.Fatalf("exportCmd(default output) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(defaultDir, "pipeline-bundle.tar.gz")); err != nil {
		t.Fatalf("default output bundle missing: %v", err)
	}
	if err := os.Chdir(previousDir); err != nil {
		t.Fatalf("restore working directory: %v", err)
	}
	if _, err := executeCaptured(t, "export", "-o", filepath.Join(t.TempDir(), "via-execute.tar.gz"), workflow); err != nil {
		t.Fatalf("Execute(export) error = %v", err)
	}
	entries := readBundle(t, archivePath)
	if entries["manifest.yaml"] != "steps: []\n" {
		t.Fatalf("bundle manifest = %q", entries["manifest.yaml"])
	}
	var bundleInfo map[string]interface{}
	if err := json.Unmarshal([]byte(entries["bundle-info.json"]), &bundleInfo); err != nil || bundleInfo["includesImages"] != true {
		t.Fatalf("bundle info = (%v, %v)", bundleInfo, err)
	}
	if err := importCmd(nil); err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("importCmd(no args) error = %v", err)
	}
	if err := importCmd([]string{"--unknown"}); err == nil {
		t.Fatal("importCmd() accepted an unknown flag")
	}
	output, err = captureStdout(func() error {
		return importCmd([]string{"--registry", "registry.local:5000", "--apply=false", archivePath})
	})
	if err != nil || !strings.Contains(output, "registry.local:5000") || strings.Contains(output, "Applying workflow manifest") {
		t.Fatalf("importCmd(flags) = (%q, %v)", output, err)
	}
	output, err = captureStdout(func() error { return importCmd([]string{archivePath}) })
	if err != nil || !strings.Contains(output, "Applying workflow manifest") {
		t.Fatalf("importCmd(defaults) = (%q, %v)", output, err)
	}
	if _, err := executeCaptured(t, "import", "--apply=false", archivePath); err != nil {
		t.Fatalf("Execute(import) error = %v", err)
	}
	output, err = captureStdout(versionCmd)
	if err != nil || !strings.Contains(output, "git commit: unknown") || !strings.Contains(output, "built:      unknown") {
		t.Fatalf("versionCmd() = (%q, %v)", output, err)
	}
}

func TestTarHelpersErrorAndSuccessPaths(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(file, []byte("file data"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := addFileToTar(tar.NewWriter(io.Discard), filepath.Join(t.TempDir(), "missing"), "missing"); err == nil {
		t.Fatal("addFileToTar() succeeded for missing input")
	}
	statError := errors.New("stat failed")
	if err := addFileToTarWithStat(tar.NewWriter(io.Discard), file, "data.txt", func(*os.File) (os.FileInfo, error) {
		return nil, statError
	}); !errors.Is(err, statError) {
		t.Fatalf("addFileToTarWithStat() error = %v, want stat error", err)
	}
	if err := addFileToTar(tar.NewWriter(&failAfterWriter{failAt: 1}), file, "data.txt"); err == nil {
		t.Fatal("addFileToTar() succeeded when tar header write failed")
	}
	largeFile := filepath.Join(t.TempDir(), "large.bin")
	if err := os.WriteFile(largeFile, []byte(strings.Repeat("x", 2048)), 0o600); err != nil {
		t.Fatalf("write large input: %v", err)
	}
	if err := addFileToTar(tar.NewWriter(&failAfterWriter{failAt: 2}), largeFile, "large.bin"); err == nil {
		t.Fatal("addFileToTar() succeeded when copy failed")
	}
	closed := tar.NewWriter(io.Discard)
	if err := closed.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := addStringToTar(closed, "data", "data.json"); err == nil {
		t.Fatal("addStringToTar() succeeded for closed writer")
	}
	if err := addFileToTar(tar.NewWriter(io.Discard), file, "data.txt"); err != nil {
		t.Fatalf("addFileToTar() error = %v", err)
	}
}

func executeCaptured(t *testing.T, args ...string) (string, error) {
	t.Helper()
	previous := os.Args
	os.Args = append([]string{"nextkube"}, args...)
	t.Cleanup(func() { os.Args = previous })
	return captureStdout(Execute)
}

func captureStdout(run func() error) (string, error) {
	previous := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = writer
	runErr := run()
	_ = writer.Close()
	os.Stdout = previous
	output, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if runErr != nil {
		return string(output), runErr
	}
	return string(output), readErr
}

func readBundle(t *testing.T, path string) map[string]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("open gzip: %v", err)
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	entries := make(map[string]string)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar header: %v", err)
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("read tar entry: %v", err)
		}
		entries[header.Name] = string(data)
	}
	return entries
}

func TestScrubSecrets(t *testing.T) {
	input := `
apiVersion: flowengine.io/v1alpha1
kind: KttmApp
spec:
  nodes:
    - id: node1
      params:
        password: my-secret-password
        token: some-jwt-token
        username: admin
        aws_secret_key: abc123def
`
	expected := `
apiVersion: flowengine.io/v1alpha1
kind: KttmApp
spec:
  nodes:
    - id: node1
      params:
        password: "***SCRUBBED***"
        token: "***SCRUBBED***"
        username: admin
        aws_secret_key: "***SCRUBBED***"
`
	result := scrubSecrets([]byte(input))
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}
