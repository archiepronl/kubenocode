package engine

import (
	"testing"

	"github.com/kubeworkflow/flowengine/core/api/v1alpha1"
)

func TestArgoCompiler_Compile(t *testing.T) {
	compiler := NewArgoCompiler()

	app := &v1alpha1.FullStackApplication{
		Spec: v1alpha1.FullStackApplicationSpec{
			WorkflowDAG: []v1alpha1.WorkflowNode{
				{
					ID:   "node1",
					Type: "script/python",
				},
				{
					ID:      "node2",
					Type:    "script/bash",
					Outputs: []string{"node1"},
				},
			},
		},
	}

	workflow, err := compiler.Compile(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workflow == nil {
		t.Fatalf("expected workflow to be non-nil")
	}
}
