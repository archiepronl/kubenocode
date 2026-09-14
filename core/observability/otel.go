// Package observability implements OpenTelemetry instrumentation for FlowEngine.
//
// This package provides:
//   - Trace context propagation through the full workflow execution pipeline
//   - Per-node duration histograms and resource usage gauges
//   - OTLP exporter configuration (Jaeger, Tempo, or any OTLP-compatible backend)
//
// Implements spec Section 4 (Q2 2028 roadmap):
//
//	"Embed an OpenTelemetry agent within the execution runtime. Track processing
//	 times, internal mutations, and file transfers as single trace IDs whenever an
//	 unstructured payload moves through a workflow."
//
// Metric definitions (NFR telemetry targets):
//
//	flowengine_node_duration_seconds  — histogram (p50/p99 step execution time)
//	flowengine_node_memory_ratio      — gauge (mem used / mem limit)
//	flowengine_node_cpu_throttle_ratio— gauge (CPU throttled / CPU limit)
//	flowengine_envelope_payload_bytes — histogram (payload size distribution)
//	flowengine_nats_queue_depth       — gauge (NATS consumer pending count)
package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// ─────────────────────────────────────────────
//  Package-level tracer and meter
// ─────────────────────────────────────────────

const instrumentationName = "github.com/kubeworkflow/flowengine"

// Tracer is the package-level OpenTelemetry tracer.
// Used to instrument workflow execution steps.
var Tracer trace.Tracer

// Meter is the package-level OpenTelemetry meter.
// Used for Prometheus-compatible metric recording.
var Meter metric.Meter

// ─────────────────────────────────────────────
//  Config
// ─────────────────────────────────────────────

// Config holds OpenTelemetry setup parameters.
type Config struct {
	// ServiceName identifies this process in traces (e.g., "flowengine-operator").
	ServiceName string

	// ServiceVersion is the application version (injected via -ldflags).
	ServiceVersion string

	// OTLPEndpoint is the gRPC endpoint for the OTLP trace collector.
	// Example: "jaeger.monitoring.svc.cluster.local:4317"
	// Leave empty to disable trace exporting (useful in air-gapped environments).
	OTLPEndpoint string

	// PrometheusPort is the port where the Prometheus metrics HTTP handler listens.
	// If 0, Prometheus metrics are not exposed.
	PrometheusPort int
}

// ─────────────────────────────────────────────
//  Metrics instruments
// ─────────────────────────────────────────────

// Instruments holds all registered metric instruments.
// Initialize once via Setup() and use throughout the application.
type Instruments struct {
	// NodeDuration tracks how long each workflow node step takes.
	// Labels: workflow_id, node_id, node_type, namespace
	NodeDuration metric.Float64Histogram

	// NodeMemoryRatio tracks memory usage as a fraction of the limit (0.0 – 1.0).
	NodeMemoryRatio metric.Float64ObservableGauge

	// NodeCPUThrottleRatio tracks CPU throttling fraction (0.0 – 1.0).
	NodeCPUThrottleRatio metric.Float64ObservableGauge

	// EnvelopePayloadBytes tracks the distribution of payload sizes flowing through the system.
	EnvelopePayloadBytes metric.Int64Histogram

	// NATSQueueDepth tracks the number of pending messages in NATS consumers.
	NATSQueueDepth metric.Int64ObservableGauge

	// WorkflowExecutions counts total workflow execution attempts.
	// Labels: workflow_id, namespace, phase (succeeded|failed|running)
	WorkflowExecutions metric.Int64Counter
}

// ─────────────────────────────────────────────
//  Setup
// ─────────────────────────────────────────────

// Setup initializes the OpenTelemetry SDK: tracer provider, meter provider,
// and all metric instruments. Returns a shutdown function that must be called
// on application exit to flush pending telemetry.
func Setup(ctx context.Context, cfg Config) (*Instruments, func(), error) {
	// Build the service resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("building OTel resource: %w", err)
	}

	// ── Trace provider ────────────────────────────────────────────────────────
	var traceShutdown func(context.Context) error

	tracerOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}

	if cfg.OTLPEndpoint != "" {
		exp, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlptracegrpc.WithInsecure(), // TLS configured externally via SPIFFE/SPIRE mTLS
		)
		if err != nil {
			return nil, nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
		}
		tracerOpts = append(tracerOpts, sdktrace.WithBatcher(exp))
		traceShutdown = exp.Shutdown
	}

	tp := sdktrace.NewTracerProvider(tracerOpts...)
	otel.SetTracerProvider(tp)
	Tracer = tp.Tracer(instrumentationName)

	// ── Metric provider ───────────────────────────────────────────────────────
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	Meter = mp.Meter(instrumentationName)

	// ── Register instruments ──────────────────────────────────────────────────
	inst, err := registerInstruments(Meter)
	if err != nil {
		return nil, nil, fmt.Errorf("registering OTel instruments: %w", err)
	}

	// Composite shutdown function
	shutdown := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("[OTel] Error shutting down tracer provider: %v\n", err)
		}
		if err := mp.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("[OTel] Error shutting down meter provider: %v\n", err)
		}
		if traceShutdown != nil {
			if err := traceShutdown(shutdownCtx); err != nil {
				fmt.Printf("[OTel] Error shutting down OTLP exporter: %v\n", err)
			}
		}
	}

	fmt.Printf("[OTel] Initialized — service=%s version=%s endpoint=%s\n",
		cfg.ServiceName, cfg.ServiceVersion, cfg.OTLPEndpoint)
	return inst, shutdown, nil
}

// registerInstruments creates all metric instruments on the given meter.
func registerInstruments(meter metric.Meter) (*Instruments, error) {
	nodeDuration, err := meter.Float64Histogram(
		"flowengine_node_duration_seconds",
		metric.WithDescription("Duration of workflow node step execution in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(.05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120),
	)
	if err != nil {
		return nil, fmt.Errorf("creating node_duration histogram: %w", err)
	}

	envelopeBytes, err := meter.Int64Histogram(
		"flowengine_envelope_payload_bytes",
		metric.WithDescription("Distribution of raw payload sizes routed through the envelope system"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(
			1<<10,   // 1KB
			1<<20,   // 1MB
			10<<20,  // 10MB
			100<<20, // 100MB
			1<<30,   // 1GB
			10<<30,  // 10GB
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating envelope_payload_bytes histogram: %w", err)
	}

	wfExecutions, err := meter.Int64Counter(
		"flowengine_workflow_executions_total",
		metric.WithDescription("Total number of workflow execution attempts"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating workflow_executions counter: %w", err)
	}

	// Async gauges (read externally via callback — actual values come from kubelet/Prometheus)
	memRatio, err := meter.Float64ObservableGauge(
		"flowengine_node_memory_ratio",
		metric.WithDescription("Ratio of memory used to memory limit for a workflow node (0.0–1.0)"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating node_memory_ratio gauge: %w", err)
	}

	cpuThrottle, err := meter.Float64ObservableGauge(
		"flowengine_node_cpu_throttle_ratio",
		metric.WithDescription("Ratio of CPU throttled time to total CPU time for a workflow node"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating node_cpu_throttle_ratio gauge: %w", err)
	}

	queueDepth, err := meter.Int64ObservableGauge(
		"flowengine_nats_queue_depth",
		metric.WithDescription("Number of pending messages in the NATS consumer for a workflow node"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating nats_queue_depth gauge: %w", err)
	}

	return &Instruments{
		NodeDuration:         nodeDuration,
		NodeMemoryRatio:      memRatio,
		NodeCPUThrottleRatio: cpuThrottle,
		EnvelopePayloadBytes: envelopeBytes,
		NATSQueueDepth:       queueDepth,
		WorkflowExecutions:   wfExecutions,
	}, nil
}

// ─────────────────────────────────────────────
//  Trace helpers
// ─────────────────────────────────────────────

// NodeAttributes returns a standard set of OTel span/metric attributes for a workflow node.
func NodeAttributes(workflowID, nodeID, nodeType, namespace string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("flowengine.workflow.id", workflowID),
		attribute.String("flowengine.node.id", nodeID),
		attribute.String("flowengine.node.type", nodeType),
		attribute.String("flowengine.namespace", namespace),
	}
}

// StartNodeSpan opens a new OTel trace span for a workflow node execution.
// The caller must call the returned span.End() when the step completes.
//
// Usage:
//
//	ctx, span := observability.StartNodeSpan(ctx, "my-workflow", "source-s3", "connector/s3", "default")
//	defer span.End()
func StartNodeSpan(ctx context.Context, workflowID, nodeID, nodeType, namespace string) (context.Context, trace.Span) {
	return Tracer.Start(ctx, fmt.Sprintf("node.%s", nodeID),
		trace.WithAttributes(NodeAttributes(workflowID, nodeID, nodeType, namespace)...),
	)
}

// RecordEnvelopeTransfer records the payload size metric when an envelope is routed.
func (inst *Instruments) RecordEnvelopeTransfer(ctx context.Context, payloadBytes int64, workflowID, nodeID, backend string) {
	inst.EnvelopePayloadBytes.Record(ctx, payloadBytes,
		metric.WithAttributes(
			attribute.String("flowengine.workflow.id", workflowID),
			attribute.String("flowengine.node.id", nodeID),
			attribute.String("flowengine.storage.backend", backend),
		),
	)
}

// RecordNodeDuration records the step execution duration for a completed node.
func (inst *Instruments) RecordNodeDuration(ctx context.Context, durationSec float64, workflowID, nodeID, nodeType, namespace string) {
	inst.NodeDuration.Record(ctx, durationSec,
		metric.WithAttributes(NodeAttributes(workflowID, nodeID, nodeType, namespace)...),
	)
}

// RecordWorkflowExecution increments the execution counter with a phase label.
func (inst *Instruments) RecordWorkflowExecution(ctx context.Context, workflowID, namespace, phase string) {
	inst.WorkflowExecutions.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("flowengine.workflow.id", workflowID),
			attribute.String("flowengine.namespace", namespace),
			attribute.String("flowengine.phase", phase),
		),
	)
}
