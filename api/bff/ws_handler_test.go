package bff

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleNodeLogs_MissingParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws/logs", nil)
	w := httptest.NewRecorder()

	// Pass nil NATS connection for test that should fail early
	HandleNodeLogs(nil, w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for missing params, got %d", w.Code)
	}
}

func TestHandleNodeLogs_WebSocketUpgrade(t *testing.T) {
	// Start NATS server mockup or skip depending on integration test level.
	// For this test, we just check if it upgrades successfully when params are present.
	// Since we need an actual NATS server for a full integration test, we will
	// just verify the HTTP upgrader behavior.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HandleNodeLogs(nil, w, r)
	}))
	defer s.Close()

	// Try to connect without WebSocket headers
	res, err := http.Get(s.URL + "?app=test&node=n1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 when not sending WS headers, got %d", res.StatusCode)
	}
}

// Ensure the Upgrader struct is idempotent and configured correctly
func TestUpgraderCheckOrigin(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if !upgrader.CheckOrigin(req) {
		t.Errorf("expected CheckOrigin to return true for all origins")
	}
}
