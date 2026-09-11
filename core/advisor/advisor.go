// Package advisor implements the AI Advisor engine (Section 4 of the spec).
//
// The AI Advisor operates in two phases:
//
// Phase 1 — Pre-Deploy Static Linter (synchronous, see core/linter/graph.go):
//   - Runs before deployment, blocks on errors
//   - Implemented in the GraphLinter
//
// Phase 2 — Runtime Telemetry Advisor (this package, asynchronous):
//   - Polls Prometheus for live container metrics every 60s
//   - When thresholds are breached, constructs a structured prompt
//   - Sends the prompt to the configured LLM (Gemini or local rule engine)
//   - Publishes advice to the UI via NATS (flowengine.{ns}.advisor.{nodeId}.advice)
//
// In air-gapped environments (no LLM API access), the advisor falls back to
// a deterministic rule engine that provides the same advice categories
// without requiring an external API call (NFR hybrid design decision).
package advisor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────
//  Metric snapshot
// ─────────────────────────────────────────────

// NodeMetrics represents a point-in-time snapshot of a workflow node's resource usage.
// Populated by querying Prometheus (or ClickHouse for historical analysis).
type NodeMetrics struct {
	NodeID       string    `json:"nodeId"`
	WorkflowID   string    `json:"workflowId"`
	Namespace    string    `json:"namespace"`
	CPUPercent   float64   `json:"cpuPercent"`   // % of CPU limit used
	MemMB        float64   `json:"memMb"`        // Memory used in MB
	MemLimitMB   float64   `json:"memLimitMb"`   // Memory limit in MB
	MemPercent   float64   `json:"memPercent"`   // % of memory limit used
	DurationMs   int64     `json:"durationMs"`   // Average step duration (last 5m)
	P99DurationMs int64    `json:"p99DurationMs"` // p99 step duration (last 5m)
	ErrorRate    float64   `json:"errorRate"`    // % of executions that errored
	QueueDepth   int64     `json:"queueDepth"`   // NATS queue depth (if applicable)
	PayloadSizeMB float64  `json:"payloadSizeMb"` // Average payload size
	SampledAt    time.Time `json:"sampledAt"`
}

// ─────────────────────────────────────────────
//  Advice types
// ─────────────────────────────────────────────

// AdviceSeverity classifies the urgency of an advisor recommendation.
type AdviceSeverity string

const (
	AdviceCritical AdviceSeverity = "critical" // Immediate action required (OOM, crash loop)
	AdviceWarning  AdviceSeverity = "warning"  // Action recommended (nearing limits)
	AdviceInfo     AdviceSeverity = "info"     // Optimization opportunity
)

// Advice is a single recommendation produced by the AI Advisor for a specific node.
type Advice struct {
	NodeID    string         `json:"nodeId"`
	Severity  AdviceSeverity `json:"severity"`
	Category  string         `json:"category"` // "memory", "cpu", "throughput", "logic", "cost"
	Title     string         `json:"title"`
	Message   string         `json:"message"`   // Plain-English recommendation shown in the UI
	Action    string         `json:"action"`    // Specific suggested change
	Timestamp time.Time      `json:"timestamp"`
}

// AdvisorReport groups all advice for a single analysis run.
type AdvisorReport struct {
	WorkflowID string    `json:"workflowId"`
	Namespace  string    `json:"namespace"`
	Advice     []Advice  `json:"advice"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// ─────────────────────────────────────────────
//  Alert thresholds
// ─────────────────────────────────────────────

const (
	memCriticalPct    = 90.0  // % of memory limit → critical alert
	memWarningPct     = 75.0  // % of memory limit → warning alert
	cpuThrottlePct    = 80.0  // % of CPU limit throttled → warning
	errorRateWarning  = 5.0   // % error rate → warning
	errorRateCritical = 20.0  // % error rate → critical
	queueDepthWarning = 5000  // NATS queue depth → warning
	p99DurationWarnMs = 30000 // p99 > 30s → warning (slow step)
)

// ─────────────────────────────────────────────
//  Rule-based advisor (works offline / air-gapped)
// ─────────────────────────────────────────────

// RuleAdvisor implements the deterministic rule-based advisor.
// It generates advice without any LLM API call and works in air-gapped environments.
type RuleAdvisor struct{}

// NewRuleAdvisor returns a ready-to-use RuleAdvisor.
func NewRuleAdvisor() *RuleAdvisor { return &RuleAdvisor{} }

// Analyze evaluates a NodeMetrics snapshot against thresholds and returns advice.
func (a *RuleAdvisor) Analyze(m NodeMetrics) []Advice {
	var advice []Advice

	// ── Memory utilization ────────────────────────────────────────────────────
	if m.MemPercent >= memCriticalPct {
		advice = append(advice, Advice{
			NodeID:   m.NodeID,
			Severity: AdviceCritical,
			Category: "memory",
			Title:    "Critical Memory Pressure",
			Message: fmt.Sprintf(
				"Node %q is using %.0f%% of its memory limit (%.0f/%.0f MB). "+
					"This node is at risk of OOMKill, which will fail the workflow.",
				m.NodeID, m.MemPercent, m.MemMB, m.MemLimitMB,
			),
			Action: fmt.Sprintf(
				"Increase the memory limit to at least %.0f MB, or enable micro-batch processing "+
					"to process data in smaller chunks instead of loading the full payload into memory.",
				m.MemLimitMB*1.5,
			),
		})
	} else if m.MemPercent >= memWarningPct {
		advice = append(advice, Advice{
			NodeID:   m.NodeID,
			Severity: AdviceWarning,
			Category: "memory",
			Title:    "High Memory Usage",
			Message: fmt.Sprintf(
				"Node %q is using %.0f%% of its memory limit (%.0f/%.0f MB). "+
					"Consider increasing the limit before this reaches the critical threshold.",
				m.NodeID, m.MemPercent, m.MemMB, m.MemLimitMB,
			),
			Action: "Set resources.limits.memory to at least " +
				fmt.Sprintf("%.0fMi", m.MemLimitMB*1.25) + " in the WorkflowNode spec.",
		})
	}

	// ── CPU throttling ────────────────────────────────────────────────────────
	if m.CPUPercent >= cpuThrottlePct {
		advice = append(advice, Advice{
			NodeID:   m.NodeID,
			Severity: AdviceWarning,
			Category: "cpu",
			Title:    "CPU Throttling Detected",
			Message: fmt.Sprintf(
				"Node %q is using %.0f%% of its CPU limit and is being throttled by the kernel CFS scheduler. "+
					"Throttling increases step execution time significantly.",
				m.NodeID, m.CPUPercent,
			),
			Action: "Increase resources.limits.cpu or switch this node to a burstable QoS class " +
				"by setting requests lower than limits.",
		})
	}

	// ── Error rate ────────────────────────────────────────────────────────────
	if m.ErrorRate >= errorRateCritical {
		advice = append(advice, Advice{
			NodeID:   m.NodeID,
			Severity: AdviceCritical,
			Category: "reliability",
			Title:    "High Error Rate",
			Message: fmt.Sprintf(
				"Node %q has a %.0f%% error rate in the last 5 minutes. "+
					"The workflow will fail frequently at this rate.",
				m.NodeID, m.ErrorRate,
			),
			Action: "Check the node logs for stack traces. Consider adding a retry policy " +
				"(retryPolicy.maxRetries: 3) and exponential backoff to handle transient failures.",
		})
	} else if m.ErrorRate >= errorRateWarning {
		advice = append(advice, Advice{
			NodeID:   m.NodeID,
			Severity: AdviceWarning,
			Category: "reliability",
			Title:    "Elevated Error Rate",
			Message: fmt.Sprintf(
				"Node %q has a %.0f%% error rate. This may indicate rate limiting, "+
					"connection timeouts, or data quality issues.",
				m.NodeID, m.ErrorRate,
			),
			Action: "Review the node logs and consider adding connection pooling or retry logic.",
		})
	}

	// ── NATS queue backpressure ───────────────────────────────────────────────
	if m.QueueDepth >= queueDepthWarning {
		advice = append(advice, Advice{
			NodeID:   m.NodeID,
			Severity: AdviceWarning,
			Category: "throughput",
			Title:    "NATS Queue Backpressure",
			Message: fmt.Sprintf(
				"The NATS queue for node %q has %d pending messages. "+
					"This node is not processing messages fast enough.",
				m.NodeID, m.QueueDepth,
			),
			Action: "Enable KEDA auto-scaling for this node to spin up additional consumer replicas " +
				"automatically based on queue depth. Or increase the node's CPU/memory allocation.",
		})
	}

	// ── Slow step detection ───────────────────────────────────────────────────
	if m.P99DurationMs >= int64(p99DurationWarnMs) {
		advice = append(advice, Advice{
			NodeID:   m.NodeID,
			Severity: AdviceInfo,
			Category: "performance",
			Title:    "Slow Step Execution (p99)",
			Message: fmt.Sprintf(
				"Node %q has a p99 step duration of %ds. "+
					"Slow steps can cascade latency across downstream nodes.",
				m.NodeID, m.P99DurationMs/1000,
			),
			Action: "Profile the node to identify bottlenecks. For I/O-bound nodes, consider " +
				"enabling parallel processing by splitting the payload and using a fan-out topology. " +
				"For compute-bound nodes, increase CPU limits or switch to a GPU-enabled node pool.",
		})
	}

	return advice
}

// ─────────────────────────────────────────────
//  LLM Prompt builder
// ─────────────────────────────────────────────

// BuildLLMPrompt constructs the structured prompt sent to the LLM Advisor.
// The prompt is designed to produce concise, actionable plain-English advice
// that matches the level of detail shown in the spec (Section 4).
func BuildLLMPrompt(m NodeMetrics, ruleAdvice []Advice) string {
	existingAdvice, _ := json.Marshal(ruleAdvice)

	return strings.TrimSpace(fmt.Sprintf(`
You are an infrastructure optimization advisor for the KubeWorkFlow platform.
Your job is to provide concise, actionable, plain-English recommendations to improve
workflow performance and reliability. Focus on changes the user can make in their
KubeWorkFlow configuration — not generic Kubernetes advice.

## Current Node Metrics

- Node ID: %s
- Workflow ID: %s
- CPU Usage: %.1f%% of limit
- Memory Usage: %.1fMB of %.1fMB limit (%.1f%%)
- Average Step Duration: %dms (p99: %dms)
- Error Rate: %.1f%%
- NATS Queue Depth: %d messages
- Average Payload Size: %.1fMB

## Rule-Based Issues Already Detected

%s

## Your Task

Review the metrics above and provide 2-3 additional insights the rule-based system may have missed.
Focus on patterns, correlations, and workflow-level optimizations.
Format each recommendation as a brief paragraph starting with the impact and ending with the specific action.
Do NOT repeat any advice already listed in the rule-based issues above.
Use plain English — this will be shown directly to a non-technical user in the UI.
`,
		m.NodeID, m.WorkflowID,
		m.CPUPercent,
		m.MemMB, m.MemLimitMB, m.MemPercent,
		m.DurationMs, m.P99DurationMs,
		m.ErrorRate,
		m.QueueDepth,
		m.PayloadSizeMB,
		string(existingAdvice),
	))
}

// ─────────────────────────────────────────────
//  Advisor loop
// ─────────────────────────────────────────────

// AdvisorLoop polls for metrics and generates advice on a configurable interval.
// It implements the "Active Telemetry Monitoring Loop" from the spec (Section 4).
//
// Usage:
//
//	loop := NewAdvisorLoop(ruleAdvisor, metricsClient, natsClient)
//	go loop.Run(ctx, "default", "my-workflow-id", 60*time.Second)
type AdvisorLoop struct {
	rules *RuleAdvisor
	// In production: add Prometheus client, NATS client, and optional LLM client
}

// NewAdvisorLoop constructs a new AdvisorLoop.
func NewAdvisorLoop(rules *RuleAdvisor) *AdvisorLoop {
	return &AdvisorLoop{rules: rules}
}

// Run starts the polling loop. It calls the metrics collector, runs the rule advisor,
// optionally calls the LLM advisor, and publishes advice to NATS.
// The loop runs until the context is cancelled.
func (l *AdvisorLoop) Run(ctx context.Context, namespace, workflowID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("[AdvisorLoop] Starting for workflow %s/%s (interval=%s)\n", namespace, workflowID, interval)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[AdvisorLoop] Stopping for workflow %s/%s\n", namespace, workflowID)
			return
		case <-ticker.C:
			// In production:
			// 1. Query Prometheus for each node's metrics
			// 2. Run rule advisor
			// 3. If rule advisor found issues OR LLM enabled: call LLM
			// 4. Publish advice to NATS: flowengine.{ns}.advisor.{nodeId}.advice
			fmt.Printf("[AdvisorLoop] Tick: collecting metrics for %s/%s\n", namespace, workflowID)
		}
	}
}
