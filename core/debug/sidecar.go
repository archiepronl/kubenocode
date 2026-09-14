// Package debug implements the interactive breakpoint debugger for KTTM.
//
// KTTM-REQ-014: The execution plane can pause at user-defined breakpoints,
// stream granular console logs and step states in real-time, and allow the
// developer to inspect the raw data payload before proceeding to the next node.
//
// Architecture:
//   - The kttm-debug-sidecar container is injected into Argo step pods
//     when KttmApp.spec.debug.enabled = true.
//   - The sidecar intercepts stdio between the step container and NATS,
//     holding the pipe open until the developer sends a "continue" signal.
//   - The BFF API server bridges WebSocket connections from the Developer SDK
//     to the debug sidecar via gRPC streaming.
package debug

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  Event types
// ─────────────────────────────────────────────

// EventType identifies the kind of debug event.
type EventType string

const (
	EventLog       EventType = "log"       // Stdout/stderr line from the step container
	EventPaused    EventType = "paused"    // Execution paused at breakpoint
	EventPayload   EventType = "payload"   // Raw payload bytes (hex dump or JSON)
	EventResumed   EventType = "resumed"   // Developer sent "continue"
	EventCompleted EventType = "completed" // Step finished
)

// DebugEvent is a single event emitted by the debug sidecar.
type DebugEvent struct {
	Type      EventType       `json:"type"`
	NodeID    string          `json:"nodeId"`
	AppName   string          `json:"appName"`
	Timestamp time.Time       `json:"timestamp"`
	Message   string          `json:"message,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"` // Raw bytes excerpt (first 4KB)
	ExitCode  *int            `json:"exitCode,omitempty"`
}

// ─────────────────────────────────────────────
//  Sidecar
// ─────────────────────────────────────────────

// Sidecar is the debug sidecar component that runs in the Argo step pod.
// It intercepts stdout of the step container and holds it at breakpoints.
type Sidecar struct {
	nodeID  string
	appName string

	// events is the channel used to emit debug events to the BFF.
	events chan DebugEvent

	// mu guards the paused state.
	mu     sync.Mutex
	paused bool
	resume chan struct{} // closed to unblock the step
}

// NewSidecar creates a new debug sidecar for a given node.
func NewSidecar(appName, nodeID string) *Sidecar {
	return &Sidecar{
		appName: appName,
		nodeID:  nodeID,
		events:  make(chan DebugEvent, 64),
		resume:  make(chan struct{}),
	}
}

// Events returns a read-only channel of debug events.
// The BFF proxies these over WebSocket to the Developer SDK.
func (s *Sidecar) Events() <-chan DebugEvent { return s.events }

// ─────────────────────────────────────────────
//  Intercept pipeline
// ─────────────────────────────────────────────

// InterceptAndStream reads from src (step container stdout/stderr),
// forwards log lines to the events channel, and pauses at breakpoints
// until Continue() is called.
//
// Once drained, the data is forwarded to dst (NATS pipe or next sink).
// This preserves the raw payload without modification (KTTM-REQ-031).
func (s *Sidecar) InterceptAndStream(ctx context.Context, src io.Reader, dst io.Writer, breakOnStart bool) error {
	// Emit initial paused event if this node has a breakpoint
	if breakOnStart {
		s.pause(ctx, nil)
	}

	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1MB scan buffer for large payloads

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()

		// Stream log line to Developer SDK
		s.emit(EventLog, string(line), nil)

		// Forward byte-for-byte to dst (no normalization — KTTM-NFR raw preservation)
		if _, err := fmt.Fprintf(dst, "%s\n", line); err != nil {
			return fmt.Errorf("debug sidecar: writing to dst: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("debug sidecar: scanner: %w", err)
	}

	s.emit(EventCompleted, "step output stream ended", nil)
	return nil
}

// ─────────────────────────────────────────────
//  Breakpoint control
// ─────────────────────────────────────────────

// pause emits a Paused event and blocks until Continue() is called.
// payloadExcerpt is an optional JSON representation of the first 4KB of payload.
func (s *Sidecar) pause(ctx context.Context, payloadExcerpt json.RawMessage) {
	s.mu.Lock()
	s.paused = true
	s.resume = make(chan struct{}) // re-arm
	s.mu.Unlock()

	s.emit(EventPaused, fmt.Sprintf("Paused at node: %s. Send 'continue' to proceed.", s.nodeID), payloadExcerpt)

	// Block until context is cancelled or developer sends Continue()
	select {
	case <-s.resume:
		s.emit(EventResumed, "Developer resumed execution", nil)
	case <-ctx.Done():
	}
}

// Continue unblocks a paused node, allowing the step to proceed.
// Called by the BFF when the Developer SDK sends a "continue" command.
func (s *Sidecar) Continue() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.paused {
		close(s.resume)
		s.paused = false
	}
}

// IsPaused returns true if execution is currently held at a breakpoint.
func (s *Sidecar) IsPaused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.paused
}

// ─────────────────────────────────────────────
//  Emit helper
// ─────────────────────────────────────────────

func (s *Sidecar) emit(t EventType, msg string, payload json.RawMessage) {
	ev := DebugEvent{
		Type:      t,
		NodeID:    s.nodeID,
		AppName:   s.appName,
		Timestamp: time.Now().UTC(),
		Message:   msg,
		Payload:   payload,
	}
	select {
	case s.events <- ev:
	default:
		// Channel full — drop event rather than blocking step execution
	}
}

// ─────────────────────────────────────────────
//  Sidecar pod spec injection
// ─────────────────────────────────────────────

// SidecarContainer returns the Kubernetes container spec for the debug sidecar.
// Injected into Argo Workflow step pods when debug.enabled = true (KTTM-REQ-014).
func SidecarContainer(appName, nodeID, bffAddr string) map[string]interface{} {
	return map[string]interface{}{
		"name":  "kttm-debug-sidecar",
		"image": "kttm/debug-sidecar:latest",
		"env": []map[string]string{
			{"name": "KTTM_APP_NAME", "value": appName},
			{"name": "KTTM_NODE_ID", "value": nodeID},
			{"name": "KTTM_BFF_ADDR", "value": bffAddr},
		},
		"resources": map[string]interface{}{
			"requests": map[string]string{"cpu": "10m", "memory": "16Mi"},
			"limits":   map[string]string{"cpu": "50m", "memory": "32Mi"},
		},
	}
}
