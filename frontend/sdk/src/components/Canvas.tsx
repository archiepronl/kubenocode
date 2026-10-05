import React, { useCallback } from 'react';
import { ReactFlow, addEdge, applyNodeChanges, applyEdgeChanges } from '@xyflow/react';
import { useKttmAppSync } from '../hooks/useKttmAppSync';
import '@xyflow/react/dist/style.css';

// The visual DAG canvas entrypoint
export function WorkflowCanvas() {
  const { nodes, setNodes, edges, setEdges, updateCRD } = useKttmAppSync();

  // Idempotent application of state changes
  const onNodesChange = useCallback(
    (changes: any) => setNodes((nds: any) => applyNodeChanges(changes, nds)),
    [setNodes]
  );
  
  const onEdgesChange = useCallback(
    (changes: any) => setEdges((eds: any) => applyEdgeChanges(changes, eds)),
    [setEdges]
  );

  const onConnect = useCallback(
    (connection: any) => {
      setEdges((eds: any) => {
        // Prevent duplicate edges to ensure graph idempotency
        const exists = eds.find((e: any) => e.source === connection.source && e.target === connection.target);
        if (exists) return eds;
        return addEdge(connection, eds);
      });
      // The useEffect in useKttmAppSync handles the API sync to avoid race conditions
    },
    [setEdges]
  );

  return (
    <div style={{ height: '100vh', width: '100vw', background: '#0a0a0a' }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        fitView
      />
    </div>
  );
}
