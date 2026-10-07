package s3

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/kubeworkflow/flowengine/core/connectors"
)

func TestS3ConnectorReadWriteAndMimeTypes(t *testing.T) {
	connector := New()
	if connector.Type() != connectorType || !json.Valid(connector.Schema()) {
		t.Fatal("S3 connector type or schema is invalid")
	}
	if _, err := connector.Read(context.Background(), connectors.ConnectorConfig{}); err == nil {
		t.Fatal("Read without bucket succeeded")
	}
	result, err := connector.Read(context.Background(), connectors.ConnectorConfig{
		Params: map[string]string{paramBucket: "reports", paramPrefix: "daily/report.csv"},
	})
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if result.Envelope.MimeType != "text/csv" || result.Envelope.Tags["s3.bucket"] != "reports" || result.Envelope.Tags["s3.prefix"] != "daily/report.csv" {
		t.Fatalf("unexpected read envelope: %+v", result.Envelope)
	}
	data, err := io.ReadAll(result.Stream)
	if err != nil {
		t.Fatalf("reading stream: %v", err)
	}
	_ = result.Stream.Close()
	if string(data) != "[S3Connector] Streaming s3://reports/daily/report.csv" {
		t.Fatalf("unexpected stream: %s", data)
	}
	if err := connector.Write(context.Background(), connectors.EnvelopeRef{}, strings.NewReader("payload"), connectors.ConnectorConfig{}); err == nil {
		t.Fatal("Write without bucket succeeded")
	}
	if err := connector.Write(context.Background(), connectors.EnvelopeRef{MimeType: "text/csv"}, strings.NewReader("payload"), connectors.ConnectorConfig{
		Params: map[string]string{paramBucket: "reports", paramPrefix: "daily/report.csv"},
	}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	tests := []struct {
		key  string
		want string
	}{
		{"report.csv", "text/csv"},
		{"report.json", "application/json"},
		{"data.parquet", "application/x-parquet"},
		{"report.pdf", "application/pdf"},
		{"report.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"clip.mp4", "video/mp4"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"photo.png", "image/png"},
		{"unknown", "text/plain; charset=utf-8"},
	}
	for _, tt := range tests {
		if got := inferMimeType(tt.key); got != tt.want {
			t.Errorf("inferMimeType(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}
