package linter

import (
	"testing"

	"github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestGraphLinter_Lint(t *testing.T) {
	linter := New()

	t.Run("Empty DAG", func(t *testing.T) {
		result := linter.Lint([]v1alpha1.WorkflowNode{})
		if result.HasErrors() {
			t.Errorf("expected no errors for empty DAG, got %v", result.ErrorStrings())
		}
	})

	t.Run("Valid simple DAG", func(t *testing.T) {
		dag := []v1alpha1.WorkflowNode{
			{
				ID:        "nodeA",
				Type:      "trigger/webhook",
				Outputs:   []string{"nodeB"},
				Resources: &corev1.ResourceRequirements{},
			},
			{
				ID:        "nodeB",
				Type:      "script/python",
				Script:    "print('hello')",
				Resources: &corev1.ResourceRequirements{},
			},
		}
		result := linter.Lint(dag)
		if result.HasErrors() {
			t.Errorf("expected valid DAG, got errors: %v", result.ErrorStrings())
		}
	})

	t.Run("Cycle detection", func(t *testing.T) {
		dag := []v1alpha1.WorkflowNode{
			{ID: "nodeA", Outputs: []string{"nodeB"}},
			{ID: "nodeB", Outputs: []string{"nodeA"}},
		}
		result := linter.Lint(dag)
		if !result.HasErrors() {
			t.Errorf("expected cycle detection error")
		}
	})

	t.Run("Dangling edge", func(t *testing.T) {
		dag := []v1alpha1.WorkflowNode{
			{ID: "nodeA", Outputs: []string{"unknown-node"}},
		}
		result := linter.Lint(dag)
		if !result.HasErrors() {
			t.Errorf("expected dangling edge error")
		}
	})

	t.Run("Missing secret ref", func(t *testing.T) {
		dag := []v1alpha1.WorkflowNode{
			{ID: "nodeS3", Type: "connector/s3"}, // missing SecretRef
		}
		result := linter.Lint(dag)
		if !result.HasErrors() {
			t.Errorf("expected missing secret ref error")
		}
	})

	t.Run("Isolated node warning", func(t *testing.T) {
		dag := []v1alpha1.WorkflowNode{
			{ID: "nodeA", Outputs: []string{"nodeB"}},
			{ID: "nodeB"},
			{ID: "nodeIsolated"}, // no incoming, no outgoing
		}
		result := linter.Lint(dag)
		if result.HasErrors() {
			t.Errorf("expected isolated node to only produce warning, not error")
		}
		hasWarning := false
		for _, f := range result.Findings {
			if f.Code == "ISOLATED_NODE" {
				hasWarning = true
			}
		}
		if !hasWarning {
			t.Errorf("expected isolated node warning")
		}
	})
}
