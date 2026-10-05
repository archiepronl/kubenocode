# Phase 3 & 4 — Visual Benchmark (Developer SDK & AI Advisor)
# KubeNoCode Platform · Implementation Guide

To evolve KTTM from a backend execution engine into a true low-code developer experience, we must build the **Visual DAG Canvas** and the **AI Advisor**. This phase bridges the gap between the `KttmApp` CRD and the developer by providing a drag-and-drop workspace that natively synchronizes with the underlying Kubernetes YAML, augmented by real-time observability and rule-based diagnostics.

We will accomplish this by building three primary architecture pillars:

1. **The XYFlow Visual Editor (Frontend)**: A React-based interactive canvas that natively parses and writes the `KttmApp` schema.
2. **The JSON Schema Engine**: A dynamic configuration sidebar that renders input fields automatically based on connector definitions.
3. **The Telemetry & Advisor Loop (Backend)**: A goroutine that polls node metrics, streams logs via NATS WebSockets, and flags performance issues (e.g., CPU throttling, OOM risk).

---

## Step 1: The Visual DAG Canvas (ReactFlow)

The frontend developer SDK is built on React and `@xyflow/react`. It maintains bidirectional sync with the Kubernetes `KttmApp` spec.

### File 1: Workflow Canvas Component (`frontend/sdk/src/components/Canvas.tsx`)
This component renders the interactive workflow nodes and connects them.

```tsx
// frontend/sdk/src/components/Canvas.tsx
import { useCallback } from 'react';
import { ReactFlow, addEdge, applyNodeChanges, applyEdgeChanges } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { CustomNode } from './CustomNode';
import { useKttmAppSync } from '../hooks/useKttmAppSync';

const nodeTypes = { kttmNode: CustomNode };

export function WorkflowCanvas() {
  const { nodes, setNodes, edges, setEdges, updateCRD } = useKttmAppSync();

  const onNodesChange = useCallback(
    (changes) => setNodes((nds) => applyNodeChanges(changes, nds)),
    [setNodes]
  );
  
  const onEdgesChange = useCallback(
    (changes) => setEdges((eds) => applyEdgeChanges(changes, eds)),
    [setEdges]
  );

  const onConnect = useCallback(
    (connection) => {
      setEdges((eds) => addEdge(connection, eds));
      updateCRD(); // Syncs back to the KttmApp YAML automatically
    },
    [setEdges, updateCRD]
  );

  return (
    <div style={{ height: '100vh', width: '100vw' }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        nodeTypes={nodeTypes}
        fitView
      />
    </div>
  );
}
```

---

## Step 2: Schema-Driven Configuration Sidebar

When a user clicks on a node in the canvas, they see a form dynamically generated from the connector's JSON schema (e.g., S3 Bucket Name vs. Postgres Table).

### File 2: Config Sidebar (`frontend/sdk/src/components/ConfigSidebar.tsx`)

```tsx
// frontend/sdk/src/components/ConfigSidebar.tsx
import Form from '@rjsf/core';
import validator from '@rjsf/validator-ajv8';

export function ConfigSidebar({ selectedNode, schemas, onUpdate }) {
  if (!selectedNode) return <div className="sidebar-empty">Select a node</div>;

  // Retrieve the JSON schema for the selected node's connector type (e.g. "sink/postgres")
  const nodeSchema = schemas[selectedNode.data.type];

  const handleSubmit = ({ formData }) => {
    onUpdate(selectedNode.id, formData);
  };

  return (
    <div className="sidebar-container">
      <h3>Configure: {selectedNode.data.name}</h3>
      <Form
        schema={nodeSchema}
        formData={selectedNode.data.params}
        validator={validator}
        onSubmit={handleSubmit}
      />
    </div>
  );
}
```

---

## Step 3: Real-Time Telemetry & Log Streaming (BFF)

The Backend-For-Frontend (BFF) proxies NATS messages to the browser via WebSockets, allowing the UI to show live node status colors (Idle → Running → Succeeded) and stream raw logs without refreshing.

### File 3: WebSocket Streaming Handler (`api/bff/ws_handler.go`)

```go
// api/bff/ws_handler.go
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

// HandleNodeLogs upgrades the HTTP connection and streams NATS logs to the browser
func HandleNodeLogs(nc *nats.Conn, w http.ResponseWriter, r *http.Request) {
	appName := r.URL.Query().Get("app")
	nodeID := r.URL.Query().Get("node")
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Subscribe to the NATS subject where the operator streams Argo pod logs
	subject := fmt.Sprintf("kttm.apps.%s.nodes.%s.log", appName, nodeID)
	sub, err := nc.Subscribe(subject, func(m *nats.Msg) {
		_ = conn.WriteMessage(websocket.TextMessage, m.Data)
	})
	if err != nil {
		return
	}
	defer sub.Unsubscribe()

	// Keep connection alive until client disconnects
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
```

---

## Step 4: The Rule-Based AI Advisor Engine

Before calling expensive LLM APIs, the system runs a deterministic rule engine that scans Prometheus metrics (or the GraphLinter) to flag performance issues.

### File 4: Rule Advisor (`core/advisor/advisor.go`)

```go
// core/advisor/advisor.go
package advisor

import (
	"fmt"
)

type NodeMetrics struct {
	NodeID     string
	MemPercent float64
	CPUPercent float64
	ErrorRate  float64
}

type Advice struct {
	Severity string `json:"severity"` // "critical", "warning", "info"
	Category string `json:"category"`
	Message  string `json:"message"`
	Action   string `json:"action"`
}

// Analyze returns actionable advice for a workflow node based on live metrics
func Analyze(m NodeMetrics) []Advice {
	var results []Advice

	// Memory Pressure Rule
	if m.MemPercent >= 90.0 {
		results = append(results, Advice{
			Severity: "critical",
			Category: "memory",
			Message:  fmt.Sprintf("Node '%s' is critically close to OOMKill (%.1f%% memory used).", m.NodeID, m.MemPercent),
			Action:   "Increase resources.limits.memory by at least 25% in the node configuration.",
		})
	}

	// High Error Rate Rule
	if m.ErrorRate >= 20.0 {
		results = append(results, Advice{
			Severity: "critical",
			Category: "reliability",
			Message:  fmt.Sprintf("Node '%s' has an error rate of %.1f%%.", m.NodeID, m.ErrorRate),
			Action:   "Check the node logs. Consider adding a retryPolicy with exponential backoff.",
		})
	}

	return results
}
```

---

## Validation & QA Checklist (MS2)

To satisfy the QA Architect and pass Phase 3 & 4:
1. **Frontend Tests**: Ensure `useKttmAppSync` has unit tests verifying that removing a node from the canvas properly deletes it from the raw YAML representation.
2. **Advisor Unit Tests**: Validate `Analyze()` returns `critical` memory warnings when metrics exceed 90%, and `warning` when CPU exceeds 80%.
3. **Integration**: Run the pipeline from the UI. Watch the node colors flip from grey → blue → green via WebSocket sync, and verify the NATS log streaming drawer successfully renders real Argo stdout.
