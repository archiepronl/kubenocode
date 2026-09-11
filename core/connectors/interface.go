// Package connectors defines the universal Connector interface and the ConnectorRegistry
// that powers FR-3 (Universal Input Connector Architecture).
//
// Every data source and sink in FlowEngine implements this single interface.
// This enables the operator to discover available connectors and expose their
// configuration schemas to the UI editor for auto-generated config forms.
//
// The interface is designed for raw payload preservation (FR-4):
//   - Read() returns an io.Reader of raw bytes — never pre-converted to JSON
//   - The Envelope carries only metadata
//   - Write() consumes an io.Reader without forcing serialization
package connectors

import (
	"context"
	"encoding/json"
	"io"
)

// ─────────────────────────────────────────────
//  Connector configuration
// ─────────────────────────────────────────────

// ConnectorConfig holds runtime configuration for a connector instance.
// Non-sensitive params come from WorkflowNode.Params (stored in CRD spec).
// Sensitive credentials are injected via environment variables from the SecretRef.
type ConnectorConfig struct {
	// Params holds non-sensitive configuration key-value pairs.
	// These map directly to WorkflowNode.Params in the CRD spec.
	Params map[string]string

	// Env holds environment variables injected from Kubernetes Secrets via SecretRef.
	// Connectors should read credentials from Env, never from Params.
	Env map[string]string
}

// Get returns a config value, checking Params first then Env.
func (c ConnectorConfig) Get(key string) string {
	if v, ok := c.Params[key]; ok {
		return v
	}
	return c.Env[key]
}

// ─────────────────────────────────────────────
//  Connector interface
// ─────────────────────────────────────────────

// ReadResult is returned by Connector.Read().
type ReadResult struct {
	// Envelope carries the metadata for the payload (mimeType, size, location).
	// The actual bytes are available via Stream.
	Envelope EnvelopeRef

	// Stream is the raw byte stream of the payload.
	// The caller is responsible for closing this reader.
	// The bytes must NOT be normalized or converted — they flow as-is (FR-4.1).
	Stream io.ReadCloser
}

// EnvelopeRef is a minimal envelope used by connectors before the full
// materializer.Envelope is constructed by the engine.
type EnvelopeRef struct {
	MimeType    string
	PayloadSize int64
	Tags        map[string]string
}

// Connector is the universal interface for all data sources and sinks in FlowEngine.
// Implementing this interface allows a new connector to be discovered, configured,
// and used in workflows across all three user tiers.
type Connector interface {
	// Read opens a connection to the source and returns the raw payload stream.
	// Implementations must not buffer the entire payload in memory —
	// use streaming reads for large payloads (FR-1.4, FR-4.4).
	Read(ctx context.Context, cfg ConnectorConfig) (*ReadResult, error)

	// Write sends a raw payload stream to the sink.
	// Implementations must stream the bytes without loading them all into memory.
	Write(ctx context.Context, env EnvelopeRef, r io.Reader, cfg ConnectorConfig) error

	// Schema returns the JSON Schema for this connector's configuration parameters.
	// The UI editor uses this schema to auto-generate the connector's config form.
	Schema() json.RawMessage

	// Type returns the connector's type identifier (e.g., "connector/s3").
	Type() string
}

// ─────────────────────────────────────────────
//  Registry
// ─────────────────────────────────────────────

// Registry maintains a catalog of all registered connectors.
// It is used by the operator and UI to discover available connector types.
type Registry struct {
	connectors map[string]Connector
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{connectors: make(map[string]Connector)}
}

// Register adds a Connector to the registry.
// If a connector with the same Type() already exists, it is replaced.
func (r *Registry) Register(c Connector) {
	r.connectors[c.Type()] = c
}

// Get retrieves a Connector by its type string.
func (r *Registry) Get(connectorType string) (Connector, bool) {
	c, ok := r.connectors[connectorType]
	return c, ok
}

// All returns all registered connectors.
func (r *Registry) All() []Connector {
	result := make([]Connector, 0, len(r.connectors))
	for _, c := range r.connectors {
		result = append(result, c)
	}
	return result
}

// Schemas returns a map of connector type → JSON Schema for all registered connectors.
// Used by the UI to build the connector configuration panel.
func (r *Registry) Schemas() map[string]json.RawMessage {
	schemas := make(map[string]json.RawMessage, len(r.connectors))
	for t, c := range r.connectors {
		schemas[t] = c.Schema()
	}
	return schemas
}
