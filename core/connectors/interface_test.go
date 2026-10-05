package connectors_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/kubeworkflow/flowengine/core/connectors"
)

type testConnector struct {
	connectorType string
	schema        json.RawMessage
}

func (c *testConnector) Read(context.Context, connectors.ConnectorConfig) (*connectors.ReadResult, error) {
	return nil, nil
}

func (c *testConnector) Write(context.Context, connectors.EnvelopeRef, io.Reader, connectors.ConnectorConfig) error {
	return nil
}

func (c *testConnector) Schema() json.RawMessage { return c.schema }

func (c *testConnector) Type() string { return c.connectorType }

func TestConnectorConfigGet(t *testing.T) {
	cfg := connectors.ConnectorConfig{
		Params: map[string]string{"shared": "param", "param-only": "value", "empty": ""},
		Env:    map[string]string{"shared": "env", "env-only": "value"},
	}

	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "params take precedence", key: "shared", want: "param"},
		{name: "parameter value", key: "param-only", want: "value"},
		{name: "environment fallback", key: "env-only", want: "value"},
		{name: "present empty parameter takes precedence", key: "empty", want: ""},
		{name: "missing value", key: "missing", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.Get(tt.key); got != tt.want {
				t.Fatalf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	registry := connectors.NewRegistry()
	if got := registry.All(); len(got) != 0 {
		t.Fatalf("empty registry has %d connectors, want 0", len(got))
	}
	if _, ok := registry.Get("missing"); ok {
		t.Fatal("Get found an unregistered connector")
	}
	if got := registry.Schemas(); len(got) != 0 {
		t.Fatalf("empty registry has %d schemas, want 0", len(got))
	}

	first := &testConnector{connectorType: "source/test", schema: json.RawMessage(`{"title":"first"}`)}
	second := &testConnector{connectorType: "sink/test", schema: json.RawMessage(`{"title":"second"}`)}
	replacement := &testConnector{connectorType: "source/test", schema: json.RawMessage(`{"title":"replacement"}`)}
	registry.Register(first)
	registry.Register(second)
	registry.Register(replacement)

	got, ok := registry.Get("source/test")
	if !ok || got != replacement {
		t.Fatalf("Get(source/test) = (%v, %t), want replacement", got, ok)
	}

	all := registry.All()
	if len(all) != 2 {
		t.Fatalf("All returned %d connectors, want 2", len(all))
	}
	types := map[string]bool{}
	for _, connector := range all {
		types[connector.Type()] = true
	}
	if !types["source/test"] || !types["sink/test"] {
		t.Fatalf("All returned unexpected connector types: %v", types)
	}

	schemas := registry.Schemas()
	if got := string(schemas["source/test"]); got != string(replacement.schema) {
		t.Errorf("source schema = %s, want %s", got, replacement.schema)
	}
	if got := string(schemas["sink/test"]); got != string(second.schema) {
		t.Errorf("sink schema = %s, want %s", got, second.schema)
	}
}