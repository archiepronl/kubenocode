package engine

import (
	"testing"

	v1alpha1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMapCRDToExecutionGraphEmptyInput(t *testing.T) {
	if got := MapCRDToExecutionGraph(nil); got != nil {
		t.Fatalf("MapCRDToExecutionGraph(nil) = %v, want nil", got)
	}
	if got := MapCRDToExecutionGraph(&v1alpha1.FullStackApplication{}); got != nil {
		t.Fatalf("empty workflow graph = %v, want nil", got)
	}
}

func TestMapCRDToExecutionGraphCopiesNodes(t *testing.T) {
	app := &v1alpha1.FullStackApplication{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "apps"},
		Spec: v1alpha1.FullStackApplicationSpec{WorkflowDAG: []v1alpha1.WorkflowNode{{
			ID: "source", Type: "connector/s3", Label: "Input", GroupID: "group-a",
			PackagingMode: "sidecar", Language: "go", Params: map[string]string{"bucket": "orders"},
			SecretRef: "aws-creds", Image: "example/worker:v1", Script: "run", Outputs: []string{"sink"},
			MimeTypeHint: "application/json",
		}}},
	}

	graph := MapCRDToExecutionGraph(app)
	if graph == nil {
		t.Fatal("MapCRDToExecutionGraph() returned nil")
	}
	if graph.ID != "orders" || graph.EntryNode != "source" || len(graph.Nodes) != 1 || graph.CreatedAt.IsZero() {
		t.Fatalf("unexpected graph metadata: %+v", graph)
	}
	node := graph.Nodes["source"]
	if node == nil {
		t.Fatal("mapped graph is missing source node")
	}
	if node.Type != "connector/s3" || node.Label != "Input" || node.GroupID != "group-a" ||
		node.PackagingMode != "sidecar" || node.Language != "go" || node.SecretRef != "aws-creds" ||
		node.Image != "example/worker:v1" || node.Script != "run" || node.MimeTypeHint != "application/json" {
		t.Fatalf("mapped node fields are incomplete: %+v", node)
	}
	if node.Params["bucket"] != "orders" || len(node.Outputs) != 1 || node.Outputs[0] != "sink" {
		t.Fatalf("mapped node data is incomplete: %+v", node)
	}

	app.Spec.WorkflowDAG[0].Params["bucket"] = "changed"
	app.Spec.WorkflowDAG[0].Outputs[0] = "changed"
	if node.Params["bucket"] != "orders" || node.Outputs[0] != "sink" {
		t.Fatal("mapped node aliases mutable CRD maps or slices")
	}
}
