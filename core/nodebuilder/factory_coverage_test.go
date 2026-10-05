package nodebuilder

import (
	"context"
	"strings"
	"testing"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
)

func TestBuildNodeDeterministicAndLanguageValidation(t *testing.T) {
	factory := New(Config{RegistryAddr: "registry.local:5000", Namespace: "team-a"})
	app := &kttmv1.KttmApp{}
	node := &kttmv1.WorkflowNode{ID: "transform", Type: "script/python", Script: "print('stable')"}
	first, err := factory.BuildNode(context.Background(), app, node)
	if err != nil {
		t.Fatalf("BuildNode() error = %v", err)
	}
	second, err := factory.BuildNode(context.Background(), app, node)
	if err != nil {
		t.Fatalf("repeated BuildNode() error = %v", err)
	}
	if first.ImageRef != second.ImageRef || first.Digest != second.Digest || !strings.Contains(first.ImageRef, "registry.local:5000/kttm/team-a/") {
		t.Fatalf("repeated build is not deterministic: first=%+v second=%+v", first, second)
	}
	if first.BuildDuration < 0 {
		t.Fatalf("negative build duration: %s", first.BuildDuration)
	}

	customLanguage := &kttmv1.WorkflowNode{ID: "custom", Type: "script/javascript", Language: "bash", Script: "echo ok"}
	if _, err := factory.BuildNode(context.Background(), app, customLanguage); err != nil {
		t.Fatalf("BuildNode() with explicit language error = %v", err)
	}
	if _, err := factory.BuildNode(context.Background(), app, &kttmv1.WorkflowNode{ID: "bad", Type: "script/unknown"}); err == nil {
		t.Fatal("BuildNode() accepted an unsupported inferred language")
	}
	if _, err := factory.BuildNode(context.Background(), app, &kttmv1.WorkflowNode{ID: "bad", Type: "script/python", Language: "rust"}); err == nil {
		t.Fatal("BuildNode() accepted an unsupported explicit language")
	}
}

func TestBuildAppHandlesSkipsAndErrors(t *testing.T) {
	factory := New(Config{RegistryAddr: "registry", Namespace: "ns"})
	app := &kttmv1.KttmApp{Spec: kttmv1.KttmAppSpec{WorkflowDAG: kttmv1.WorkflowDAGSpec{Nodes: []kttmv1.WorkflowNode{
		{ID: "source", Type: "connector/s3"},
		{ID: "script", Type: "script/go", Script: "package main"},
	}}}}
	results, err := factory.BuildApp(context.Background(), app)
	if err != nil {
		t.Fatalf("BuildApp() error = %v", err)
	}
	if len(results) != 1 || app.Spec.WorkflowDAG.Nodes[0].Image != "" || app.Spec.WorkflowDAG.Nodes[1].Image != results["script"].ImageRef {
		t.Fatalf("BuildApp() did not update only script nodes: %+v", app.Spec.WorkflowDAG.Nodes)
	}
	originalImage := app.Spec.WorkflowDAG.Nodes[1].Image
	secondResults, err := factory.BuildApp(context.Background(), app)
	if err != nil || secondResults["script"].ImageRef != originalImage || app.Spec.WorkflowDAG.Nodes[1].Image != originalImage {
		t.Fatalf("repeated BuildApp() changed image: results=%v err=%v", secondResults, err)
	}

	partial := &kttmv1.KttmApp{Spec: kttmv1.KttmAppSpec{WorkflowDAG: kttmv1.WorkflowDAGSpec{Nodes: []kttmv1.WorkflowNode{
		{ID: "valid", Type: "script/bash", Script: "echo ok"},
		{ID: "invalid", Type: "script/rust"},
	}}}}
	partialResults, err := factory.BuildApp(context.Background(), partial)
	if err == nil || len(partialResults) != 1 || partialResults["valid"] == nil {
		t.Fatalf("BuildApp() partial failure = (%v, %v), want prior result and error", partialResults, err)
	}
	empty, err := factory.BuildApp(context.Background(), &kttmv1.KttmApp{})
	if err != nil || len(empty) != 0 {
		t.Fatalf("BuildApp(empty) = (%v, %v), want empty success", empty, err)
	}
}

func TestDockerfileGenerationAndHelpers(t *testing.T) {
	tests := []struct {
		language string
		params   map[string]string
		contains []string
	}{
		{"python", map[string]string{"packages": "pandas"}, []string{"RUN pip install --no-cache-dir pandas", `ENTRYPOINT ["python", "script.py"]`}},
		{"javascript", map[string]string{"packages": "lodash"}, []string{"RUN npm install --omit=dev lodash", `ENTRYPOINT ["node", "script.js"]`}},
		{"bash", nil, []string{"RUN chmod +x script.sh", `ENTRYPOINT ["/bin/sh", "script.sh"]`}},
		{"r", map[string]string{"packages": "dplyr,ggplot2"}, []string{"install.packages(c('dplyr','ggplot2')", `ENTRYPOINT ["Rscript", "script.R"]`}},
		{"go", nil, []string{"go build -o /kttm/worker main.go", `ENTRYPOINT ["/kttm/worker"]`}},
		{"java", nil, []string{`ENTRYPOINT ["java", "-jar", "app.jar"]`}},
		{"python", nil, []string{`ENTRYPOINT ["python", "script.py"]`}},
		{"javascript", map[string]string{}, []string{`ENTRYPOINT ["node", "script.js"]`}},
		{"r", map[string]string{}, []string{`ENTRYPOINT ["Rscript", "script.R"]`}},
	}
	for _, tt := range tests {
		t.Run(tt.language+"/"+strings.Join(tt.contains, "-"), func(t *testing.T) {
			generated := generateDockerfile("base:latest", tt.language, "script", tt.params)
			for _, expected := range tt.contains {
				if !strings.Contains(generated, expected) {
					t.Errorf("Dockerfile for %s missing %q:\n%s", tt.language, expected, generated)
				}
			}
		})
	}
	if isScriptNode("script/python") != true || isScriptNode("connector/s3") {
		t.Fatal("isScriptNode() returned an unexpected result")
	}
	if inferLanguage("script/javascript") != "javascript" || inferLanguage("script") != "" {
		t.Fatal("inferLanguage() did not handle slash and no-slash values")
	}
	if scriptHash("same", "go") != scriptHash("same", "go") || scriptHash("same", "go") == scriptHash("different", "go") {
		t.Fatal("scriptHash() is not content deterministic")
	}
}