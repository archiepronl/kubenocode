package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestNATSSubjectsAndDefaults(t *testing.T) {
	if got := EnvelopeSubject("team", "orders", "sink"); got != "flowengine.team.orders.sink.envelope" {
		t.Errorf("EnvelopeSubject() = %q", got)
	}
	if got := UIEventsSubject("team", "session"); got != "flowengine.team.ui.session.events" {
		t.Errorf("UIEventsSubject() = %q", got)
	}
	if got := TelemetrySubject("team", "worker"); got != "flowengine.team.telemetry.worker.metrics" {
		t.Errorf("TelemetrySubject() = %q", got)
	}
	if got := StatusSubject("team", "orders"); got != "flowengine.team.control.orders.status" {
		t.Errorf("StatusSubject() = %q", got)
	}
	defaults := DefaultConfig()
	if defaults.URL != nats.DefaultURL || defaults.MaxReconnects != -1 || defaults.ReconnectWait != 2*time.Second {
		t.Fatalf("unexpected NATS defaults: %+v", defaults)
	}
	client := &Client{}
	if client.IsConnected() {
		t.Fatal("empty client reports connected")
	}
	client.Close()
}

func TestNATSClientAgainstDisposableJetStream(t *testing.T) {
	t.Setenv("DOCKER_API_VERSION", "1.41")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	name := fmt.Sprintf("kttm-nats-test-%d", os.Getpid())
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Name:         name,
			Image:        "nats:2.10-alpine",
			ExposedPorts: []string{"4222/tcp"},
			Cmd:          []string{"-js"},
			WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start disposable NATS server: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("terminate disposable NATS server: %v", err)
		}
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("get NATS host: %v", err)
	}
	port, err := container.MappedPort(ctx, "4222/tcp")
	if err != nil {
		t.Fatalf("get NATS port: %v", err)
	}
	clientURL := fmt.Sprintf("nats://%s:%s", host, port.Port())
	client, err := NewClient(ctx, Config{URL: clientURL, MaxReconnects: 0, ReconnectWait: 10 * time.Millisecond})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	t.Cleanup(client.Close)
	if !client.IsConnected() {
		t.Fatal("new NATS client is not connected")
	}
	handleNATSReconnect(client.nc)
	if err := client.ensureStream(ctx); err != nil {
		t.Fatalf("repeat ensureStream() error = %v", err)
	}
	credentialPath := filepath.Join(t.TempDir(), "invalid.creds")
	if err := os.WriteFile(credentialPath, []byte("not a NATS credentials file"), 0o600); err != nil {
		t.Fatalf("write invalid credentials: %v", err)
	}
	if _, err := NewClient(ctx, Config{URL: clientURL, CredentialsFile: credentialPath, MaxReconnects: 0}); err == nil {
		t.Fatal("NewClient() accepted invalid credentials")
	}
	canceledSetup, cancelSetup := context.WithCancel(context.Background())
	cancelSetup()
	if _, err := NewClient(canceledSetup, Config{URL: clientURL, MaxReconnects: 0}); err == nil || !strings.Contains(err.Error(), "ensuring JetStream stream") {
		t.Fatalf("NewClient(canceled setup) error = %v", err)
	}
	jetStreamErr := errors.New("JetStream setup failed")
	if _, err := newClient(ctx, Config{URL: clientURL, MaxReconnects: 0}, func(*nats.Conn) (jetstream.JetStream, error) {
		return nil, jetStreamErr
	}); !errors.Is(err, jetStreamErr) {
		t.Fatalf("newClient() error = %v, want JetStream setup error", err)
	}

	workflowID := "orders-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	envelopeReceived := make(chan []byte, 1)
	if err := client.SubscribeEnvelopes(ctx, "team", workflowID, func(data []byte) { envelopeReceived <- data }); err != nil {
		t.Fatalf("SubscribeEnvelopes() error = %v", err)
	}
	envelope := map[string]string{"mimeType": "application/json", "id": "event-1"}
	if err := client.PublishEnvelope(ctx, "team", workflowID, "source", envelope); err != nil {
		t.Fatalf("PublishEnvelope() error = %v", err)
	}
	select {
	case data := <-envelopeReceived:
		var decoded map[string]string
		if err := json.Unmarshal(data, &decoded); err != nil || decoded["id"] != "event-1" {
			t.Fatalf("envelope message = %s (decode error %v)", data, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive JetStream envelope")
	}

	uiReceived := make(chan UIEvent, 1)
	uiSub, err := client.SubscribeUIEvents("team", workflowID, func(event UIEvent) { uiReceived <- event })
	if err != nil {
		t.Fatalf("SubscribeUIEvents() error = %v", err)
	}
	t.Cleanup(func() { _ = uiSub.Unsubscribe() })
	if err := client.nc.Flush(); err != nil {
		t.Fatalf("flush UI subscription: %v", err)
	}
	if err := client.PublishUIEvent(ctx, "team", UIEvent{SessionID: workflowID, WorkflowID: workflowID, TriggerID: "submit", Payload: map[string]string{"name": "Ada"}}); err != nil {
		t.Fatalf("PublishUIEvent() error = %v", err)
	}
	select {
	case event := <-uiReceived:
		if event.TriggerID != "submit" || event.Payload["name"] != "Ada" || event.Timestamp.IsZero() {
			t.Fatalf("unexpected UI event: %+v", event)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive UI event")
	}

	if err := client.PublishStatus(ctx, StatusUpdate{WorkflowID: workflowID, Namespace: "team", Phase: "Running"}); err != nil {
		t.Fatalf("PublishStatus() error = %v", err)
	}
	if err := client.PublishTelemetry(ctx, "team", TelemetryEvent{NodeID: "source", WorkflowID: workflowID, CPUPercent: 50}); err != nil {
		t.Fatalf("PublishTelemetry() error = %v", err)
	}
	if err := client.PublishEnvelope(ctx, "team", workflowID, "bad", make(chan int)); err == nil || !strings.Contains(err.Error(), "marshalling envelope") {
		t.Fatalf("PublishEnvelope(unmarshalable) error = %v", err)
	}

	canceled, cancelPublish := context.WithCancel(context.Background())
	cancelPublish()
	if err := client.PublishEnvelope(canceled, "team", workflowID, "canceled", envelope); err == nil {
		t.Fatal("PublishEnvelope() succeeded with canceled context")
	}
	if err := client.PublishStatus(canceled, StatusUpdate{WorkflowID: workflowID, Namespace: "team"}); err == nil {
		t.Fatal("PublishStatus() succeeded with canceled context")
	}
	if err := client.PublishTelemetry(canceled, "team", TelemetryEvent{NodeID: "source"}); err == nil {
		t.Fatal("PublishTelemetry() succeeded with canceled context")
	}
	if err := client.SubscribeEnvelopes(canceled, "team", workflowID+"-canceled", func([]byte) {}); err == nil {
		t.Fatal("SubscribeEnvelopes() succeeded with canceled context")
	}

	if _, err := NewClient(ctx, Config{URL: "://invalid", MaxReconnects: 0}); err == nil {
		t.Fatal("NewClient() accepted an invalid URL")
	}
	if err := client.nc.Publish(UIEventsSubject("team", workflowID), []byte("{")); err != nil {
		t.Fatalf("publish invalid UI event: %v", err)
	}
	if err := client.nc.Flush(); err != nil {
		t.Fatalf("flush invalid UI event: %v", err)
	}
	client.Close()
	client.nc.Close()
	client.Close()
	if client.IsConnected() {
		t.Fatal("closed NATS client reports connected")
	}
	if _, err := client.SubscribeUIEvents("team", workflowID, func(UIEvent) {}); err == nil {
		t.Fatal("SubscribeUIEvents() succeeded on a closed connection")
	}
	if err := client.PublishUIEvent(ctx, "team", UIEvent{SessionID: workflowID}); err == nil {
		t.Fatal("PublishUIEvent() succeeded on a closed connection")
	}
}

func TestPublishMethodsReturnMarshalErrors(t *testing.T) {
	wantErr := errors.New("encode failed")
	client := &Client{marshal: func(interface{}) ([]byte, error) { return nil, wantErr }}
	tests := []struct {
		name string
		call func() error
	}{
		{name: "envelope", call: func() error {
			return client.PublishEnvelope(context.Background(), "team", "wf", "node", map[string]string{})
		}},
		{name: "ui event", call: func() error { return client.PublishUIEvent(context.Background(), "team", UIEvent{}) }},
		{name: "status", call: func() error { return client.PublishStatus(context.Background(), StatusUpdate{}) }},
		{name: "telemetry", call: func() error { return client.PublishTelemetry(context.Background(), "team", TelemetryEvent{}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, wantErr) {
				t.Fatalf("publish error = %v, want marshal error", err)
			}
		})
	}
}

func TestNATSConnectionCallbackHelpers(t *testing.T) {
	logNATSDisconnect(nil)
	logNATSDisconnect(errors.New("connection lost"))
	logNATSReconnect("nats://localhost:4222")
	called := false
	handleUIEvent(&nats.Msg{Data: []byte("{")}, func(UIEvent) { called = true })
	if called {
		t.Fatal("malformed UI event was delivered to the handler")
	}
}
