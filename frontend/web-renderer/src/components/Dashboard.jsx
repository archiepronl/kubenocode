import React, { useEffect, useState } from 'react';

export function Dashboard({ onOpenEditor }) {
  const [apps, setApps] = useState([]);
  const [loading, setLoading] = useState(true);
  const [newAppName, setNewAppName] = useState('');

  const fetchApps = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/apps');
      if (res.ok) {
        setApps(await res.json() || []);
      }
    } catch (err) {
      console.error("Failed to fetch apps", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchApps();
  }, []);

  const handleDelete = async (name) => {
    if (!confirm(`Are you sure you want to delete ${name}?`)) return;
    try {
      await fetch(`/api/apps/${name}`, { method: 'DELETE' });
      fetchApps();
    } catch (err) {
      console.error("Failed to delete", err);
    }
  };

  const handleCreate = (e) => {
    e.preventDefault();
    if (newAppName.trim()) {
      onOpenEditor(newAppName.trim());
    }
  };

  return (
    <div style={{ padding: '40px', maxWidth: '1000px', margin: '0 auto', color: 'white' }}>
      <h1 style={{ color: '#00D1FF', marginBottom: '40px' }}>FlowEngine Workflows</h1>
      
      <div style={{ background: '#151515', padding: '20px', borderRadius: '8px', marginBottom: '40px', border: '1px solid #333' }}>
        <h3 style={{ marginTop: 0 }}>Create New Workflow</h3>
        <form onSubmit={handleCreate} style={{ display: 'flex', gap: '10px' }}>
          <input 
            type="text" 
            placeholder="Workflow Name (e.g., etl-pipeline)" 
            value={newAppName}
            onChange={(e) => setNewAppName(e.target.value)}
            style={{ flex: 1, padding: '10px', background: '#222', border: '1px solid #444', color: 'white', borderRadius: '4px' }}
            required
          />
          <button type="submit" style={{ padding: '10px 20px', background: '#00D1FF', color: 'black', border: 'none', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>
            Create
          </button>
        </form>
      </div>

      <h3 style={{ borderBottom: '1px solid #333', paddingBottom: '10px' }}>Saved Workflows</h3>
      
      {loading ? (
        <p>Loading workflows...</p>
      ) : apps.length === 0 ? (
        <p style={{ color: '#aaa' }}>No workflows found. Create one above.</p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
          {apps.map(app => (
            <div key={app} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: '#1e1e1e', padding: '15px 20px', borderRadius: '6px', border: '1px solid #333' }}>
              <div style={{ fontSize: '1.1rem', fontWeight: '500' }}>{app}</div>
              <div style={{ display: 'flex', gap: '10px' }}>
                <button onClick={() => onOpenEditor(app)} style={{ padding: '8px 15px', background: '#333', color: 'white', border: '1px solid #555', borderRadius: '4px', cursor: 'pointer' }}>
                  Edit
                </button>
                <button onClick={() => handleDelete(app)} style={{ padding: '8px 15px', background: '#4a1111', color: '#ff6b6b', border: '1px solid #7a1f1f', borderRadius: '4px', cursor: 'pointer' }}>
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
