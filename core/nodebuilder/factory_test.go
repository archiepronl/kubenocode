package nodebuilder

import (
	"context"
	"testing"

	"github.com/kubeworkflow/flowengine/core/api/v1alpha1"
)

func TestNodeBuilderFactory(t *testing.T) {
	cfg := Config{
		RegistryAddr: "localhost:5000",
		Namespace:    "default",
	}
	factory := New(cfg)
	app := &v1alpha1.KttmApp{}

	t.Run("Script Node Python", func(t *testing.T) {
		node := &v1alpha1.WorkflowNode{
			ID:     "node1",
			Type:   "script/python",
			Script: "print('hello world')",
		}
		res, err := factory.BuildNode(context.Background(), app, node)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Fatalf("expected build result for script node")
		}
	})

	t.Run("Non-Script Node", func(t *testing.T) {
		node := &v1alpha1.WorkflowNode{
			ID:   "node2",
			Type: "connector/s3",
		}
		res, err := factory.BuildNode(context.Background(), app, node)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != nil {
			t.Fatalf("expected nil build result for non-script node")
		}
	})

	t.Run("Unknown Language", func(t *testing.T) {
		node := &v1alpha1.WorkflowNode{
			ID:   "node3",
			Type: "script/unknownlang",
		}
		res, err := factory.BuildNode(context.Background(), app, node)
		if err == nil {
			t.Fatalf("expected error for unknown language")
		}
		if res != nil {
			t.Fatalf("expected nil result")
		}
	})

	t.Run("BuildApp", func(t *testing.T) {
		a := &v1alpha1.KttmApp{
			Spec: v1alpha1.KttmAppSpec{
				WorkflowDAG: v1alpha1.WorkflowDAGSpec{
					Nodes: []v1alpha1.WorkflowNode{
						{ID: "n1", Type: "script/python", Script: "print()"},
					},
				},
			},
		}
		results, err := factory.BuildApp(context.Background(), a)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result")
		}
	})
}
