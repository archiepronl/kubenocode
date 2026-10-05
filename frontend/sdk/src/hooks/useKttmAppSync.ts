import { useState, useCallback, useEffect, useRef } from 'react';

// Idempotent hook for syncing ReactFlow state to the KttmApp CRD YAML
export function useKttmAppSync() {
  const [nodes, setNodes] = useState([]);
  const [edges, setEdges] = useState([]);
  
  // Use a ref to prevent infinite sync loops (idempotency guard)
  const isSyncing = useRef(false);

  const updateCRD = useCallback(async () => {
    if (isSyncing.current) return;
    isSyncing.current = true;
    
    try {
      const payload = {
        nodes: nodes.map(n => ({
          id: n.id,
          type: n.data.type,
          params: n.data.params || {}
        })),
        edges: edges.map(e => ({
          from: e.source,
          to: e.target
        }))
      };
      
      // Idempotent API call to update the YAML
      await fetch('/api/bff/sync', {
        method: 'PUT', // PUT is idempotent by REST design
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
    } finally {
      isSyncing.current = false;
    }
  }, [nodes, edges]);

  // Only trigger sync if the nodes or edges actually changed length or connections
  useEffect(() => {
    updateCRD();
  }, [nodes.length, edges.length, updateCRD]);

  return { nodes, setNodes, edges, setEdges, updateCRD };
}
