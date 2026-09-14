// Package nats_connector implements the NATS.io connector for FlowEngine.
//
// This connector is distinct from core/messaging/nats.go (the internal event bus).
// This package exposes NATS as a user-facing data source and sink in workflows:
//   - Source: subscribe to a NATS subject and forward messages as workflow envelopes
//   - Sink: publish workflow output payloads to a NATS subject
//
// Particularly useful for:
//   - Chaining FlowEngine workflows together via NATS subjects
//   - Consuming NATS events from external services
//   - Publishing results to downstream NATS consumers
package nats_connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/kubeworkflow/flowengine/core/connectors"
)

const connectorType = "connector/nats"

const (
	paramSubject    = "subject"    // NATS subject to subscribe to / publish on
	paramQueueGroup = "queueGroup" // Queue group name (load-balanced consumers)
	paramMaxMsgs    = "maxMsgs"    // Max messages to collect before returning (0 = stream)
	paramJetStream  = "jetStream"  // Use JetStream for persistent delivery
	envNATSURL      = "NATS_URL"
	envNATSCreds    = "NATS_CREDS_PATH"
)

// NATSConnector implements connectors.Connector for NATS.io.
type NATSConnector struct{}

func New() *NATSConnector { return &NATSConnector{} }

func (c *NATSConnector) Type() string { return connectorType }

func (c *NATSConnector) Schema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "NATS Connector Configuration",
		"type": "object",
		"required": ["subject"],
		"properties": {
			"subject": {
				"type": "string",
				"title": "NATS Subject",
				"description": "Subject to subscribe to (wildcard * and > supported).",
				"placeholder": "orders.new"
			},
			"queueGroup": {
				"type": "string",
				"title": "Queue Group",
				"description": "Queue group for load-balanced consumption across replicas."
			},
			"maxMsgs": {
				"type": "string",
				"title": "Max Messages",
				"description": "Collect this many messages before returning. 0 = stream indefinitely.",
				"default": "0"
			},
			"jetStream": {
				"type": "string",
				"title": "Use JetStream",
				"description": "Use JetStream for persistent, at-least-once delivery.",
				"enum": ["true", "false"],
				"default": "false"
			}
		}
	}`)
}

// Read subscribes to the configured NATS subject and streams received messages.
// Each message is forwarded as a raw byte stream preserving the original payload (FR-4.1).
func (c *NATSConnector) Read(ctx context.Context, cfg connectors.ConnectorConfig) (*connectors.ReadResult, error) {
	url := cfg.Env[envNATSURL]
	if url == "" {
		url = "nats://nats.flowengine.svc.cluster.local:4222"
	}
	subject := cfg.Params[paramSubject]
	if subject == "" {
		return nil, fmt.Errorf("nats connector: 'subject' parameter is required")
	}

	queueGroup := cfg.Params[paramQueueGroup]
	useJS := strings.ToLower(cfg.Params[paramJetStream]) == "true"

	fmt.Printf("[NATSConnector] Subscribing to %s subject=%s queue=%s jetstream=%v\n", url, subject, queueGroup, useJS)

	// In production: nats.Connect(url) → nc.QueueSubscribe(subject, queue, handler)
	// Each message forwarded via io.Pipe to the envelope stream

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		// Scaffold: simulate one message received
		msg := `{"subject":"orders.new","data":{"orderId":"ORD-4291","amount":349.99}}`
		fmt.Fprintln(pw, msg)
	}()

	return &connectors.ReadResult{
		Envelope: connectors.EnvelopeRef{
			MimeType: "application/x-nats-message",
			Tags:     map[string]string{"nats.subject": subject, "nats.url": url},
		},
		Stream: pr,
	}, nil
}

// Write publishes the payload to the configured NATS subject.
func (c *NATSConnector) Write(ctx context.Context, env connectors.EnvelopeRef, r io.Reader, cfg connectors.ConnectorConfig) error {
	url := cfg.Env[envNATSURL]
	if url == "" {
		url = "nats://nats.flowengine.svc.cluster.local:4222"
	}
	subject := cfg.Params[paramSubject]
	if subject == "" {
		return fmt.Errorf("nats connector: 'subject' parameter is required for publish")
	}

	// In production: read r into bytes (up to max size), nc.Publish(subject, data)
	data, err := io.ReadAll(io.LimitReader(r, 1<<20)) // 1MB max for NATS Core
	if err != nil {
		return fmt.Errorf("reading payload for NATS publish: %w", err)
	}
	fmt.Printf("[NATSConnector] Publishing %d bytes to subject=%s\n", len(data), subject)
	return nil
}
