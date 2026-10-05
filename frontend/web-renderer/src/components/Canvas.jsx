import React, { useCallback, useState, useEffect, useMemo } from 'react';
import { ReactFlow, addEdge, applyNodeChanges, applyEdgeChanges, Background, Controls } from '@xyflow/react';
import { useKttmAppSync } from '../hooks/useKttmAppSync.js';
import { ConfigSidebar } from './ConfigSidebar.jsx';
import * as yaml from 'js-yaml';
import '@xyflow/react/dist/style.css';

const schemas = {
  'trigger/webhook': {
    type: 'object',
    properties: { path: { type: 'string', title: 'Webhook Path' } }
  },
  'script/python': {
    type: 'object',
    properties: { script: { type: 'string', title: 'Python Code' } }
  }
};

export function WorkflowCanvas({ appName: initialAppName, onBack }) {
  const { nodes, setNodes, edges, setEdges, appName, setAppName, loadApp, saveApp, loading } = useKttmAppSync();
  const [selectedNodeId, setSelectedNodeId] = useState(null);
  const [viewMode, setViewMode] = useState('visual'); // 'visual' | 'yaml'
  const [yamlText, setYamlText] = useState('');

  // Load the app passed from the Dashboard on mount
  useEffect(() => {
    if (initialAppName) {
      setAppName(initialAppName);
      loadApp(initialAppName);
    }
  }, [initialAppName, loadApp, setAppName]);

  // Sync state to YAML text when switching to YAML view
  useEffect(() => {
    if (viewMode === 'yaml') {
      const payload = {
        nodes: nodes.map(n => ({ id: n.id, type: n.data.type, params: n.data.params || {}, position: n.position })),
        edges: edges.map(e => ({ from: e.source, to: e.target }))
      };
      setYamlText(yaml.dump(payload));
    }
  }, [viewMode, nodes, edges]);

  const handleApplyYaml = () => {
    try {
      const parsed = yaml.load(yamlText);
      if (parsed) {
        const loadedNodes = (parsed.nodes || []).map(n => ({
          id: n.id,
          type: 'default',
          position: n.position || { x: 100, y: 100 },
          data: { name: n.params?.name || n.id, type: n.type, params: n.params || {} }
        }));
        
        const loadedEdges = (parsed.edges || []).map(e => ({
          id: `e-${e.from}-${e.to}`,
          source: e.from,
          target: e.to
        }));

        setNodes(loadedNodes);
        setEdges(loadedEdges);
        alert("YAML applied successfully!");
      }
    } catch (err) {
      alert("Invalid YAML: " + err.message);
    }
  };

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
    },
    [setEdges]
  );

  const onNodeClick = useCallback((event, node) => {
    setSelectedNodeId(node.id);
  }, []);

  const selectedNode = nodes.find(n => n.id === selectedNodeId);

  const handleUpdateNode = (id, formData) => {
    setNodes(nds => nds.map(n => {
      if (n.id === id) {
        return { ...n, data: { ...n.data, params: formData } };
      }
      return n;
    }));
  };

  const addNode = (type) => {
    const newNode = {
      id: Math.random().toString(36).substr(2, 9),
      type: 'default',
      position: { x: Math.random() * 200 + 100, y: Math.random() * 200 + 100 },
      data: { name: `New ${type.split('/')[1]}`, type, params: {} }
    };
    setNodes(nds => [...nds, newNode]);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', width: '100vw', background: '#0a0a0a', color: 'white' }}>
      
      {/* Header Toolbar */}
      <div style={{ padding: '15px', background: '#111', borderBottom: '1px solid #333', display: 'flex', gap: '15px', alignItems: 'center' }}>
        <button onClick={onBack} style={{ background: '#333', color: 'white', border: 'none', padding: '5px 15px', borderRadius: '4px', cursor: 'pointer' }}>
          &larr; Back
        </button>
        <h2 style={{ margin: 0, fontSize: '1.2rem', color: '#00D1FF' }}>Editor</h2>
        
        <div style={{ display: 'flex', background: '#222', borderRadius: '4px', overflow: 'hidden', marginLeft: '20px' }}>
          <button 
            onClick={() => setViewMode('visual')} 
            style={{ padding: '5px 15px', border: 'none', background: viewMode === 'visual' ? '#444' : 'transparent', color: 'white', cursor: 'pointer' }}>
            Visual
          </button>
          <button 
            onClick={() => setViewMode('yaml')} 
            style={{ padding: '5px 15px', border: 'none', background: viewMode === 'yaml' ? '#444' : 'transparent', color: 'white', cursor: 'pointer' }}>
            YAML
          </button>
        </div>

        <div style={{ marginLeft: 'auto', display: 'flex', gap: '10px', alignItems: 'center' }}>
          <span style={{ color: '#aaa' }}>Editing:</span>
          <strong style={{ color: 'white', marginRight: '15px' }}>{appName}.yaml</strong>
          
          <button onClick={saveApp} style={{ background: '#00D1FF', color: '#000', border: 'none', padding: '5px 15px', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>
            Save YAML
          </button>
        </div>
      </div>

      <div style={{ display: 'flex', flex: 1, position: 'relative' }}>
        
        {viewMode === 'visual' ? (
          <>
            {/* Node Catalog Sidebar */}
            <div style={{ width: '200px', borderRight: '1px solid #333', padding: '15px', background: '#151515' }}>
              <h3 style={{ marginTop: 0, fontSize: '1rem', color: '#aaa' }}>Catalog</h3>
              <button onClick={() => addNode('trigger/webhook')} style={{ width: '100%', padding: '10px', marginBottom: '10px', background: '#222', color: 'white', border: '1px solid #444', cursor: 'pointer', borderRadius: '4px' }}>+ Webhook Trigger</button>
              <button onClick={() => addNode('script/python')} style={{ width: '100%', padding: '10px', background: '#222', color: 'white', border: '1px solid #444', cursor: 'pointer', borderRadius: '4px' }}>+ Python Script</button>
            </div>

            {/* ReactFlow Canvas */}
            <div style={{ flex: 1, position: 'relative' }}>
              {loading ? (
                <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>Loading YAML...</div>
              ) : (
                <ReactFlow
                  nodes={nodes}
                  edges={edges}
                  onNodesChange={onNodesChange}
                  onEdgesChange={onEdgesChange}
                  onConnect={onConnect}
                  onNodeClick={onNodeClick}
                  fitView
                  colorMode="dark"
                >
                  <Background />
                  <Controls />
                </ReactFlow>
              )}
            </div>

            {/* Node Config Sidebar */}
            <div style={{ width: '300px', borderLeft: '1px solid #333', background: '#151515' }}>
              <ConfigSidebar selectedNode={selectedNode} schemas={schemas} onUpdate={handleUpdateNode} />
            </div>
          </>
        ) : (
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', padding: '20px', background: '#1e1e1e' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '10px' }}>
              <h3 style={{ margin: 0, color: '#aaa' }}>YAML Source Editor</h3>
              <button onClick={handleApplyYaml} style={{ background: '#333', color: 'white', border: '1px solid #555', padding: '5px 15px', borderRadius: '4px', cursor: 'pointer' }}>
                Apply Changes to Visual UI
              </button>
            </div>
            <textarea 
              value={yamlText}
              onChange={(e) => setYamlText(e.target.value)}
              style={{ flex: 1, width: '100%', background: '#0d0d0d', color: '#a6e22e', fontFamily: 'monospace', padding: '15px', border: '1px solid #333', borderRadius: '4px', fontSize: '14px', resize: 'none' }}
              spellCheck="false"
            />
          </div>
        )}
      </div>
    </div>
  );
}
