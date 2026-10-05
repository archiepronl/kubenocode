package debug

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestInterceptAndStreamForwardsEventsAndBytes(t *testing.T) {
	sidecar := NewSidecar("orders", "transform")
	var output bytes.Buffer
	if err := sidecar.InterceptAndStream(context.Background(), strings.NewReader("first\nsecond\n"), &output, false); err != nil {
		t.Fatalf("InterceptAndStream() error = %v", err)
	}
	if output.String() != "first\nsecond\n" {
		t.Fatalf("forwarded output = %q", output.String())
	}
	if got := sidecar.Events(); len(got) != 3 {
		t.Fatalf("event count = %d, want two logs and completion", len(got))
	}
	for _, want := range []EventType{EventLog, EventLog, EventCompleted} {
		event := <-sidecar.Events()
		if event.Type != want || event.AppName != "orders" || event.NodeID != "transform" || event.Timestamp.IsZero() {
			t.Fatalf("unexpected event: %+v", event)
		}
	}
}

func TestInterceptAndStreamBreakpointCanResume(t *testing.T) {
	sidecar := NewSidecar("orders", "step")
	finished := make(chan error, 1)
	go func() {
		finished <- sidecar.InterceptAndStream(context.Background(), strings.NewReader("payload\n"), io.Discard, true)
	}()
	select {
	case event := <-sidecar.Events():
		if event.Type != EventPaused || !sidecar.IsPaused() {
			t.Fatalf("expected paused event and state, got %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("sidecar did not pause")
	}
	sidecar.Continue()
	sidecar.Continue()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("InterceptAndStream() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("sidecar did not resume")
	}
	if sidecar.IsPaused() {
		t.Fatal("sidecar remained paused after Continue")
	}
	events := []EventType{(<-sidecar.Events()).Type, (<-sidecar.Events()).Type, (<-sidecar.Events()).Type}
	if !containsEvent(events, EventResumed) || !containsEvent(events, EventCompleted) {
		t.Fatalf("missing resumed or completed event: %v", events)
	}
}

func TestInterceptAndStreamErrorAndCancellationPaths(t *testing.T) {
	t.Run("writer error", func(t *testing.T) {
		err := NewSidecar("app", "node").InterceptAndStream(context.Background(), strings.NewReader("line\n"), failingWriter{}, false)
		if err == nil || !strings.Contains(err.Error(), "writing to dst") {
			t.Fatalf("InterceptAndStream() error = %v, want writer error", err)
		}
	})
	t.Run("scanner token too large", func(t *testing.T) {
		input := strings.NewReader(strings.Repeat("x", (1<<20)+1))
		err := NewSidecar("app", "node").InterceptAndStream(context.Background(), input, io.Discard, false)
		if err == nil || !strings.Contains(err.Error(), "scanner") {
			t.Fatalf("InterceptAndStream() error = %v, want scanner error", err)
		}
	})
	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := NewSidecar("app", "node").InterceptAndStream(ctx, strings.NewReader("line\n"), io.Discard, false)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("InterceptAndStream() error = %v, want context.Canceled", err)
		}
	})
	t.Run("cancel while paused", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		sidecar := NewSidecar("app", "node")
		finished := make(chan error, 1)
		go func() { finished <- sidecar.InterceptAndStream(ctx, strings.NewReader("line\n"), io.Discard, true) }()
		select {
		case <-sidecar.Events():
		case <-time.After(time.Second):
			t.Fatal("missing paused event")
		}
		cancel()
		select {
		case err := <-finished:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("paused cancellation returned error = %v, want context.Canceled", err)
			}
		case <-time.After(time.Second):
			t.Fatal("pause did not unblock on context cancellation")
		}
	})
}

func TestSidecarHelpersAndEventOverflow(t *testing.T) {
	sidecar := NewSidecar("app", "node")
	sidecar.Continue()
	for range 64 {
		sidecar.emit(EventLog, "fill", nil)
	}
	sidecar.emit(EventCompleted, "dropped", nil)
	if len(sidecar.Events()) != 64 {
		t.Fatalf("event queue length = %d, want capped at 64", len(sidecar.Events()))
	}
	container := SidecarContainer("orders", "source", "api:8080")
	if container["name"] != "kttm-debug-sidecar" || container["image"] != "kttm/debug-sidecar:latest" {
		t.Fatalf("unexpected sidecar container: %v", container)
	}
	if len(container["env"].([]map[string]string)) != 3 {
		t.Fatalf("unexpected sidecar env: %v", container["env"])
	}
}

func containsEvent(events []EventType, want EventType) bool {
	for _, event := range events {
		if event == want {
			return true
		}
	}
	return false
}