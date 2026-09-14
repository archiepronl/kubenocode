package engine

import (
	"context"
	"io"
	"time"
)

// ─────────────────────────────────────────────
//  Unified Execution Graph Models
// ─────────────────────────────────────────────
// These structures represent the internal domain model of the execution engine.
// They decouple the engine from the Kubernetes API (v1alpha1.FullStackApplication)
// to allow for offline execution, testing, and schema evolution.

// ExecutionGraph represents the compiled directed acyclic graph ready for execution.
type ExecutionGraph struct {
	ID        string
	Nodes     map[string]*ExecutionNode
	EntryNode string
	CreatedAt time.Time
}

// ExecutionNode maps a v1alpha1.WorkflowNode into a rapid-access internal structure.
// It normalizes disparate CRD fields (scripts, images, resources) into a unified execution contract.
type ExecutionNode struct {
	ID            string
	Type          string // Format: "<category>/<adapter>"
	Label         string
	GroupID       string
	PackagingMode string
	Language      string

	// Configuration parameters resolved at compile time (secrets remain as references).
	Params    map[string]string
	SecretRef string

	// Payload/Execution definitions
	Image  string
	Script string

	// Directed Edges
	Outputs []string

	// MimeTypeHint provides an optional hint about expected output mime type.
	MimeTypeHint string
}

// ─────────────────────────────────────────────
//  MimeType Structure Handlers & Constants
// ─────────────────────────────────────────────

// Core IANA MimeType definitions used across the engine to route data
// through the Materializer adapters.
const (
	MimeTypeJSON         = "application/json"
	MimeTypeCSV          = "text/csv"
	MimeTypeXML          = "application/xml"
	MimeTypeOctetStream  = "application/octet-stream"
	MimeTypePlainText    = "text/plain"
	MimeTypeParquet      = "application/vnd.apache.parquet"
	MimeTypeKTTMInternal = "application/vnd.kttm.internal+json" // For internal control planes
)

// MimeHandler defines the contract for parsing and serializing raw non-normalized
// data streams within the pipeline.
type MimeHandler interface {
	// Accept checks if the handler supports the given MimeType.
	Accept(mimeType string) bool

	// Process reads the raw input stream, normalizes or validates the structure,
	// and returns a transformed stream along with any detected structural metadata.
	Process(ctx context.Context, mimeType string, r io.Reader) (io.ReadCloser, map[string]string, error)
}
