package bff

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HandleNodeLogs upgrades the HTTP connection and streams NATS logs to the browser.
// This function is idempotent: connecting multiple times safely multiplexes the NATS subject.
func HandleNodeLogs(nc *nats.Conn, w http.ResponseWriter, r *http.Request) {
	appName := r.URL.Query().Get("app")
	nodeID := r.URL.Query().Get("node")

	if appName == "" || nodeID == "" {
		http.Error(w, "missing 'app' or 'node' query parameters", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// If NATS connection is nil (e.g. during certain tests), return gracefully
	if nc == nil {
		return
	}

	// Subscribe to the NATS subject where the operator streams Argo pod logs
	subject := fmt.Sprintf("kttm.apps.%s.nodes.%s.log", appName, nodeID)
	sub, err := nc.Subscribe(subject, func(m *nats.Msg) {
		// Idempotent stream push
		_ = conn.WriteMessage(websocket.TextMessage, m.Data)
	})
	if err != nil {
		return
	}
	defer sub.Unsubscribe()

	// Keep connection alive until client disconnects (read pump)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
