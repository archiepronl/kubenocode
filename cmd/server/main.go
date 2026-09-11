package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/kubeworkflow/flowengine/internal/workflow"
)

type server struct{}

func (server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "flowengine-api"})
}

func (server) schema(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"title": "Order intake",
		"description": "Submit an order to the workflow runtime.",
		"fields": []map[string]any{
			{"id": "customer", "label": "Customer", "type": "text", "required": true, "placeholder": "Acme Inc."},
			{"id": "orderId", "label": "Order ID", "type": "text", "required": true, "placeholder": "ORD-1042"},
			{"id": "priority", "label": "Priority", "type": "select", "options": []string{"standard", "expedited", "critical"}},
		},
		"submitLabel": "Start workflow",
	})
}

func (server) workflows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var definition workflow.Definition
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&definition); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workflow JSON"})
		return
	}
	if err := workflow.Validate(definition); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted", "workflow": definition.Name})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	handler := http.NewServeMux()
	api := server{}
	handler.HandleFunc("GET /healthz", api.health)
	handler.HandleFunc("GET /api/schema", api.schema)
	handler.HandleFunc("POST /api/workflows", api.workflows)
	log.Printf("flowengine API listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}