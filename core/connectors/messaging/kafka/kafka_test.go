package kafka

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/kubeworkflow/flowengine/core/connectors"
)

func TestKafkaConnectorReadWriteAndHelpers(t *testing.T) {
	connector := New()
	if connector.Type() != connectorType || !json.Valid(connector.Schema()) {
		t.Fatal("Kafka connector type or schema is invalid")
	}
	if _, err := connector.Read(context.Background(), connectors.ConnectorConfig{}); err == nil {
		t.Fatal("Read without brokers succeeded")
	}
	if _, err := connector.Read(context.Background(), connectors.ConnectorConfig{Env: map[string]string{envBootstrapServers: "broker:9092"}}); err == nil {
		t.Fatal("Read without topic succeeded")
	}

	result, err := connector.Read(context.Background(), connectors.ConnectorConfig{
		Params: map[string]string{paramTopic: "orders"},
		Env:    map[string]string{envBootstrapServers: "broker:9092", paramGroupID: "worker-group"},
	})
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if result.Envelope.MimeType != "application/x-kafka-records" || result.Envelope.Tags["kafka.topic"] != "orders" ||
		result.Envelope.Tags["kafka.group"] != "worker-group" || result.Envelope.Tags["kafka.brokers"] != "broker:9092" {
		t.Fatalf("unexpected read envelope: %+v", result.Envelope)
	}
	data, err := io.ReadAll(result.Stream)
	if err != nil {
		t.Fatalf("reading stream: %v", err)
	}
	_ = result.Stream.Close()
	if strings.Count(string(data), "offset") != 2 {
		t.Fatalf("stream does not contain both records: %s", data)
	}

	defaultGroup, err := connector.Read(context.Background(), connectors.ConnectorConfig{
		Params: map[string]string{paramTopic: "orders"}, Env: map[string]string{envBootstrapServers: "broker:9092"},
	})
	if err != nil {
		t.Fatalf("Read() with default group error = %v", err)
	}
	if defaultGroup.Envelope.Tags["kafka.group"] != "flowengine-consumer" {
		t.Fatalf("default group = %q", defaultGroup.Envelope.Tags["kafka.group"])
	}
	_ = defaultGroup.Stream.Close()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	canceledResult, err := connector.Read(canceled, connectors.ConnectorConfig{
		Params: map[string]string{paramTopic: "orders"}, Env: map[string]string{envBootstrapServers: "broker:9092"},
	})
	if err != nil {
		t.Fatalf("Read(canceled context) error = %v", err)
	}
	if _, err := io.ReadAll(canceledResult.Stream); err == nil {
		t.Fatal("canceled stream completed without an error")
	}
	_ = canceledResult.Stream.Close()

	if err := connector.Write(context.Background(), connectors.EnvelopeRef{}, strings.NewReader("payload"), connectors.ConnectorConfig{}); err == nil {
		t.Fatal("Write without topic succeeded")
	}
	if err := connector.Write(context.Background(), connectors.EnvelopeRef{}, strings.NewReader("payload"), connectors.ConnectorConfig{
		Params: map[string]string{paramTopic: "orders"}, Env: map[string]string{envBootstrapServers: "broker:9092"},
	}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if got := coalesce("  ", "value", "later"); got != "value" {
		t.Fatalf("coalesce() = %q", got)
	}
	if got := coalesce(" \t"); got != "" {
		t.Fatalf("coalesce(blank) = %q", got)
	}
}