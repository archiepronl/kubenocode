import React, { useState } from 'react';
import Form from '@rjsf/core';
import validator from '@rjsf/validator-ajv8';
import '../rjsf-dark.css';

const uiSchema = {
  contract: {
    schemaDefinition: {
      "ui:widget": "textarea",
      "ui:options": {
        rows: 5
      }
    }
  },
  server: {
    "ui:order": ["protocol", "host", "port", "exposure"]
  },
  endpoint: {
    "ui:order": ["path", "methods"]
  }
};

export function ConfigSidebar({ selectedNode, schemas, onUpdate }) {
  const [testState, setTestState] = useState(null); // null, 'pending', 'running', 'error'
  const [testUrl, setTestUrl] = useState('');

  if (!selectedNode) return <div style={{ padding: '30px', color: '#888', textAlign: 'center', marginTop: '50px' }}>Select a node on the canvas to configure it.</div>;

  const nodeSchema = schemas[selectedNode.data.type] || {};

  const handleChange = ({ formData }) => {
    onUpdate(selectedNode.id, formData);
  };

  const handleLiveTest = async () => {
    setTestState('pending');
    try {
      const res = await fetch('/api/test/webhook', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(selectedNode.data.params || {})
      });
      if (!res.ok) throw new Error("Failed to start test");
      const data = await res.json();
      pollTestStatus(data.id);
    } catch (err) {
      setTestState('error');
      console.error(err);
    }
  };

  const pollTestStatus = (id) => {
    const interval = setInterval(async () => {
      try {
        const res = await fetch(`/api/test/webhook/${id}`);
        const data = await res.json();
        if (data.phase === 'Running') {
          clearInterval(interval);
          setTestState('running');
          setTestUrl(data.url);
        }
      } catch (err) {
        clearInterval(interval);
        setTestState('error');
      }
    }, 2000);
  };

  return (
    <div className="sidebar-container" style={{ display: 'flex', flexDirection: 'column', background: '#111', color: '#fff', height: '100%', overflow: 'hidden' }}>
      <div style={{ padding: '25px 25px 15px', borderBottom: '1px solid #333' }}>
        <h3 style={{ margin: 0, color: '#fff', fontSize: '1.4rem' }}>
          Configure: <span style={{ color: '#00D1FF' }}>{selectedNode.data.name || selectedNode.id}</span>
        </h3>
      </div>
      
      <div style={{ flex: 1, overflowY: 'auto', padding: '25px' }}>
        <Form
          schema={nodeSchema}
          uiSchema={uiSchema}
          formData={selectedNode.data.params}
          validator={validator}
          onChange={handleChange}
          className="rjsf-dark"
        >
          <></> {/* Hides the default submit button */}
        </Form>
      </div>

      {selectedNode.data.type === 'trigger/webhook' && (
        <div style={{ padding: '20px 25px', borderTop: '1px solid #333', background: '#0a0a0a' }}>
          <h4 style={{ margin: '0 0 10px 0', color: '#00D1FF', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span style={{ fontSize: '1.2rem' }}>⚡</span> Live Cluster Testing
          </h4>
          <p style={{ fontSize: '12px', color: '#aaa', margin: '0 0 15px 0', lineHeight: '1.4' }}>
            Deploy a temporary listener pod to the cluster to test this webhook configuration live.
          </p>
          
          {testState === 'running' ? (
            <div style={{ background: '#111', padding: '12px', borderRadius: '6px', border: '1px solid #00D1FF' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                <p style={{ fontSize: '13px', color: '#00D1FF', margin: 0, fontWeight: 'bold' }}>✅ Pod Running</p>
                <button 
                  onClick={() => { setTestState(null); setTestUrl(''); }}
                  style={{ background: 'transparent', color: '#aaa', border: 'none', cursor: 'pointer', fontSize: '11px', textDecoration: 'underline' }}>
                  Stop Test
                </button>
              </div>
              <div style={{ background: '#000', padding: '8px', borderRadius: '4px', border: '1px solid #333' }}>
                <code style={{ color: '#a6e22e', wordBreak: 'break-all', fontSize: '11px' }}>{testUrl}</code>
              </div>
              <p style={{ fontSize: '11px', color: '#777', marginTop: '10px', marginBottom: 0 }}>Send a test payload to this URL.</p>
            </div>
          ) : (
            <button 
              onClick={handleLiveTest}
              disabled={testState === 'pending'}
              style={{ 
                width: '100%', 
                padding: '12px', 
                background: testState === 'pending' ? '#333' : 'linear-gradient(90deg, #00D1FF 0%, #0077FF 100%)', 
                color: testState === 'pending' ? '#888' : 'white', 
                border: 'none', 
                borderRadius: '6px', 
                cursor: testState === 'pending' ? 'not-allowed' : 'pointer', 
                fontWeight: 'bold',
                boxShadow: testState === 'pending' ? 'none' : '0 4px 12px rgba(0, 209, 255, 0.3)',
                transition: 'all 0.2s ease'
              }}>
              {testState === 'pending' ? 'Provisioning Pod...' : 'Test Configuration Live'}
            </button>
          )}
          {testState === 'error' && <p style={{ color: '#ff4444', fontSize: '12px', marginTop: '10px', textAlign: 'center' }}>Failed to provision test pod. Check RBAC permissions.</p>}
        </div>
      )}
    </div>
  );
}
