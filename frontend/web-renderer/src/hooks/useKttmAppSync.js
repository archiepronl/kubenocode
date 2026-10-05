import { useState, useCallback } from 'react';

export function useKttmAppSync() {
  const [nodes, setNodes] = useState([]);
  const [edges, setEdges] = useState([]);
  const [appName, setAppName] = useState('default');
  const [loading, setLoading] = useState(false);

  const loadApp = useCallback(async (name = 'default') => {
    setLoading(true);
    try {
      const res = await fetch(`/api/apps/${name}`);
      if (res.ok) {
        const data = await res.json();
        const loadedNodes = (data.nodes || []).map(n => ({
          id: n.id,
          type: n.type || n.data?.type, // handle legacy or direct types
          position: n.position || { x: 100, y: 100 },
          data: { name: n.params?.name || n.id, type: n.type, params: n.params || {} }
        }));
        
        const loadedEdges = (data.edges || []).map(e => ({
          id: `e-${e.from}-${e.to}`,
          source: e.from,
          target: e.to
        }));

        if (loadedNodes.length > 0) {
          setNodes(loadedNodes);
          setEdges(loadedEdges);
        }
      }
    } catch (err) {
      console.error("Failed to load app:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  const saveApp = useCallback(async () => {
    try {
      const payload = {
        nodes: nodes.map(n => ({
          id: n.id,
          type: n.data.type,
          params: n.data.params || {},
          position: n.position
        })),
        edges: edges.map(e => ({
          from: e.source,
          to: e.target
        }))
      };
      
      const res = await fetch(`/api/apps/${appName}/sync`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (res.ok) {
        alert("Workflow saved successfully to " + appName + ".yaml!");
      } else {
        alert("Failed to save workflow.");
      }
    } catch (err) {
      console.error("Failed to save:", err);
      alert("Error saving workflow.");
    }
  }, [nodes, edges, appName]);

  return { nodes, setNodes, edges, setEdges, appName, setAppName, loadApp, saveApp, loading };
}
