package materializer

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type testProcessor struct {
	mimeTypes []string
	result    interface{}
	err       error
}

func (p testProcessor) MimeTypes() []string { return p.mimeTypes }

func (p testProcessor) Process(_ context.Context, _ interface{}, r io.Reader) (interface{}, io.ReadCloser, error) {
	if p.err != nil {
		return nil, nil, p.err
	}
	return p.result, io.NopCloser(r), nil
}

func TestRouteStorageBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		crossPod bool
		want     StorageBackend
	}{
		{name: "negative size routes to shared memory", size: -1, want: StorageShm},
		{name: "below shared memory threshold", size: shmThresholdBytes - 1, want: StorageShm},
		{name: "shared memory threshold routes to emptydir", size: shmThresholdBytes, want: StorageEmptyDir},
		{name: "below emptydir threshold", size: emptyDirThresholdBytes - 1, want: StorageEmptyDir},
		{name: "emptydir threshold routes to minio", size: emptyDirThresholdBytes, want: StorageMinIO},
		{name: "cross pod routes to minio", size: 1, crossPod: true, want: StorageMinIO},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RouteStorage(tt.size, tt.crossPod); got != tt.want {
				t.Fatalf("RouteStorage(%d, %t) = %q, want %q", tt.size, tt.crossPod, got, tt.want)
			}
		})
	}
}

func TestPayloadPath(t *testing.T) {
	tests := []struct {
		backend StorageBackend
		want    string
	}{
		{backend: StorageShm, want: "/dev/shm/flowengine/run-a/node-b.bin"},
		{backend: StorageEmptyDir, want: "/data/flowengine/run-a/node-b.bin"},
		{backend: StorageMinIO, want: "s3://flowengine-payloads/run-a/node-b"},
		{backend: StorageInline, want: ""},
		{backend: StorageBackend("unknown"), want: ""},
	}
	for _, tt := range tests {
		if got := PayloadPath(tt.backend, "run-a", "node-b"); got != tt.want {
			t.Errorf("PayloadPath(%q) = %q, want %q", tt.backend, got, tt.want)
		}
	}
}

func TestMaterializerDispatchAndFallback(t *testing.T) {
	input := &Envelope{MimeType: " Application/PDF ", ExecutionID: "run-a"}
	materializer := New()
	gotEnvelope, output, err := materializer.Materialize(context.Background(), input, strings.NewReader("pdf bytes"))
	if err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}
	if gotEnvelope != input {
		t.Fatalf("Materialize() envelope = %p, want original %p", gotEnvelope, input)
	}
	assertStream(t, output, "pdf bytes")

	passthrough := &Materializer{}
	gotEnvelope, output, err = passthrough.Materialize(context.Background(), input, strings.NewReader("unmatched"))
	if err != nil {
		t.Fatalf("fallback Materialize() error = %v", err)
	}
	if gotEnvelope != input {
		t.Fatal("fallback did not preserve the input envelope")
	}
	assertStream(t, output, "unmatched")
}

func TestMaterializerCustomAdapterOutcomes(t *testing.T) {
	input := &Envelope{MimeType: "application/custom"}
	replacement := &Envelope{MimeType: "application/processed"}

	t.Run("first matching adapter returns replacement envelope", func(t *testing.T) {
		m := &Materializer{}
		m.Register(testProcessor{mimeTypes: []string{"application/"}, result: &Envelope{MimeType: "wrong"}})
		m.Register(testProcessor{mimeTypes: []string{"application/custom"}, result: replacement})
		got, output, err := m.Materialize(context.Background(), input, strings.NewReader("custom"))
		if err != nil {
			t.Fatalf("Materialize() error = %v", err)
		}
		if got.MimeType != "wrong" {
			t.Fatalf("first matching adapter was not selected: %+v", got)
		}
		assertStream(t, output, "custom")
	})

	t.Run("non-envelope result preserves input envelope", func(t *testing.T) {
		m := &Materializer{}
		m.Register(testProcessor{mimeTypes: []string{"application/custom"}, result: "not an envelope"})
		got, output, err := m.Materialize(context.Background(), input, strings.NewReader("custom"))
		if err != nil {
			t.Fatalf("Materialize() error = %v", err)
		}
		if got != input {
			t.Fatal("non-envelope adapter result replaced the input envelope")
		}
		assertStream(t, output, "custom")
	})

	t.Run("processor error is returned", func(t *testing.T) {
		m := &Materializer{}
		m.Register(testProcessor{mimeTypes: []string{"application/custom"}, err: errors.New("process failed")})
		got, output, err := m.Materialize(context.Background(), input, strings.NewReader("custom"))
		if err == nil || got != nil || output != nil {
			t.Fatalf("Materialize() = (%v, %v, %v), want error with nil outputs", got, output, err)
		}
	})
}

func TestSelectAdapterNormalizesAndRequiresPrefix(t *testing.T) {
	m := &Materializer{}
	first := testProcessor{mimeTypes: []string{"application/"}}
	second := testProcessor{mimeTypes: []string{"application/json"}}
	m.Register(first)
	m.Register(second)
	if got := m.selectAdapter("  APPLICATION/JSON; charset=utf-8 "); got == nil {
		t.Fatal("selectAdapter did not normalize MIME type")
	}
	if got := m.selectAdapter("text/plain"); got != nil {
		t.Fatalf("selectAdapter(text/plain) = %v, want nil", got)
	}
}

func assertStream(t *testing.T, stream io.ReadCloser, want string) {
	t.Helper()
	if stream == nil {
		t.Fatal("stream is nil")
	}
	got, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("reading stream: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Errorf("closing stream: %v", err)
	}
	if string(got) != want {
		t.Errorf("stream = %q, want %q", got, want)
	}
}
