package nats_connector

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kubeworkflow/flowengine/core/connectors"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestNATSConnectorReadWrite(t *testing.T) {
	connector := New()
	if connector.Type() != connectorType || !json.Valid(connector.Schema()) {
		t.Fatal("NATS connector type or schema is invalid")
	}
	if _, err := connector.Read(context.Background(), connectors.ConnectorConfig{}); err == nil {
		t.Fatal("Read without subject succeeded")
	}
	result, err := connector.Read(context.Background(), connectors.ConnectorConfig{
		Params: map[string]string{paramSubject: "orders.new", paramQueueGroup: "workers", paramJetStream: "TRUE"},
	})
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if result.Envelope.MimeType != "application/x-nats-message" || result.Envelope.Tags["nats.subject"] != "orders.new" ||
		result.Envelope.Tags["nats.url"] != "nats://nats.flowengine.svc.cluster.local:4222" {
		t.Fatalf("unexpected read envelope: %+v", result.Envelope)
	}
	data, err := io.ReadAll(result.Stream)
	if err != nil {
		t.Fatalf("reading stream: %v", err)
	}
	_ = result.Stream.Close()
	if !strings.Contains(string(data), "ORD-4291") {
		t.Fatalf("unexpected NATS stream: %s", data)
	}

	if err := connector.Write(context.Background(), connectors.EnvelopeRef{}, strings.NewReader("payload"), connectors.ConnectorConfig{}); err == nil {
		t.Fatal("Write without subject succeeded")
	}
	if err := connector.Write(context.Background(), connectors.EnvelopeRef{}, failingReader{}, connectors.ConnectorConfig{
		Params: map[string]string{paramSubject: "orders.new"},
	}); err == nil {
		t.Fatal("Write with a failing reader succeeded")
	}
	if err := connector.Write(context.Background(), connectors.EnvelopeRef{}, strings.NewReader("payload"), connectors.ConnectorConfig{
		Params: map[string]string{paramSubject: "orders.new"}, Env: map[string]string{envNATSURL: "nats://localhost:4222"},
	}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
}
