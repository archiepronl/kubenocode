package bff

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandleSync_ValidJSON(t *testing.T) {
	// Setup test directory
	_ = os.MkdirAll("/tmp/kttm-apps", 0755)
	defer os.RemoveAll("/tmp/kttm-apps")

	payload := map[string]interface{}{
		"nodes": []interface{}{
			map[string]interface{}{"id": "1", "type": "trigger", "params": map[string]interface{}{}},
		},
		"edges": []interface{}{},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/apps/default/sync", bytes.NewReader(body))
	// Mock chi/mux path param
	req.SetPathValue("name", "default")
	w := httptest.NewRecorder()

	HandleAppSync(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	// Verify file was written
	if _, err := os.Stat("/tmp/kttm-apps/default.yaml"); os.IsNotExist(err) {
		t.Errorf("expected file /tmp/kttm-apps/default.yaml to be created")
	}
}

func TestHandleAppGet_Found(t *testing.T) {
	_ = os.MkdirAll("/tmp/kttm-apps", 0755)
	defer os.RemoveAll("/tmp/kttm-apps")
	_ = os.WriteFile("/tmp/kttm-apps/default.yaml", []byte(`nodes: []`), 0644)

	req := httptest.NewRequest(http.MethodGet, "/api/apps/default", nil)
	req.SetPathValue("name", "default")
	w := httptest.NewRecorder()

	HandleAppGet(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}
}

func TestHandleAppGet_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/apps/missing", nil)
	req.SetPathValue("name", "missing")
	w := httptest.NewRecorder()

	HandleAppGet(w, req)

	if w.Code != http.StatusOK { // Returns empty DAG on missing
		t.Errorf("expected 200 OK for missing file (creates empty DAG), got %d", w.Code)
	}
}
