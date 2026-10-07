package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kubeworkflow/flowengine/core/connectors"
)

func TestWebhookReadPreservesBodyAndGuardsMethod(t *testing.T) {
	port := unusedPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resultCh := startRead(t, ctx, connectors.ConnectorConfig{Params: map[string]string{
		paramPath: "/events", paramPort: port, paramMethod: http.MethodPost,
	}})

	wrongMethod := requestUntilResponse(t, http.MethodGet, port, "/events", nil, nil)
	if wrongMethod.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", wrongMethod.StatusCode)
	}
	_ = wrongMethod.Body.Close()

	body := []byte(`{"event":"created"}`)
	response := requestUntilResponse(t, http.MethodPost, port, "/events", body, nil)
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("POST status = %d, want 202", response.StatusCode)
	}
	_ = response.Body.Close()
	assertWebhookResult(t, resultCh, body, "application/octet-stream")
}

func TestWebhookReadVerifiesHMACAndPreservesContentType(t *testing.T) {
	port := unusedPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resultCh := startRead(t, ctx, connectors.ConnectorConfig{
		Params: map[string]string{paramPath: "/signed", paramPort: port},
		Env:    map[string]string{envHMACSecret: "test-secret"},
	})
	body := []byte("signed payload")
	bad := requestUntilResponse(t, http.MethodPost, port, "/signed", body, map[string]string{
		"Content-Type": "application/custom", "X-FlowEngine-Signature": "sha256=invalid",
	})
	if bad.StatusCode != http.StatusForbidden {
		t.Fatalf("invalid signature status = %d, want 403", bad.StatusCode)
	}
	_ = bad.Body.Close()

	signature := hmacSignature(body, "test-secret")
	good := requestUntilResponse(t, http.MethodPost, port, "/signed", body, map[string]string{
		"Content-Type": "application/custom", "X-Hub-Signature-256": "sha256=" + signature,
	})
	if good.StatusCode != http.StatusAccepted {
		t.Fatalf("valid signature status = %d, want 202", good.StatusCode)
	}
	_ = good.Body.Close()
	assertWebhookResult(t, resultCh, body, "application/custom")
}

func TestWebhookReadReturnsOnContextCancellation(t *testing.T) {
	port := unusedPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := startRead(t, ctx, connectors.ConnectorConfig{Params: map[string]string{paramPort: port}})
	waitForListener(t, port)
	cancel()
	select {
	case result := <-resultCh:
		if result.err != context.Canceled {
			t.Fatalf("Read() error = %v, want context.Canceled", result.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read() did not return after context cancellation")
	}
}

func TestWebhookReadReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("reserve listener: %v", err)
	}
	port := fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)
	defer listener.Close()
	resultCh := startRead(t, context.Background(), connectors.ConnectorConfig{Params: map[string]string{paramPort: port}})
	select {
	case result := <-resultCh:
		if result.err == nil || !strings.Contains(result.err.Error(), "webhook server error") {
			t.Fatalf("Read() error = %v, want listen error", result.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read() did not return after listen failure")
	}
}

func TestWebhookHelpersAndWrite(t *testing.T) {
	connector := New()
	if connector.Type() != connectorType || !json.Valid(connector.Schema()) {
		t.Fatal("webhook connector type or schema is invalid")
	}
	if err := connector.Write(context.Background(), connectors.EnvelopeRef{}, strings.NewReader("data"), connectors.ConnectorConfig{}); err == nil {
		t.Fatal("Write() succeeded for a source-only connector")
	}
	if !verifyHMAC([]byte("body"), hmacSignature([]byte("body"), "secret"), "secret") {
		t.Fatal("verifyHMAC rejected an unprefixed valid signature")
	}
	if verifyHMAC([]byte("body"), "sha256=invalid", "secret") {
		t.Fatal("verifyHMAC accepted an invalid signature")
	}
	if got := coalesce("", "first", "second"); got != "first" {
		t.Fatalf("coalesce() = %q", got)
	}
	if got := coalesce("", ""); got != "" {
		t.Fatalf("coalesce(empty) = %q", got)
	}
	reader := newBytesReader([]byte("abc"))
	buffer := make([]byte, 2)
	if n, err := reader.Read(buffer); err != nil || n != 2 || string(buffer) != "ab" {
		t.Fatalf("first bytesReader.Read() = (%d, %v, %q)", n, err, buffer)
	}
	if n, err := reader.Read(buffer); err != nil || n != 1 || string(buffer[:n]) != "c" {
		t.Fatalf("second bytesReader.Read() = (%d, %v, %q)", n, err, buffer[:n])
	}
	if n, err := reader.Read(buffer); n != 0 || err != io.EOF {
		t.Fatalf("EOF bytesReader.Read() = (%d, %v), want (0, EOF)", n, err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("bytesReader.Close() error = %v", err)
	}
}

func TestWebhookBodyReadFailureReturnsBadRequest(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/signed", nil)
	request.Body = io.NopCloser(failingBody{})
	results := make(chan *connectors.ReadResult, 1)
	handleWebhookRequest(response, request, results, "/signed", http.MethodPost, "secret")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("handler status = %d, want 400", response.Code)
	}
	select {
	case result := <-results:
		t.Fatalf("handler emitted a result after body read failure: %+v", result)
	default:
	}
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, fmt.Errorf("body read failed") }

type readResult struct {
	result *connectors.ReadResult
	err    error
}

func startRead(t *testing.T, ctx context.Context, cfg connectors.ConnectorConfig) <-chan readResult {
	t.Helper()
	resultCh := make(chan readResult, 1)
	go func() {
		result, err := New().Read(ctx, cfg)
		resultCh <- readResult{result: result, err: err}
	}()
	return resultCh
}

func requestUntilResponse(t *testing.T, method, port, path string, body []byte, headers map[string]string) *http.Response {
	t.Helper()
	client := &http.Client{
		Timeout:   250 * time.Millisecond,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	deadline := time.Now().Add(3 * time.Second)
	url := "http://127.0.0.1:" + port + path
	for time.Now().Before(deadline) {
		request, err := http.NewRequest(method, url, strings.NewReader(string(body)))
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		for key, value := range headers {
			request.Header.Set(key, value)
		}
		response, err := client.Do(request)
		if err == nil {
			return response
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server at %s did not respond", url)
	return nil
}

func assertWebhookResult(t *testing.T, resultCh <-chan readResult, wantBody []byte, wantMIME string) {
	t.Helper()
	select {
	case outcome := <-resultCh:
		if outcome.err != nil {
			t.Fatalf("Read() error = %v", outcome.err)
		}
		if outcome.result.Envelope.MimeType != wantMIME || outcome.result.Envelope.PayloadSize != int64(len(wantBody)) {
			t.Fatalf("unexpected envelope: %+v", outcome.result.Envelope)
		}
		got, err := io.ReadAll(outcome.result.Stream)
		if err != nil {
			t.Fatalf("reading webhook stream: %v", err)
		}
		_ = outcome.result.Stream.Close()
		if string(got) != string(wantBody) {
			t.Fatalf("stream = %q, want %q", got, wantBody)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Read() did not return after webhook request")
	}
}

func unusedPort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	port := fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)
	if err := listener.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}
	return port
}

func waitForListener(t *testing.T, port string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 50*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("listener did not open port %s", port)
}

func hmacSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
