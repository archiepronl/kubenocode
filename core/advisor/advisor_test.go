package advisor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRuleAdvisor_Analyze_MemoryCritical(t *testing.T) {
	ra := NewRuleAdvisor()
	m := NodeMetrics{
		NodeID:     "node1",
		MemPercent: 95.0,
		MemMB:      950,
		MemLimitMB: 1000,
	}

	advice := ra.Analyze(m)
	if len(advice) != 1 {
		t.Fatalf("expected 1 advice, got %d", len(advice))
	}
	if advice[0].Severity != AdviceCritical {
		t.Errorf("expected critical severity, got %s", advice[0].Severity)
	}
	if advice[0].Category != "memory" {
		t.Errorf("expected memory category, got %s", advice[0].Category)
	}
}

func TestRuleAdvisor_Analyze_MemoryWarning(t *testing.T) {
	ra := NewRuleAdvisor()
	m := NodeMetrics{
		NodeID:     "node1",
		MemPercent: 80.0,
		MemMB:      800,
		MemLimitMB: 1000,
	}

	advice := ra.Analyze(m)
	if len(advice) != 1 {
		t.Fatalf("expected 1 advice, got %d", len(advice))
	}
	if advice[0].Severity != AdviceWarning {
		t.Errorf("expected warning severity, got %s", advice[0].Severity)
	}
}

func TestRuleAdvisor_Analyze_CPUThrottling(t *testing.T) {
	ra := NewRuleAdvisor()
	m := NodeMetrics{
		NodeID:     "node1",
		CPUPercent: 85.0,
	}

	advice := ra.Analyze(m)
	if len(advice) != 1 {
		t.Fatalf("expected 1 advice, got %d", len(advice))
	}
	if advice[0].Severity != AdviceWarning {
		t.Errorf("expected warning severity, got %s", advice[0].Severity)
	}
	if advice[0].Category != "cpu" {
		t.Errorf("expected cpu category, got %s", advice[0].Category)
	}
}

func TestRuleAdvisor_Analyze_ErrorRateCritical(t *testing.T) {
	ra := NewRuleAdvisor()
	m := NodeMetrics{
		NodeID:    "node1",
		ErrorRate: 25.0,
	}

	advice := ra.Analyze(m)
	if len(advice) != 1 {
		t.Fatalf("expected 1 advice, got %d", len(advice))
	}
	if advice[0].Severity != AdviceCritical {
		t.Errorf("expected critical severity, got %s", advice[0].Severity)
	}
	if advice[0].Category != "reliability" {
		t.Errorf("expected reliability category, got %s", advice[0].Category)
	}
}

func TestRuleAdvisor_Analyze_ErrorRateWarning(t *testing.T) {
	ra := NewRuleAdvisor()
	m := NodeMetrics{
		NodeID:    "node1",
		ErrorRate: 10.0,
	}

	advice := ra.Analyze(m)
	if len(advice) != 1 {
		t.Fatalf("expected 1 advice, got %d", len(advice))
	}
	if advice[0].Severity != AdviceWarning {
		t.Errorf("expected warning severity, got %s", advice[0].Severity)
	}
}

func TestRuleAdvisor_Analyze_QueueDepthWarning(t *testing.T) {
	ra := NewRuleAdvisor()
	m := NodeMetrics{
		NodeID:     "node1",
		QueueDepth: 6000,
	}

	advice := ra.Analyze(m)
	if len(advice) != 1 {
		t.Fatalf("expected 1 advice, got %d", len(advice))
	}
	if advice[0].Severity != AdviceWarning {
		t.Errorf("expected warning severity, got %s", advice[0].Severity)
	}
	if advice[0].Category != "throughput" {
		t.Errorf("expected throughput category, got %s", advice[0].Category)
	}
}

func TestRuleAdvisor_Analyze_P99Duration(t *testing.T) {
	ra := NewRuleAdvisor()
	m := NodeMetrics{
		NodeID:        "node1",
		P99DurationMs: 40000,
	}

	advice := ra.Analyze(m)
	if len(advice) != 1 {
		t.Fatalf("expected 1 advice, got %d", len(advice))
	}
	if advice[0].Severity != AdviceInfo {
		t.Errorf("expected info severity, got %s", advice[0].Severity)
	}
	if advice[0].Category != "performance" {
		t.Errorf("expected performance category, got %s", advice[0].Category)
	}
}

func TestBuildLLMPrompt(t *testing.T) {
	m := NodeMetrics{
		NodeID:        "node1",
		WorkflowID:    "wf1",
		CPUPercent:    50.0,
		MemMB:         500.0,
		MemLimitMB:    1000.0,
		MemPercent:    50.0,
		DurationMs:    1000,
		P99DurationMs: 2000,
		ErrorRate:     1.0,
		QueueDepth:    10,
		PayloadSizeMB: 5.0,
	}

	prompt := BuildLLMPrompt(m, []Advice{})
	if !strings.Contains(prompt, "node1") {
		t.Errorf("expected prompt to contain node ID")
	}
}

func TestAdvisorLoop(t *testing.T) {
	loop := NewAdvisorLoop(NewRuleAdvisor())
	if loop == nil {
		t.Fatalf("expected non-nil loop")
	}
	ctx, cancel := context.WithCancel(context.Background())
	stop := time.AfterFunc(10*time.Millisecond, cancel)
	loop.Run(ctx, "team-a", "orders", time.Millisecond)
	stop.Stop()
}
