import React, { useEffect, useState } from 'react';
import { PermissionGate } from '../hooks/usePermissions.jsx';
import { AdminConsole } from './AdminConsole.jsx';
import { OrganizationSettings } from './OrganizationSettings.jsx';

// Common UI Components
const Card = ({ children, style }) => (
  <div style={{ background: '#FFFFFF', borderRadius: '8px', border: '1px solid #E2E8F0', padding: '20px', boxShadow: '0 1px 2px rgba(0,0,0,0.05)', ...style }}>
    {children}
  </div>
);

const Button = ({ children, primary, onClick, style, type="button" }) => (
  <button type={type} onClick={onClick} style={{
    background: primary ? '#4F46E5' : '#FFFFFF',
    color: primary ? '#FFFFFF' : '#334155',
    border: primary ? 'none' : '1px solid #CBD5E1',
    padding: '8px 16px',
    borderRadius: '6px',
    fontWeight: '500',
    cursor: 'pointer',
    fontSize: '0.875rem',
    transition: 'all 0.2s',
    ...style
  }}>
    {children}
  </button>
);

const Badge = ({ status }) => {
  const colors = {
    'Successful': { bg: '#D1FAE5', text: '#065F46' },
    'Running': { bg: '#DBEAFE', text: '#1E40AF' },
    'Failed': { bg: '#FEE2E2', text: '#991B1B' },
  };
  const color = colors[status] || { bg: '#F1F5F9', text: '#475569' };
  return (
    <span style={{ background: color.bg, color: color.text, padding: '2px 8px', borderRadius: '12px', fontSize: '0.75rem', fontWeight: '600' }}>
      {status}
    </span>
  );
};

export function Dashboard({ onOpenEditor, onLogout }) {
  const [apps, setApps] = useState([]);
  const [loading, setLoading] = useState(true);
  const [newAppName, setNewAppName] = useState('');
  const [activeTab, setActiveTab] = useState('Overview');
  const [activeOrg, setActiveOrg] = useState(null);

  useEffect(() => {
    const saved = localStorage.getItem('nextkube_orgs');
    if (saved) {
      const parsed = JSON.parse(saved);
      if (parsed.length > 0) setActiveOrg(parsed[0]);
    }
  }, []);

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
      setNewAppName('');
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', color: '#0F172A', overflow: 'hidden' }}>
      {/* Header */}
      <header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px', height: '60px', background: '#FFFFFF', borderBottom: '1px solid #E2E8F0', flexShrink: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '32px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', fontWeight: 'bold', fontSize: '1.25rem', color: '#1E293B' }}>
            <div style={{ width: '24px', height: '24px', background: '#4F46E5', borderRadius: '4px' }}></div>
            NextKube
          </div>
          <nav style={{ display: 'flex', gap: '24px', fontSize: '0.95rem', fontWeight: '500', color: '#64748B' }}>
            <span style={{ color: '#0F172A', cursor: 'pointer' }}>Home</span>
            <span style={{ cursor: 'pointer' }}>Projects</span>
            <span style={{ cursor: 'pointer' }}>Settings</span>
          </nav>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px', fontSize: '0.95rem', fontWeight: '500' }}>
          <span style={{ color: '#64748B', cursor: 'pointer' }}>Help</span>
          <Button primary onClick={onLogout}>Log out</Button>
        </div>
      </header>

      {/* Main Layout */}
      <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        
        {/* Left Nav */}
        <aside style={{ width: '240px', background: '#FFFFFF', borderRight: '1px solid #E2E8F0', padding: '24px 0', flexShrink: 0, overflowY: 'auto' }}>
          
          {/* Dynamic Organization Menu */}
          {activeOrg && (
            <div style={{ marginBottom: '24px' }}>
              <div style={{ padding: '0 24px', fontSize: '0.75rem', fontWeight: '700', color: '#94A3B8', textTransform: 'uppercase', marginBottom: '8px', letterSpacing: '0.05em' }}>
                {activeOrg.name}
              </div>
              <nav style={{ display: 'flex', flexDirection: 'column', gap: '4px', padding: '0 16px' }}>
                {/* Recursively render tree. For simplicity in this demo, just map root children */}
                {activeOrg.tree?.children?.map(child => (
                  <div key={child.id}>
                    <div style={{ padding: '8px 16px', fontSize: '0.875rem', fontWeight: '600', color: '#475569', display: 'flex', alignItems: 'center', gap: '8px' }}>
                      {child.type === 'structural' ? '📁' : child.type === 'leaf_workflow' ? '⚡️' : '🌐'} {child.name}
                    </div>
                    {child.children?.map(sub => (
                      <div key={sub.id} onClick={() => { setActiveTab('dynamic_'+sub.id); onOpenEditor(sub.name); }} style={{ 
                        padding: '8px 16px 8px 40px', borderRadius: '6px', cursor: 'pointer', fontSize: '0.875rem',
                        background: activeTab === 'dynamic_'+sub.id ? '#EEF2FF' : 'transparent',
                        color: activeTab === 'dynamic_'+sub.id ? '#4F46E5' : '#64748B',
                        display: 'flex', alignItems: 'center', gap: '8px'
                      }}>
                        {sub.type === 'structural' ? '📁' : sub.type === 'leaf_workflow' ? '⚡️' : '🌐'} {sub.name}
                      </div>
                    ))}
                  </div>
                ))}
              </nav>
            </div>
          )}

          <div style={{ padding: '0 24px', fontSize: '0.75rem', fontWeight: '700', color: '#94A3B8', textTransform: 'uppercase', marginBottom: '8px', letterSpacing: '0.05em' }}>
            System
          </div>
          <nav style={{ display: 'flex', flexDirection: 'column', gap: '4px', padding: '0 16px' }}>
            {['Overview', 'Security', 'Organization'].map((item) => {
              const isActive = activeTab === item;
              return (
                <div key={item} onClick={() => setActiveTab(item)} style={{ 
                  padding: '10px 16px', 
                  borderRadius: '6px', 
                  background: isActive ? '#EEF2FF' : 'transparent',
                  color: isActive ? '#4F46E5' : '#64748B',
                  fontWeight: isActive ? '600' : '500',
                  cursor: 'pointer'
                }}>
                  {item}
                </div>
              );
            })}
          </nav>
        </aside>

        {/* Center Content */}
        {activeTab === 'Organization' ? (
          <PermissionGate requiredPermission="admin" fallback={<div style={{ padding: '32px 48px' }}>Access Denied: Admins Only</div>}>
            <div style={{ flex: 1, overflow: 'hidden' }}>
              <OrganizationSettings onOrganizationChanged={setActiveOrg} />
            </div>
          </PermissionGate>
        ) : activeTab === 'Security' ? (
          <PermissionGate requiredPermission="admin" fallback={<div style={{ padding: '32px 48px' }}>Access Denied: Admins Only</div>}>
            <AdminConsole />
          </PermissionGate>
        ) : (
          <main style={{ flex: 1, padding: '32px 48px', overflowY: 'auto' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '32px' }}>
            <div>
              <h1 style={{ fontSize: '1.875rem', fontWeight: '700', margin: '0 0 8px 0' }}>Workflow Overview</h1>
              <p style={{ color: '#64748B', margin: 0 }}>Welcome to NextKube, manage your enterprise workflows.</p>
            </div>
            
            <PermissionGate requiredPermission="app:create">
              <form onSubmit={handleCreate} style={{ display: 'flex', gap: '8px' }}>
                <input 
                  type="text" 
                  placeholder="New workflow name..." 
                  value={newAppName}
                  onChange={(e) => setNewAppName(e.target.value)}
                  style={{ padding: '8px 12px', border: '1px solid #CBD5E1', borderRadius: '6px', fontSize: '0.875rem', width: '200px' }}
                  required
                />
                <Button primary type="submit">Create Workflow</Button>
              </form>
            </PermissionGate>
          </div>

          {/* Metric Cards */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '24px', marginBottom: '32px' }}>
            {[
              { label: 'Projects', val: '18', trend: '+2 this week' },
              { label: 'Workflows', val: apps.length.toString(), trend: 'Active definitions' },
              { label: 'Running', val: '2', trend: 'Live executions' },
              { label: 'Failed', val: '0', trend: 'Needs attention', alert: true }
            ].map(m => (
              <Card key={m.label} style={{ padding: '24px' }}>
                <div style={{ color: '#64748B', fontSize: '0.875rem', fontWeight: '600', marginBottom: '8px' }}>{m.label}</div>
                <div style={{ fontSize: '2.25rem', fontWeight: '700', marginBottom: '8px', color: m.alert ? '#0F172A' : '#0F172A' }}>{m.val}</div>
                <div style={{ fontSize: '0.75rem', color: m.alert ? '#10B981' : '#94A3B8', fontWeight: '500' }}>{m.trend}</div>
              </Card>
            ))}
          </div>

          {/* Saved Workflows Table */}
          <Card style={{ marginBottom: '32px', padding: 0, overflow: 'hidden' }}>
            <div style={{ padding: '20px 24px', borderBottom: '1px solid #E2E8F0', background: '#F8FAFC' }}>
              <h3 style={{ margin: 0, fontSize: '1rem', fontWeight: '600' }}>Saved Workflows</h3>
            </div>
            {loading ? (
              <div style={{ padding: '24px', color: '#64748B' }}>Loading workflows...</div>
            ) : apps.length === 0 ? (
              <div style={{ padding: '24px', color: '#64748B' }}>No workflows found. Create one above.</div>
            ) : (
              <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid #E2E8F0', color: '#64748B', fontSize: '0.75rem', textTransform: 'uppercase' }}>
                    <th style={{ padding: '16px 24px', fontWeight: '600' }}>Name</th>
                    <th style={{ padding: '16px 24px', fontWeight: '600' }}>Status</th>
                    <th style={{ padding: '16px 24px', fontWeight: '600' }}>Last Edit</th>
                    <th style={{ padding: '16px 24px', fontWeight: '600', textAlign: 'right' }}>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {apps.map(app => (
                    <tr key={app} style={{ borderBottom: '1px solid #F1F5F9' }}>
                      <td style={{ padding: '16px 24px', fontWeight: '500' }}>{app}</td>
                      <td style={{ padding: '16px 24px' }}><Badge status="Successful" /></td>
                      <td style={{ padding: '16px 24px', color: '#64748B', fontSize: '0.875rem' }}>Just now</td>
                      <td style={{ padding: '16px 24px', textAlign: 'right' }}>
                        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
                          <Button onClick={() => onOpenEditor(app)}>Edit</Button>
                          <PermissionGate requiredPermission="app:delete">
                            <Button onClick={() => handleDelete(app)} style={{ color: '#EF4444', borderColor: '#FECACA', background: '#FEF2F2' }}>Delete</Button>
                          </PermissionGate>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </Card>

          {/* Recent Executions Mock */}
          <Card style={{ padding: 0, overflow: 'hidden' }}>
            <div style={{ padding: '20px 24px', borderBottom: '1px solid #E2E8F0', background: '#F8FAFC' }}>
              <h3 style={{ margin: 0, fontSize: '1rem', fontWeight: '600' }}>Recent Executions</h3>
            </div>
            <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #E2E8F0', color: '#64748B', fontSize: '0.75rem', textTransform: 'uppercase' }}>
                  <th style={{ padding: '16px 24px', fontWeight: '600' }}>Workflow</th>
                  <th style={{ padding: '16px 24px', fontWeight: '600' }}>Project</th>
                  <th style={{ padding: '16px 24px', fontWeight: '600' }}>Start Time</th>
                  <th style={{ padding: '16px 24px', fontWeight: '600' }}>Duration</th>
                  <th style={{ padding: '16px 24px', fontWeight: '600' }}>Status</th>
                </tr>
              </thead>
              <tbody>
                <tr style={{ borderBottom: '1px solid #F1F5F9' }}>
                  <td style={{ padding: '16px 24px', fontWeight: '500' }}>Build and Test</td>
                  <td style={{ padding: '16px 24px', color: '#64748B' }}>Payments Platform</td>
                  <td style={{ padding: '16px 24px', color: '#64748B' }}>Today at 7:33 AM</td>
                  <td style={{ padding: '16px 24px', color: '#64748B' }}>22m 13s</td>
                  <td style={{ padding: '16px 24px' }}><Badge status="Successful" /></td>
                </tr>
                <tr>
                  <td style={{ padding: '16px 24px', fontWeight: '500' }}>Deploy to Staging</td>
                  <td style={{ padding: '16px 24px', color: '#64748B' }}>Customer Data</td>
                  <td style={{ padding: '16px 24px', color: '#64748B' }}>Today at 7:33 PM</td>
                  <td style={{ padding: '16px 24px', color: '#64748B' }}>20m 42s</td>
                  <td style={{ padding: '16px 24px' }}><Badge status="Running" /></td>
                </tr>
              </tbody>
            </table>
          </Card>
          </main>
        )}

        {/* Right Help Panel */}
        <aside style={{ width: '280px', background: '#FAFAF9', borderLeft: '1px solid #E2E8F0', padding: '32px 24px', flexShrink: 0 }}>
          <h3 style={{ fontSize: '1.125rem', fontWeight: '600', margin: '0 0 12px 0' }}>Help</h3>
          <p style={{ color: '#64748B', fontSize: '0.875rem', lineHeight: '1.5', marginBottom: '24px' }}>
            Need support with your workflows? Access our documentation or contact the support team.
          </p>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginBottom: '40px' }}>
            <a href="#" style={{ color: '#4F46E5', textDecoration: 'none', fontSize: '0.875rem', fontWeight: '500' }}>Documentation &rarr;</a>
            <a href="#" style={{ color: '#4F46E5', textDecoration: 'none', fontSize: '0.875rem', fontWeight: '500' }}>Workflow Guide &rarr;</a>
            <a href="#" style={{ color: '#4F46E5', textDecoration: 'none', fontSize: '0.875rem', fontWeight: '500' }}>Contact Support &rarr;</a>
          </div>

          <Card style={{ padding: '16px', background: '#FFFFFF' }}>
            <h4 style={{ margin: '0 0 12px 0', fontSize: '0.875rem', fontWeight: '600' }}>Settings Preview</h4>
            <div style={{ fontSize: '0.8125rem', color: '#64748B', display: 'flex', flexDirection: 'column', gap: '8px' }}>
              <div><strong>Engine:</strong> Argo Workflows</div>
              <div><strong>Repository:</strong> Local K3d</div>
              <div><strong>Authentication:</strong> RBAC (Pending)</div>
            </div>
          </Card>
        </aside>

      </div>

      {/* Footer */}
      <footer style={{ background: '#0F172A', color: '#94A3B8', padding: '12px 24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: '0.8125rem', flexShrink: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '24px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#10B981' }}></div>
            <span style={{ color: '#FFFFFF', fontWeight: '500' }}>Server Status: Healthy</span>
          </div>
          <span>Latest Message: All systems operational</span>
        </div>
        <div>
          &copy; {new Date().getFullYear()} NextKube. All rights reserved.
        </div>
      </footer>
    </div>
  );
}
