// Package messaging implements FR-1.3 (Low-Latency Messaging) using NATS.io JetStream.
//
// Architecture decisions:
//   - NATS Core handles ephemeral, at-most-once delivery (UI events, sub-ms webhook triggers)
//   - JetStream handles persistent, at-least-once delivery (workflow envelopes, telemetry)
//   - The event bus carries ONLY JSON Envelope metadata (~200 bytes), never raw payload bytes
//   - Large binaries travel via emptyDir / /dev/shm / MinIO (FR-4.4 dual-channel strategy)
//
// Subject Hierarchy:
//
//	flowengine.{ns}.{workflow-id}.{node-id}.envelope    — Envelope routing
//	flowengine.{ns}.ui.{session-id}.events              — UI reactive bindings (button clicks)
//	flowengine.{ns}.telemetry.{node-id}.metrics         — OpenTelemetry metric fan-out
//	flowengine.{ns}.control.{workflow-id}.status        — Status updates from operator
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ─────────────────────────────────────────────
//  Subject helpers
// ─────────────────────────────────────────────

const (
	// subjectEnvelopeFmt is the NATS subject template for envelope routing.
	subjectEnvelopeFmt = "flowengine.%s.%s.%s.envelope"

	// subjectUIEventsFmt is the NATS subject for UI reactive bindings.
	subjectUIEventsFmt = "flowengine.%s.ui.%s.events"

	// subjectTelemetryFmt is the NATS subject for telemetry metrics.
	subjectTelemetryFmt = "flowengine.%s.telemetry.%s.metrics"

	// subjectStatusFmt is the NATS subject for workflow status updates.
	subjectStatusFmt = "flowengine.%s.control.%s.status"

	// streamName is the JetStream stream that persists all flowengine messages.
	streamName = "FLOWENGINE"
)

// EnvelopeSubject returns the NATS subject for routing a workflow envelope.
func EnvelopeSubject(namespace, workflowID, nodeID string) string {
	return fmt.Sprintf(subjectEnvelopeFmt, namespace, workflowID, nodeID)
}

// UIEventsSubject returns the NATS subject for UI session events.
func UIEventsSubject(namespace, sessionID string) string {
	return fmt.Sprintf(subjectUIEventsFmt, namespace, sessionID)
}

// TelemetrySubject returns the NATS subject for a node's telemetry metrics.
func TelemetrySubject(namespace, nodeID string) string {
	return fmt.Sprintf(subjectTelemetryFmt, namespace, nodeID)
}

// StatusSubject returns the NATS subject for a workflow's status updates.
func StatusSubject(namespace, workflowID string) string {
	return fmt.Sprintf(subjectStatusFmt, namespace, workflowID)
}

// ─────────────────────────────────────────────
//  Message types
// ─────────────────────────────────────────────

// UIEvent is sent over NATS when the user interacts with the no-code UI
// (e.g., clicks a button). It carries the compiled payload template values
// and the target workflow trigger information (FR-2.3 Reactive Binding System).
type UIEvent struct {
	SessionID  string            `json:"sessionId"`
	WorkflowID string            `json:"workflowId"`
	TriggerID  string            `json:"triggerId"`
	Payload    map[string]string `json:"payload"`
	Timestamp  time.Time         `json:"timestamp"`
}

// StatusUpdate is published by the operator when a workflow transitions phase.
type StatusUpdate struct {
	WorkflowID string    `json:"workflowId"`
	Namespace  string    `json:"namespace"`
	Phase      string    `json:"phase"`
	Message    string    `json:"message,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// TelemetryEvent carries per-node runtime metrics for the AI Advisor loop.
type TelemetryEvent struct {
	NodeID     string    `json:"nodeId"`
	WorkflowID string    `json:"workflowId"`
	CPUPercent float64   `json:"cpuPercent"`
	MemMB      float64   `json:"memMb"`
	MemLimitMB float64   `json:"memLimitMb"`
	DurationMs int64     `json:"durationMs"`
	ErrorRate  float64   `json:"errorRate"`
	Timestamp  time.Time `json:"timestamp"`
}

// ─────────────────────────────────────────────
//  Client
// ─────────────────────────────────────────────

// Client is the FlowEngine NATS client. It wraps the NATS connection and JetStream context,
// providing typed publish/subscribe methods for all FlowEngine message types.
type Client struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// Config holds connection parameters for the NATS client.
type Config struct {
	// URL is the NATS server URL (e.g., "nats://nats.flowengine.svc:4222").
	URL string

	// CredentialsFile is the path to a NATS credentials file for authentication.
	// Leave empty for unauthenticated connections (local dev / testing).
	CredentialsFile string

	// MaxReconnects is the number of reconnect attempts before giving up. -1 = unlimited.
	MaxReconnects int

	// ReconnectWait is the delay between reconnect attempts.
	ReconnectWait time.Duration
}

// DefaultConfig returns sensible defaults for local K3d development.
func DefaultConfig() Config {
	return Config{
		URL:           nats.DefaultURL,
		MaxReconnects: -1, // unlimited reconnects
		ReconnectWait: 2 * time.Second,
	}
}

// NewClient establishes a NATS connection and configures the JetStream stream.
// It blocks until the connection is established or the context is cancelled.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	opts := []nats.Option{
		nats.Name("flowengine-client"),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("[NATS] Disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("[NATS] Reconnected to %s\n", nc.ConnectedUrl())
		}),
	}

	if cfg.CredentialsFile != "" {
		opts = append(opts, nats.UserCredentials(cfg.CredentialsFile))
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connecting to NATS at %s: %w", cfg.URL, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("creating JetStream context: %w", err)
	}

	client := &Client{nc: nc, js: js}

	if err := client.ensureStream(ctx); err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensuring JetStream stream: %w", err)
	}

	return client, nil
}

// ensureStream creates the FLOWENGINE JetStream stream if it doesn't exist.
// The stream persists all flowengine.* subjects with a 7-day retention window.
func (c *Client) ensureStream(ctx context.Context) error {
	_, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        streamName,
		Description: "FlowEngine workflow envelope, telemetry, and status messages",
		Subjects:    []string{"flowengine.>"},
		Retention:   jetstream.LimitsPolicy,
		MaxAge:      7 * 24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1, // increase to 3 in production for HA
		Compression: jetstream.S2Compression,
	})
	return err
}

// ─────────────────────────────────────────────
//  Publish methods
// ─────────────────────────────────────────────

// PublishEnvelope publishes an Envelope to the routing subject for the target node.
// The envelope is serialized as JSON; the raw payload bytes are NOT included.
func (c *Client) PublishEnvelope(ctx context.Context, namespace, workflowID, nodeID string, env interface{}) error {
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshalling envelope: %w", err)
	}
	subject := EnvelopeSubject(namespace, workflowID, nodeID)
	_, err = c.js.Publish(ctx, subject, data)
	return err
}

// PublishUIEvent publishes a UI interaction event to the session's NATS subject.
// This is called by the web-renderer's NATS WebSocket bridge when a user clicks a button (FR-2.3).
func (c *Client) PublishUIEvent(ctx context.Context, namespace string, event UIEvent) error {
	event.Timestamp = time.Now().UTC()
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshalling UI event: %w", err)
	}
	subject := UIEventsSubject(namespace, event.SessionID)
	return c.nc.Publish(subject, data) // NATS Core: at-most-once, sub-ms latency
}

// PublishStatus publishes a workflow status update.
func (c *Client) PublishStatus(ctx context.Context, update StatusUpdate) error {
	update.Timestamp = time.Now().UTC()
	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshalling status update: %w", err)
	}
	subject := StatusSubject(update.Namespace, update.WorkflowID)
	_, err = c.js.Publish(ctx, subject, data)
	return err
}

// PublishTelemetry publishes per-node telemetry metrics for the AI Advisor.
func (c *Client) PublishTelemetry(ctx context.Context, namespace string, event TelemetryEvent) error {
	event.Timestamp = time.Now().UTC()
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshalling telemetry event: %w", err)
	}
	subject := TelemetrySubject(namespace, event.NodeID)
	_, err = c.js.Publish(ctx, subject, data)
	return err
}

// ─────────────────────────────────────────────
//  Subscribe methods
// ─────────────────────────────────────────────

// SubscribeEnvelopes subscribes to all envelope messages for a specific workflow.
// The handler is called for each envelope received.
func (c *Client) SubscribeEnvelopes(ctx context.Context, namespace, workflowID string, handler func(data []byte)) error {
	subject := fmt.Sprintf("flowengine.%s.%s.*.envelope", namespace, workflowID)
	consumerName := fmt.Sprintf("flowengine-%s-%s-envelopes", namespace, workflowID)

	cons, err := c.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		return fmt.Errorf("creating envelope consumer: %w", err)
	}

	_, err = cons.Consume(func(msg jetstream.Msg) {
		handler(msg.Data())
		_ = msg.Ack()
	})
	return err
}

// SubscribeUIEvents subscribes to UI events for a specific session.
// Uses NATS Core (not JetStream) for minimum latency (FR-1.3).
func (c *Client) SubscribeUIEvents(namespace, sessionID string, handler func(event UIEvent)) (*nats.Subscription, error) {
	subject := UIEventsSubject(namespace, sessionID)
	return c.nc.Subscribe(subject, func(msg *nats.Msg) {
		var event UIEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			fmt.Printf("[NATS] Failed to decode UI event: %v\n", err)
			return
		}
		handler(event)
	})
}

// ─────────────────────────────────────────────
//  Lifecycle
// ─────────────────────────────────────────────

// Close gracefully closes the NATS connection.
func (c *Client) Close() {
	if c.nc != nil && !c.nc.IsClosed() {
		c.nc.Drain()
	}
}

// IsConnected returns true if the NATS connection is active.
func (c *Client) IsConnected() bool {
	return c.nc != nil && c.nc.IsConnected()
}
