package bff

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var AppsDir = "/tmp/kttm-apps"

type KttmAppPayload struct {
	Nodes []map[string]interface{} `json:"nodes" yaml:"nodes"`
	Edges []map[string]interface{} `json:"edges" yaml:"edges"`
}

// HandleAppList lists all saved KttmApp workflows
func HandleAppList(w http.ResponseWriter, r *http.Request) {
	_ = os.MkdirAll(AppsDir, 0755)
	files, err := os.ReadDir(AppsDir)
	if err != nil {
		http.Error(w, "failed to read apps directory", http.StatusInternalServerError)
		return
	}

	var apps []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".yaml") {
			apps = append(apps, strings.TrimSuffix(f.Name(), ".yaml"))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

// HandleAppDelete deletes a workflow file
func HandleAppDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}

	fileLocation := filepath.Join(AppsDir, fmt.Sprintf("%s.yaml", name))
	if err := os.Remove(fileLocation); err != nil && !os.IsNotExist(err) {
		http.Error(w, "failed to delete file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"deleted"}`))
}

// HandleAppSync saves the DAG state to a YAML file, idempotent by overwriting.
func HandleAppSync(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		name = "default"
	}

	var payload KttmAppPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	_ = os.MkdirAll(AppsDir, 0755)
	fileLocation := filepath.Join(AppsDir, fmt.Sprintf("%s.yaml", name))

	yamlData, err := yaml.Marshal(payload)
	if err != nil {
		http.Error(w, "failed to encode yaml", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(fileLocation, yamlData, 0644); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"saved"}`))
}

// HandleAppGet retrieves the YAML state. If missing, returns empty DAG idempotently.
func HandleAppGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		name = "default"
	}

	fileLocation := filepath.Join(AppsDir, fmt.Sprintf("%s.yaml", name))
	data, err := os.ReadFile(fileLocation)

	var payload KttmAppPayload
	if err == nil {
		_ = yaml.Unmarshal(data, &payload)
	} else {
		payload = KttmAppPayload{Nodes: []map[string]interface{}{}, Edges: []map[string]interface{}{}}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}
