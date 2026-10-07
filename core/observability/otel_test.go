package observability

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type failingMeter struct {
	metric.Meter
	failName string
	failErr  error
}

type testExporter struct{}

func (testExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }

func (testExporter) Shutdown(context.Context) error { return nil }

func (m failingMeter) Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	if name == m.failName {
		return nil, m.failErr
	}
	return m.Meter.Float64Histogram(name, options...)
}

func (m failingMeter) Int64Histogram(name string, options ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	if name == m.failName {
		return nil, m.failErr
	}
	return m.Meter.Int64Histogram(name, options...)
}

func (m failingMeter) Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	if name == m.failName {
		return nil, m.failErr
	}
	return m.Meter.Int64Counter(name, options...)
}

func (m failingMeter) Float64ObservableGauge(name string, options ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error) {
	if name == m.failName {
		return nil, m.failErr
	}
	return m.Meter.Float64ObservableGauge(name, options...)
}

func (m failingMeter) Int64ObservableGauge(name string, options ...metric.Int64ObservableGaugeOption) (metric.Int64ObservableGauge, error) {
	if name == m.failName {
		return nil, m.failErr
	}
	return m.Meter.Int64ObservableGauge(name, options...)
}

func TestSetupInstrumentsMetricsAndSpans(t *testing.T) {
	ctx := context.Background()
	instruments, shutdown, err := Setup(ctx, Config{ServiceName: "flowengine-test", ServiceVersion: "test"})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if instruments == nil || Tracer == nil || Meter == nil {
		t.Fatal("Setup() did not initialize telemetry globals and instruments")
	}
	shutdown()

	attributes := NodeAttributes("workflow", "node", "script/python", "team-a")
	if len(attributes) != 4 || attributes[0] != attribute.String("flowengine.workflow.id", "workflow") ||
		attributes[3] != attribute.String("flowengine.namespace", "team-a") {
		t.Fatalf("NodeAttributes() = %v", attributes)
	}
	spanCtx, span := StartNodeSpan(ctx, "workflow", "node", "script/python", "team-a")
	if span == nil || spanCtx == nil || !span.SpanContext().IsValid() {
		t.Fatal("StartNodeSpan() did not create a valid span")
	}
	span.End()

	instruments.RecordEnvelopeTransfer(ctx, 1024, "workflow", "node", "shm")
	instruments.RecordNodeDuration(ctx, 0.25, "workflow", "node", "script/python", "team-a")
	instruments.RecordWorkflowExecution(ctx, "workflow", "team-a", "Succeeded")
}

func TestRegisterInstrumentsReturnsCreationErrors(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	failures := []string{
		"flowengine_node_duration_seconds",
		"flowengine_envelope_payload_bytes",
		"flowengine_workflow_executions_total",
		"flowengine_node_memory_ratio",
		"flowengine_node_cpu_throttle_ratio",
		"flowengine_nats_queue_depth",
	}
	for _, name := range failures {
		t.Run(name, func(t *testing.T) {
			wantErr := errors.New("instrument creation failed")
			meter := failingMeter{Meter: provider.Meter("observability-test-" + name), failName: name, failErr: wantErr}
			if _, err := registerInstruments(meter); !errors.Is(err, wantErr) {
				t.Fatalf("registerInstruments() error = %v, want %v", err, wantErr)
			}
		})
	}
}

func TestSetupReturnsInstrumentRegistrationError(t *testing.T) {
	wantErr := errors.New("instrument registration failed")
	dependencies := defaultSetupDependencies()
	dependencies.registerInstruments = func(metric.Meter) (*Instruments, error) {
		return nil, wantErr
	}
	_, _, err := setup(context.Background(), Config{ServiceName: "test"}, dependencies)
	if !errors.Is(err, wantErr) {
		t.Fatalf("setup() error = %v, want registration error", err)
	}
}

func TestSetupRejectsMalformedOTLPEndpoint(t *testing.T) {
	if _, _, err := Setup(context.Background(), Config{OTLPEndpoint: "%"}); err == nil {
		t.Fatal("Setup() accepted a malformed OTLP endpoint")
	}
}

func TestCreateOTLPTraceExporterSuccess(t *testing.T) {
	_, shutdown, err := createOTLPTraceExporter(context.Background(), "127.0.0.1:4317")
	if err != nil {
		t.Fatalf("createOTLPTraceExporter() error = %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("exporter shutdown error = %v", err)
	}
}

func TestSetupResourceAndExporterFailures(t *testing.T) {
	wantErr := errors.New("dependency failed")
	t.Run("resource factory", func(t *testing.T) {
		dependencies := defaultSetupDependencies()
		dependencies.buildResource = func(Config) (*resource.Resource, error) { return nil, wantErr }
		if _, _, err := setup(context.Background(), Config{}, dependencies); !errors.Is(err, wantErr) {
			t.Fatalf("setup() error = %v, want resource error", err)
		}
	})
	t.Run("exporter factory", func(t *testing.T) {
		dependencies := defaultSetupDependencies()
		dependencies.createExporter = func(context.Context, string) (sdktrace.SpanExporter, func(context.Context) error, error) {
			return nil, nil, wantErr
		}
		if _, _, err := setup(context.Background(), Config{OTLPEndpoint: "collector:4317"}, dependencies); !errors.Is(err, wantErr) {
			t.Fatalf("setup() error = %v, want exporter error", err)
		}
	})
}

func TestSetupExporterSuccessAndShutdownError(t *testing.T) {
	wantErr := errors.New("exporter shutdown failed")
	dependencies := defaultSetupDependencies()
	dependencies.createExporter = func(context.Context, string) (sdktrace.SpanExporter, func(context.Context) error, error) {
		return testExporter{}, func(context.Context) error { return wantErr }, nil
	}
	_, shutdown, err := setup(context.Background(), Config{OTLPEndpoint: "collector:4317"}, dependencies)
	if err != nil {
		t.Fatalf("setup() error = %v", err)
	}
	shutdown()
	logShutdownError("test", nil)
	logShutdownError("test", wantErr)
}
