package postgres

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/kubeworkflow/flowengine/core/connectors"
)

func TestPostgresConnectorReadWriteAndHelpers(t *testing.T) {
	connector := New()
	if connector.Type() != connectorType || !json.Valid(connector.Schema()) {
		t.Fatal("Postgres connector type or schema is invalid")
	}
	if _, err := connector.Read(context.Background(), connectors.ConnectorConfig{}); err == nil {
		t.Fatal("Read without query succeeded")
	}
	if _, err := connector.Read(context.Background(), connectors.ConnectorConfig{Params: map[string]string{paramQuery: "SELECT 1"}}); err == nil {
		t.Fatal("Read without PGUSER succeeded")
	}

	query := "SELECT " + strings.Repeat("x", 110)
	result, err := connector.Read(context.Background(), connectors.ConnectorConfig{
		Params: map[string]string{paramQuery: query, paramHost: "param-host", paramDatabase: "param-db", paramPort: "6000"},
		Env:    map[string]string{envPGUser: "user", envPGPassword: "password", envPGHost: "secret-host", envPGDatabase: "secret-db"},
	})
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if result.Envelope.MimeType != "application/x-arrow-ipc-stream" || result.Envelope.PayloadSize != -1 ||
		result.Envelope.Tags["postgres.host"] != "secret-host" || result.Envelope.Tags["postgres.database"] != "secret-db" ||
		len(result.Envelope.Tags["postgres.query"]) != 103 {
		t.Fatalf("unexpected read envelope: %+v", result.Envelope)
	}
	data, err := io.ReadAll(result.Stream)
	if err != nil {
		t.Fatalf("reading stream: %v", err)
	}
	if err := result.Stream.Close(); err != nil {
		t.Errorf("closing stream: %v", err)
	}
	if !strings.Contains(string(data), `"Alice"`) || !strings.Contains(string(data), `"Bob"`) {
		t.Fatalf("unexpected stream: %s", data)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	canceledResult, err := connector.Read(canceled, connectors.ConnectorConfig{
		Params: map[string]string{paramQuery: "SELECT 1"}, Env: map[string]string{envPGUser: "user"},
	})
	if err != nil {
		t.Fatalf("Read(canceled context) error = %v", err)
	}
	if _, err := io.ReadAll(canceledResult.Stream); err == nil {
		t.Fatal("canceled stream completed without an error")
	}
	_ = canceledResult.Stream.Close()

	if err := connector.Write(context.Background(), connectors.EnvelopeRef{}, strings.NewReader("rows"), connectors.ConnectorConfig{}); err == nil {
		t.Fatal("Write without query succeeded")
	}
	if err := connector.Write(context.Background(), connectors.EnvelopeRef{}, strings.NewReader("rows"), connectors.ConnectorConfig{Params: map[string]string{paramQuery: "COPY orders"}}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if got := coalesce("", "value", "later"); got != "value" {
		t.Fatalf("coalesce() = %q, want first nonempty value", got)
	}
	if got := coalesce("", ""); got != "" {
		t.Fatalf("coalesce(empty) = %q, want empty string", got)
	}
	if got := truncate("  short  ", 20); got != "short" {
		t.Fatalf("truncate(short) = %q", got)
	}
	if got := truncate("123456", 3); got != "123..." {
		t.Fatalf("truncate(long) = %q, want 123...", got)
	}
}