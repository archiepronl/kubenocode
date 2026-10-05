package linter

import (
	"strings"
	"testing"

	"github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestLintResultFormattingAndFiltering(t *testing.T) {
	findings := []LintFinding{
		{Severity: SeverityError, Code: "BROKEN", NodeID: "node-a", Message: "bad edge"},
		{Severity: SeverityWarning, Code: "NOTICE", NodeID: "node-b", Message: "no limits"},
	}
	if got := findings[0].Error(); !strings.Contains(got, "[error][BROKEN]") || !strings.Contains(got, "bad edge") {
		t.Fatalf("Error() = %q", got)
	}
	result := &LintResult{Findings: findings}
	if !result.HasErrors() || len(result.Errors()) != 1 || len(result.ErrorStrings()) != 1 {
		t.Fatalf("error filtering failed: %+v", result)
	}
	if got := result.ErrorStrings()[0]; !strings.Contains(got, "node=\"node-a\"") {
		t.Fatalf("ErrorStrings() = %q", got)
	}
	clean := &LintResult{Findings: []LintFinding{{Severity: SeverityWarning}}}
	if clean.HasErrors() || len(clean.Errors()) != 0 || len(clean.ErrorStrings()) != 0 {
		t.Fatalf("warning-only result has errors: %+v", clean)
	}
}

func TestLintNodeTypeAndResourceBranches(t *testing.T) {
	dag := []v1alpha1.WorkflowNode{
		{ID: "custom", Type: "custom/runtime"},
		{ID: "script-missing", Type: "script/python"},
		{ID: "script-image", Type: "script/python", Image: "example/python:v1"},
		{ID: "bounded", Type: "transform/map", Resources: &corev1.ResourceRequirements{Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")}}},
	}
	result := New().Lint(dag)
	codes := map[string]int{}
	for _, finding := range result.Findings {
		codes[finding.Code]++
	}
	if codes["UNKNOWN_NODE_TYPE"] != 1 || codes["MISSING_SCRIPT_IMAGE"] != 1 || codes["NO_RESOURCE_LIMITS"] != 3 {
		t.Fatalf("unexpected findings: %+v", result.Findings)
	}
}