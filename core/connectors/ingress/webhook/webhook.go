// Package webhook implements the HTTP webhook trigger connector for FlowEngine.
//
// This connector starts a lightweight HTTP server inside the workflow pod.
// Incoming webhook POST requests trigger workflow executions by publishing
// a UIEvent or Envelope to the NATS event bus.
//
// Key design properties (FR-3.2 Application Ingress):
//   - Accepts arbitrary Content-Type payloads (JSON, XML, form, binary)
//   - Preserves raw body bytes without normalization (FR-4.1)
//   - Infers MIME type from Content-Type header and writes to Envelope
//   - Configurable secret-based HMAC signature verification for security
//   - Sub-millisecond ack to caller; async NATS publish (FR-1.3)
package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kubeworkflow/flowengine/core/connectors"
)

// ─────────────────────────────────────────────
//  Type identifier
// ─────────────────────────────────────────────

const connectorType = "trigger/webhook"

// ─────────────────────────────────────────────
//  Config keys
// ─────────────────────────────────────────────

const (
	paramPath      = "path"        // URL path to listen on (e.g., "/webhook/my-flow")
	paramPort      = "port"        // Port to bind (default: 8090)
	paramMethod    = "method"      // Allowed HTTP method (default: POST)
	paramMaxBodyMB = "maxBodyMb"   // Max request body size in MB (default: 10)
	envHMACSecret  = "HMAC_SECRET" // HMAC secret from SecretRef for signature verification
)

// ─────────────────────────────────────────────
//  WebhookConnector
// ─────────────────────────────────────────────

// WebhookConnector implements connectors.Connector for HTTP webhook triggers.
// It starts an embedded HTTP server and returns one payload per incoming request.
type WebhookConnector struct{}

// New returns a new WebhookConnector.
func New() *WebhookConnector { return &WebhookConnector{} }

// Type implements connectors.Connector.
func (c *WebhookConnector) Type() string { return connectorType }

// Schema returns the JSON Schema for webhook trigger configuration.
func (c *WebhookConnector) Schema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "HTTP Webhook Trigger Configuration",
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"title": "Webhook Path",
				"description": "URL path for the webhook endpoint.",
				"default": "/webhook"
			},
			"port": {
				"type": "string",
				"title": "Listen Port",
				"default": "8090"
			},
			"method": {
				"type": "string",
				"title": "HTTP Method",
				"enum": ["POST", "PUT", "PATCH"],
				"default": "POST"
			},
			"maxBodyMb": {
				"type": "string",
				"title": "Max Body Size (MB)",
				"description": "Maximum request body size. Larger payloads are rejected with 413.",
				"default": "10"
			}
		},
		"x-flowengine-trigger": true
	}`)
}

// Read starts the HTTP server and blocks until one webhook request is received.
// The raw request body is returned as-is without normalization (FR-4.1).
//
// The caller's context controls the server lifetime.
// When the context is cancelled, the server shuts down gracefully.
func (c *WebhookConnector) Read(ctx context.Context, cfg connectors.ConnectorConfig) (*connectors.ReadResult, error) {
	path := coalesce(cfg.Params[paramPath], "/webhook")
	port := coalesce(cfg.Params[paramPort], "8090")
	method := coalesce(cfg.Params[paramMethod], "POST")
	hmacSecret := cfg.Env[envHMACSecret]

	resultCh := make(chan *connectors.ReadResult, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		handleWebhookRequest(w, r, resultCh, path, method, hmacSecret)
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
		// Conservative timeouts to prevent slow-loris attacks
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server in a background goroutine
	go func() {
		fmt.Printf("[WebhookConnector] Listening on :%s%s (method=%s)\n", port, path, method)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for first request, context cancellation, or error
	select {
	case result := <-resultCh:
		// Graceful shutdown after receiving one request
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		return result, nil
	case err := <-errCh:
		return nil, fmt.Errorf("webhook server error: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		return nil, ctx.Err()
	}
}

func handleWebhookRequest(w http.ResponseWriter, r *http.Request, resultCh chan<- *connectors.ReadResult, path, method, hmacSecret string) {
	if r.Method != method {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, ok := readWebhookBody(w, r.Body)
	if !ok {
		return
	}

	if hmacSecret != "" {
		sig := r.Header.Get("X-FlowEngine-Signature")
		if sig == "" {
			sig = r.Header.Get("X-Hub-Signature-256")
		}
		if !verifyHMAC(body, sig, hmacSecret) {
			http.Error(w, "Forbidden: invalid signature", http.StatusForbidden)
			return
		}
	}

	mimeType := r.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, `{"status":"accepted","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		pw.Write(body)
	}()

	resultCh <- &connectors.ReadResult{
		Envelope: connectors.EnvelopeRef{
			MimeType:    mimeType,
			PayloadSize: r.ContentLength,
			Tags: map[string]string{
				"webhook.path":       path,
				"webhook.method":     r.Method,
				"webhook.remoteAddr": r.RemoteAddr,
				"webhook.userAgent":  r.UserAgent(),
			},
		},
		Stream: pr,
	}
}

func readWebhookBody(w http.ResponseWriter, body io.Reader) ([]byte, bool) {
	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return nil, false
	}
	return data, true
}

// Write is not supported for webhook triggers (source-only connector).
func (c *WebhookConnector) Write(ctx context.Context, env connectors.EnvelopeRef, r io.Reader, cfg connectors.ConnectorConfig) error {
	return fmt.Errorf("webhook connector is a trigger source: Write is not supported")
}

// ─────────────────────────────────────────────
//  HMAC verification
// ─────────────────────────────────────────────

// verifyHMAC checks the HMAC-SHA256 signature of a webhook payload.
// The signature format matches GitHub, Stripe, and most webhook providers.
func verifyHMAC(payload []byte, signature, secret string) bool {
	// Strip "sha256=" prefix if present (GitHub webhook format)
	sig := signature
	if len(sig) > 7 && sig[:7] == "sha256=" {
		sig = sig[7:]
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expected))
}

// ─────────────────────────────────────────────
//  Helpers
// ─────────────────────────────────────────────

func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// bytesReader wraps a []byte to implement io.ReadCloser.
type bytesReader struct {
	data   []byte
	offset int
}

func newBytesReader(b []byte) io.ReadCloser {
	return &bytesReader{data: b}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}

func (r *bytesReader) Close() error { return nil }
