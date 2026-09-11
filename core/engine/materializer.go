// Package materializer implements FR-4: Raw Payload Preservation (Zero Conversion Strategy).
//
// The core philosophy: the engine NEVER force-normalizes raw inbound object streams
// into JSON or any other uniform structure. Objects (videos, PDFs, blobs, binary data)
// are passed between workflow nodes as opaque byte streams, wrapped in a lightweight
// Envelope that carries only metadata — never the payload itself over the event bus.
//
// Data Transfer Decision Matrix:
//
//	< 100MB, same pod     → /dev/shm  (lowest latency)
//	< 1GB, sequential     → emptyDir  (spill-to-disk safe)
//	> 1GB or cross-pod    → MinIO     (persistent, cross-node)
//
// The NATS event bus carries ONLY the Envelope JSON (~200 bytes).
// The actual bytes live in the PayloadLocation path or object store.
package materializer

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kubeworkflow/flowengine/core/engine/adapters"
)

// ─────────────────────────────────────────────
//  Envelope — the metadata wrapper
// ─────────────────────────────────────────────

// StorageBackend identifies which physical storage tier holds the raw payload.
type StorageBackend string

const (
	// StorageShm uses Linux shared memory (/dev/shm). Fastest, same-pod only.
	StorageShm StorageBackend = "shm"

	// StorageEmptyDir uses a Kubernetes emptyDir volume mounted across init and step containers.
	StorageEmptyDir StorageBackend = "emptydir"

	// StorageMinIO routes the payload to the in-cluster MinIO (S3-compatible) object store.
	// Used for files > 1GB or when the payload must survive across multiple pods/steps.
	StorageMinIO StorageBackend = "minio"

	// StorageInline is used only for very small payloads (< 4KB) that are embedded
	// directly in the envelope for diagnostic / testing purposes.
	StorageInline StorageBackend = "inline"
)

// Envelope is the lightweight metadata carrier exchanged over the NATS event bus.
// It never contains the raw payload bytes directly (except StorageInline for tiny data).
// All downstream nodes read from PayloadLocation using the indicated StorageBackend.
type Envelope struct {
	// ExecutionID uniquely identifies the workflow execution run.
	ExecutionID string `json:"executionId"`

	// SourceNodeID is the DAG node that produced this payload.
	SourceNodeID string `json:"sourceNodeId"`

	// TargetNodeID is the DAG node that should consume this payload.
	TargetNodeID string `json:"targetNodeId,omitempty"`

	// MimeType is the IANA media type of the raw payload (e.g., "video/mp4", "application/pdf").
	// The Materializer uses this to select the correct adapter factory.
	MimeType string `json:"mimeType"`

	// PayloadSize is the total byte size of the raw payload. Used for routing decisions.
	PayloadSize int64 `json:"payloadSize"`

	// StorageBackend indicates where the raw payload is physically stored.
	StorageBackend StorageBackend `json:"storageBackend"`

	// PayloadLocation is the path or URI to the raw payload:
	//   shm:     /dev/shm/flowengine/<executionId>/<nodeId>.bin
	//   emptydir: /data/flowengine/<executionId>/<nodeId>.bin
	//   minio:   s3://flowengine-payloads/<executionId>/<nodeId>
	//   inline:  base64-encoded bytes (max 4KB)
	PayloadLocation string `json:"payloadLocation"`

	// Tags carries arbitrary node-level metadata (user-defined labels, security tags, etc.).
	Tags map[string]string `json:"tags,omitempty"`

	// CreatedAt records when this envelope was generated.
	CreatedAt time.Time `json:"createdAt"`

	// ChecksumSHA256 is the SHA-256 hex digest of the raw payload for integrity verification.
	// +optional
	ChecksumSHA256 string `json:"checksumSha256,omitempty"`
}

// ─────────────────────────────────────────────
//  Storage routing constants
// ─────────────────────────────────────────────

const (
	// shmThresholdBytes: payloads below this size are routed to /dev/shm (same-pod only).
	shmThresholdBytes int64 = 100 * 1024 * 1024 // 100 MB

	// emptyDirThresholdBytes: payloads below this size use emptyDir volumes.
	emptyDirThresholdBytes int64 = 1024 * 1024 * 1024 // 1 GB

	// paths
	shmBasePath      = "/dev/shm/flowengine"
	emptyDirBasePath = "/data/flowengine"
	minioBucket      = "flowengine-payloads"
)

// RouteStorage determines the optimal storage backend for a given payload size.
// This implements the tiered data transfer strategy from FR-4.4.
func RouteStorage(payloadSizeBytes int64, crossPod bool) StorageBackend {
	if crossPod || payloadSizeBytes >= emptyDirThresholdBytes {
		return StorageMinIO
	}
	if payloadSizeBytes < shmThresholdBytes {
		return StorageShm
	}
	return StorageEmptyDir
}

// PayloadPath returns the canonical filesystem or object-store path for an envelope.
func PayloadPath(backend StorageBackend, executionID, nodeID string) string {
	switch backend {
	case StorageShm:
		return fmt.Sprintf("%s/%s/%s.bin", shmBasePath, executionID, nodeID)
	case StorageEmptyDir:
		return fmt.Sprintf("%s/%s/%s.bin", emptyDirBasePath, executionID, nodeID)
	case StorageMinIO:
		return fmt.Sprintf("s3://%s/%s/%s", minioBucket, executionID, nodeID)
	default:
		return ""
	}
}

// ─────────────────────────────────────────────
//  Processor interface
// ─────────────────────────────────────────────

// Processor is the interface implemented by all materialization adapters.
// Each adapter is responsible for reading/transforming a specific mime type.
type Processor interface {
	// Process reads from the raw io.Reader and executes the adapter's transformation logic.
	// It returns an updated Envelope (with any output metadata changes) and the processed output stream.
	Process(ctx context.Context, env *Envelope, r io.Reader) (*Envelope, io.ReadCloser, error)

	// MimeTypes returns the list of IANA mime type prefixes this adapter handles.
	MimeTypes() []string
}

// ─────────────────────────────────────────────
//  Materializer — the adapter factory
// ─────────────────────────────────────────────

// Materializer dispatches raw payload streams to the correct processing adapter
// based on the Envelope's MimeType field. It implements the pluggable factory
// pattern from FR-4.3 (Embedded Materialization Adapters).
type Materializer struct {
	adapters []Processor
}

// New returns a Materializer pre-loaded with all built-in adapters.
// Custom adapters can be registered via Register().
func New() *Materializer {
	m := &Materializer{}
	// Register built-in adapters in priority order (most specific first)
	m.Register(adapters.NewPDFAdapter())
	m.Register(adapters.NewVideoAdapter())
	m.Register(adapters.NewImageAdapter())
	m.Register(adapters.NewAudioAdapter())
	m.Register(adapters.NewExcelAdapter())
	m.Register(adapters.NewArrowAdapter())
	m.Register(adapters.NewPassthroughAdapter())
	return m
}

// Register adds a custom Processor to the Materializer.
// Adapters are evaluated in registration order; the first match wins.
func (m *Materializer) Register(p Processor) {
	m.adapters = append(m.adapters, p)
}

// Materialize selects the correct adapter for the given Envelope and dispatches
// the raw byte stream to it. It implements the core of FR-4 (Zero Conversion Strategy).
//
// The caller is responsible for closing the returned io.ReadCloser.
func (m *Materializer) Materialize(ctx context.Context, env *Envelope, r io.Reader) (*Envelope, io.ReadCloser, error) {
	adapter := m.selectAdapter(env.MimeType)
	if adapter == nil {
		// Fallback: passthrough — bytes flow unchanged
		return env, io.NopCloser(r), nil
	}
	return adapter.Process(ctx, env, r)
}

// selectAdapter finds the first registered adapter that handles the given mimeType.
func (m *Materializer) selectAdapter(mimeType string) Processor {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	for _, a := range m.adapters {
		for _, supported := range a.MimeTypes() {
			if strings.HasPrefix(mimeType, supported) {
				return a
			}
		}
	}
	return nil
}
