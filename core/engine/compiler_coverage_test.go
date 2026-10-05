package engine

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	v1alpha1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestArgoCompilerNameAndEmptyDAG(t *testing.T) {
	compiler := NewArgoCompiler()
	if got := compiler.Name(); got != "argo" {
		t.Fatalf("Name() = %q, want argo", got)
	}
	if _, err := compiler.Compile(&v1alpha1.FullStackApplication{}); err == nil || !strings.Contains(err.Error(), "at least one node") {
		t.Fatalf("Compile(empty DAG) error = %v, want empty DAG error", err)
	}
}

func TestArgoCompilerMarshalFailure(t *testing.T) {
	compiler := NewArgoCompiler()
	compiler.marshalWorkflow = func(ArgoWorkflow) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}
	app := &v1alpha1.FullStackApplication{
		Spec: v1alpha1.FullStackApplicationSpec{WorkflowDAG: []v1alpha1.WorkflowNode{{ID: "node"}}},
	}
	if _, err := compiler.Compile(app); err == nil || !strings.Contains(err.Error(), "marshalling Argo Workflow CRD: marshal failed") {
		t.Fatalf("Compile() error = %v, want wrapped marshal error", err)
	}
}

func TestArgoCompilerZeroValueUsesDefaultSerializer(t *testing.T) {
	compiler := &ArgoCompiler{}
	app := &v1alpha1.FullStackApplication{
		ObjectMeta: metav1.ObjectMeta{Name: "zero-value"},
		Spec:       v1alpha1.FullStackApplicationSpec{WorkflowDAG: []v1alpha1.WorkflowNode{{ID: "node"}}},
	}
	if _, err := compiler.Compile(app); err != nil {
		t.Fatalf("zero-value Compile() error = %v", err)
	}
}

func TestArgoCompilerCustomOptionsAndTemplate(t *testing.T) {
	ttl := int32(45)
	app := &v1alpha1.FullStackApplication{
		TypeMeta:   metav1.TypeMeta{APIVersion: "flowengine.io/v1alpha1", Kind: "FullStackApplication"},
		ObjectMeta: metav1.ObjectMeta{Name: "Orders_App", Namespace: "team-a", UID: "app-uid"},
		Spec: v1alpha1.FullStackApplicationSpec{
			Execution: &v1alpha1.ExecutionConfig{ServiceAccountName: "custom-runner", TTLSecondsAfterFinished: &ttl},
			WorkflowDAG: []v1alpha1.WorkflowNode{
				{ID: "Source", Type: "connector/s3", Outputs: []string{"Sink"}},
				{
					ID: "Sink", Type: "script/python", Image: "example/worker:v2",
					Params: map[string]string{"bucket": "orders"}, SecretRef: "database-creds",
					Script: "echo done", Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m")},
					},
				},
			},
		},
	}

	manifest, err := NewArgoCompiler().Compile(app)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if manifest.ResourceName != "fsa-Orders_App" || manifest.Kind != "Workflow" || manifest.APIVersion != "argoproj.io/v1alpha1" {
		t.Fatalf("unexpected manifest metadata: %+v", manifest)
	}
	var workflow ArgoWorkflow
	if err := json.Unmarshal(manifest.Raw, &workflow); err != nil {
		t.Fatalf("unmarshal workflow: %v", err)
	}
	if workflow.Spec.ServiceAccountName != "custom-runner" || workflow.Spec.TTLStrategy == nil ||
		workflow.Spec.TTLStrategy.SecondsAfterCompletion == nil || *workflow.Spec.TTLStrategy.SecondsAfterCompletion != ttl {
		t.Fatalf("execution options were not applied: %+v", workflow.Spec)
	}
	if workflow.Namespace != "team-a" || len(workflow.OwnerReferences) != 1 || workflow.OwnerReferences[0].UID != "app-uid" {
		t.Fatalf("workflow ownership metadata is incomplete: %+v", workflow.ObjectMeta)
	}
	if len(workflow.Spec.Templates) != 3 || workflow.Spec.Templates[0].Name != "main" {
		t.Fatalf("unexpected templates: %+v", workflow.Spec.Templates)
	}
	if tasks := workflow.Spec.Templates[0].DAG.Tasks; len(tasks) != 2 || len(tasks[1].Dependencies) != 1 || tasks[1].Dependencies[0] != "source" {
		t.Fatalf("DAG dependencies were not compiled: %+v", tasks)
	}
	container := workflow.Spec.Templates[2].Container
	if container == nil || container.Image != "example/worker:v2" || len(container.Command) != 3 || container.Command[2] != "echo done" {
		t.Fatalf("custom container settings were not applied: %+v", container)
	}
	if len(container.EnvFrom) != 1 || container.EnvFrom[0].SecretRef == nil || container.EnvFrom[0].SecretRef.Name != "database-creds" {
		t.Fatalf("secret reference was not forwarded: %+v", container.EnvFrom)
	}
	if container.Resources.Requests.Cpu().String() != "250m" {
		t.Fatalf("custom resources were not applied: %+v", container.Resources)
	}
	if len(container.Env) != 4 {
		t.Fatalf("expected one param and three routing environment variables, got %+v", container.Env)
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "node-1", want: "node-1"},
		{input: "Node_A!", want: "node-a-"},
		{input: "", want: ""},
	}
	for _, tt := range tests {
		if got := sanitizeName(tt.input); got != tt.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}