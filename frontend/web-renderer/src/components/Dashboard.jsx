import React, { useEffect, useState } from 'react';
import { PermissionGate } from '../hooks/usePermissions.jsx';
import { AdminConsole } from './AdminConsole.jsx';
import { ProjectTreeSettings } from './ProjectTreeSettings.jsx';
import { NodePropertiesModal } from './NodePropertiesModal.jsx';

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
  const [activeProjectTree, setActiveProjectTree] = useState(null);

  // Layout flexibility states for sidebars
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const saved = localStorage.getItem('nextkube_dash_sidebar_width');
    return saved ? parseInt(saved, 10) : 240;
  });
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [helpWidth, setHelpWidth] = useState(() => {
    const saved = localStorage.getItem('nextkube_dash_help_width');
    return saved ? parseInt(saved, 10) : 280;
  });
  const [helpCollapsed, setHelpCollapsed] = useState(false);

  const isDraggingSidebar = React.useRef(false);
  const isDraggingHelp = React.useRef(false);

  useEffect(() => {
    const handleMouseMove = (e) => {
      if (isDraggingSidebar.current) {
        const newW = Math.max(160, Math.min(480, e.clientX));
        setSidebarWidth(newW);
        localStorage.setItem('nextkube_dash_sidebar_width', newW);
      }
      if (isDraggingHelp.current) {
        const newW = Math.max(180, Math.min(500, window.innerWidth - e.clientX));
        setHelpWidth(newW);
        localStorage.setItem('nextkube_dash_help_width', newW);
      }
    };
    const handleMouseUp = () => {
      if (isDraggingSidebar.current || isDraggingHelp.current) {
        isDraggingSidebar.current = false;
        isDraggingHelp.current = false;
        document.body.style.cursor = 'default';
        document.body.style.userSelect = 'auto';
      }
    };
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, []);

  // Edit Node Properties from Main Menu
  const [modalNode, setModalNode] = useState(null);

  const updateNodeInTree = (node, id, updates) => {
    if (!node) return node;
    if (node.id === id) return { ...node, ...updates };
    if (node.children) {
      return { ...node, children: node.children.map(c => updateNodeInTree(c, id, updates)) };
    }
    return node;
  };

  const handleSaveMenuNodeProperties = (nodeId, updates) => {
    if (!activeProjectTree) return;
    const newTreeRoot = updateNodeInTree(activeProjectTree.tree, nodeId, updates);
    const updatedTree = { ...activeProjectTree, tree: newTreeRoot };
    setActiveProjectTree(updatedTree);
    
    // Persist in localStorage
    const saved = localStorage.getItem('nextkube_project_trees') || localStorage.getItem('nextkube_orgs');
    if (saved) {
      try {
        const trees = JSON.parse(saved);
        const updatedList = trees.map(t => t.id === activeProjectTree.id ? updatedTree : t);
        localStorage.setItem('nextkube_project_trees', JSON.stringify(updatedList));
        localStorage.setItem('nextkube_orgs', JSON.stringify(updatedList));
      } catch (e) {
        console.error('Failed to save node update', e);
      }
    }
  };

  useEffect(() => {
    const saved = localStorage.getItem('nextkube_project_trees') || localStorage.getItem('nextkube_orgs');
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        if (parsed.length > 0) setActiveProjectTree(parsed[0]);
      } catch (e) {
        console.error('Error loading project trees', e);
      }
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
      onOpenEditor({ name: newAppName.trim(), type: 'leaf_workflow' });
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
        {!sidebarCollapsed ? (
          <aside style={{ 
            width: `${sidebarWidth}px`, 
            minWidth: '160px', 
            maxWidth: '480px', 
            background: '#FFFFFF', 
            borderRight: '1px solid #E2E8F0', 
            padding: '20px 0', 
            flexShrink: 0, 
            overflowY: 'auto',
            display: 'flex',
            flexDirection: 'column',
            position: 'relative'
          }}>
            <div style={{ display: 'flex', justifyContent: 'flex-end', padding: '0 16px 8px 16px' }}>
              <button 
                onClick={() => setSidebarCollapsed(true)} 
                title="Collapse Sidebar"
                style={{ background: '#F1F5F9', border: '1px solid #CBD5E1', borderRadius: '4px', cursor: 'pointer', padding: '2px 6px', fontSize: '0.75rem', color: '#64748B' }}>
                ◀
              </button>
            </div>
            
            {/* Dynamic Project Tree Menu */}
            {activeProjectTree && (
              <div style={{ marginBottom: '24px' }}>
                <div style={{ padding: '0 20px', fontSize: '0.75rem', fontWeight: '700', color: '#94A3B8', textTransform: 'uppercase', marginBottom: '8px', letterSpacing: '0.05em', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span>{activeProjectTree.name}</span>
                  <span style={{ fontSize: '0.7rem', color: '#4F46E5', cursor: 'pointer', textTransform: 'none', fontWeight: '600' }} onClick={() => setActiveTab('Project Tree')}>
                    Tree Settings &rarr;
                  </span>
                </div>
                <nav style={{ display: 'flex', flexDirection: 'column', gap: '4px', padding: '0 12px' }}>
                  {/* Recursively render tree nodes with edit gear and K8s indicators */}
                  {activeProjectTree.tree?.children?.map(child => (
                    <div key={child.id} style={{ marginBottom: '4px' }}>
                      <div 
                        onClick={() => {
                          if (child.type !== 'structural') {
                            setActiveTab('dynamic_'+child.id);
                            onOpenEditor({ name: child.name, type: child.type, node: child });
                          }
                        }}
                        style={{ 
                          padding: '6px 10px', fontSize: '0.875rem', fontWeight: '600', color: '#334155', 
                          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                          borderRadius: '6px', background: '#F8FAFC', border: '1px solid #F1F5F9',
                          cursor: child.type !== 'structural' ? 'pointer' : 'default'
                        }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '6px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          <span>{child.type === 'structural' ? '📁' : child.type === 'leaf_workflow' ? '⚡️' : '🌐'}</span>
                          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{child.name}</span>
                          <span style={{ fontSize: '0.65rem', padding: '1px 5px', borderRadius: '3px', background: child.k8s_mapping === 'namespace' ? '#DBEAFE' : '#E2E8F0', color: child.k8s_mapping === 'namespace' ? '#1E40AF' : '#475569' }}>
                            {child.k8s_mapping === 'namespace' ? 'ns' : child.k8s_mapping}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: '4px', alignItems: 'center' }}>
                          {child.type !== 'structural' && (
                            <button
                              title={`Open Studio for ${child.name}`}
                              onClick={(e) => { e.stopPropagation(); onOpenEditor({ name: child.name, type: child.type, node: child }); }}
                              style={{
                                background: '#EEF2FF', border: '1px solid #C7D2FE', borderRadius: '4px',
                                cursor: 'pointer', fontSize: '0.72rem', padding: '2px 5px', color: '#4F46E5', fontWeight: '600'
                              }}
                            >
                              Studio
                            </button>
                          )}
                          <button
                            title={`Edit Kubernetes properties for ${child.name}`}
                            onClick={(e) => { e.stopPropagation(); setModalNode(child); }}
                            style={{
                              background: '#FFFFFF', border: '1px solid #CBD5E1', borderRadius: '4px',
                              cursor: 'pointer', fontSize: '0.75rem', padding: '2px 5px', color: '#64748B'
                            }}
                          >
                            ⚙
                          </button>
                        </div>
                      </div>

                      {child.children?.map(sub => (
                        <div 
                          key={sub.id} 
                          style={{ 
                            padding: '6px 10px 6px 28px', borderRadius: '6px', cursor: 'pointer', fontSize: '0.85rem',
                            background: activeTab === 'dynamic_'+sub.id ? '#EEF2FF' : 'transparent',
                            color: activeTab === 'dynamic_'+sub.id ? '#4F46E5' : '#475569',
                            display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: '2px'
                          }}
                          onClick={() => { 
                            setActiveTab('dynamic_'+sub.id); 
                            onOpenEditor({ name: sub.name, type: sub.type, node: sub }); 
                          }}
                        >
                          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', overflow: 'hidden' }}>
                            <span>{sub.type === 'structural' ? '📁' : sub.type === 'leaf_workflow' ? '⚡️' : '🌐'}</span>
                            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{sub.name}</span>
                            {sub.resources?.limits?.cpu && (
                              <span style={{ fontSize: '0.6rem', padding: '1px 4px', borderRadius: '3px', background: '#FEF3C7', color: '#92400E' }}>
                                {sub.resources.limits.cpu}
                              </span>
                            )}
                          </div>
                          <div style={{ display: 'flex', gap: '4px', alignItems: 'center' }}>
                            <button
                              title={`Open Studio for ${sub.name}`}
                              onClick={(e) => { e.stopPropagation(); onOpenEditor({ name: sub.name, type: sub.type, node: sub }); }}
                              style={{
                                background: '#EEF2FF', border: '1px solid #C7D2FE', borderRadius: '4px',
                                cursor: 'pointer', fontSize: '0.7rem', padding: '1px 5px', color: '#4F46E5', fontWeight: '600'
                              }}
                            >
                              Studio
                            </button>
                            <button
                              title={`Edit Kubernetes properties for ${sub.name}`}
                              onClick={(e) => { e.stopPropagation(); setModalNode(sub); }}
                              style={{
                                background: '#FFFFFF', border: '1px solid #E2E8F0', borderRadius: '4px',
                                cursor: 'pointer', fontSize: '0.7rem', padding: '1px 4px', color: '#94A3B8'
                              }}
                            >
                              ⚙
                            </button>
                          </div>
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
            <nav style={{ display: 'flex', flexDirection: 'column', gap: '4px', padding: '0 16px', flex: 1 }}>
              {['Overview', 'Security', 'Project Tree'].map((item) => {
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
        ) : (
          <div style={{ width: '36px', background: '#FFFFFF', borderRight: '1px solid #E2E8F0', display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '16px 0', flexShrink: 0 }}>
            <button 
              onClick={() => setSidebarCollapsed(false)} 
              title="Expand Sidebar"
              style={{ background: '#F1F5F9', border: '1px solid #CBD5E1', borderRadius: '4px', cursor: 'pointer', padding: '4px', fontSize: '0.8rem', color: '#64748B' }}>
              ▶
            </button>
            <div style={{ writingMode: 'vertical-rl', transform: 'rotate(180deg)', marginTop: '24px', fontSize: '0.75rem', fontWeight: '600', color: '#94A3B8', letterSpacing: '0.05em' }}>
              NAVIGATION
            </div>
          </div>
        )}

        {/* Sidebar Drag Resizer */}
        {!sidebarCollapsed && (
          <div 
            onMouseDown={(e) => {
              e.preventDefault();
              isDraggingSidebar.current = true;
              document.body.style.cursor = 'col-resize';
              document.body.style.userSelect = 'none';
            }}
            title="Drag to resize Navigation sidebar"
            style={{ width: '5px', cursor: 'col-resize', background: 'transparent', zIndex: 10, flexShrink: 0, transition: 'background 0.2s' }}
            onMouseEnter={e => e.currentTarget.style.background = '#818CF8'}
            onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
          />
        )}

        {/* Center Content */}
        {activeTab === 'Project Tree' ? (
          <PermissionGate requiredPermission="admin" fallback={<div style={{ padding: '32px 48px' }}>Access Denied: Admins Only</div>}>
            <div style={{ flex: 1, overflow: 'hidden' }}>
              <ProjectTreeSettings onProjectTreeChanged={setActiveProjectTree} onOpenStudio={onOpenEditor} />
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
                          <Button onClick={() => onOpenEditor({ name: app, type: 'leaf_workflow' })}>Open Studio</Button>
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

        {/* Help Panel Resizer */}
        {!helpCollapsed && (
          <div 
            onMouseDown={(e) => {
              e.preventDefault();
              isDraggingHelp.current = true;
              document.body.style.cursor = 'col-resize';
              document.body.style.userSelect = 'none';
            }}
            title="Drag to resize Help panel"
            style={{ width: '5px', cursor: 'col-resize', background: 'transparent', zIndex: 10, flexShrink: 0, transition: 'background 0.2s' }}
            onMouseEnter={e => e.currentTarget.style.background = '#818CF8'}
            onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
          />
        )}

        {/* Right Help Panel */}
        {!helpCollapsed ? (
          <aside style={{ 
            width: `${helpWidth}px`, 
            minWidth: '180px', 
            maxWidth: '500px', 
            background: '#FAFAF9', 
            borderLeft: '1px solid #E2E8F0', 
            padding: '24px 20px', 
            flexShrink: 0,
            overflowY: 'auto',
            display: 'flex',
            flexDirection: 'column',
            position: 'relative'
          }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <h3 style={{ fontSize: '1rem', fontWeight: '600', margin: 0, color: '#1E293B' }}>Help & Docs</h3>
              <button 
                onClick={() => setHelpCollapsed(true)} 
                title="Collapse Help Panel"
                style={{ background: '#F1F5F9', border: '1px solid #CBD5E1', borderRadius: '4px', cursor: 'pointer', padding: '2px 6px', fontSize: '0.75rem', color: '#64748B' }}>
                ▶
              </button>
            </div>
            
            <p style={{ color: '#64748B', fontSize: '0.85rem', lineHeight: '1.5', marginBottom: '20px' }}>
              Need support with your workflows? Access our documentation or contact the support team.
            </p>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', marginBottom: '28px' }}>
              <a href="#" style={{ color: '#4F46E5', textDecoration: 'none', fontSize: '0.85rem', fontWeight: '500' }}>Documentation &rarr;</a>
              <a href="#" style={{ color: '#4F46E5', textDecoration: 'none', fontSize: '0.85rem', fontWeight: '500' }}>Workflow Guide &rarr;</a>
              <a href="#" style={{ color: '#4F46E5', textDecoration: 'none', fontSize: '0.85rem', fontWeight: '500' }}>Contact Support &rarr;</a>
            </div>

            <Card style={{ padding: '16px', background: '#FFFFFF' }}>
              <h4 style={{ margin: '0 0 10px 0', fontSize: '0.85rem', fontWeight: '600' }}>Settings Preview</h4>
              <div style={{ fontSize: '0.8rem', color: '#64748B', display: 'flex', flexDirection: 'column', gap: '6px' }}>
                <div><strong>Engine:</strong> Argo Workflows</div>
                <div><strong>Repository:</strong> Local K3d</div>
                <div><strong>Authentication:</strong> RBAC (Pending)</div>
              </div>
            </Card>
          </aside>
        ) : (
          <div style={{ width: '36px', background: '#FAFAF9', borderLeft: '1px solid #E2E8F0', display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '16px 0', flexShrink: 0 }}>
            <button 
              onClick={() => setHelpCollapsed(false)} 
              title="Expand Help Panel"
              style={{ background: '#F1F5F9', border: '1px solid #CBD5E1', borderRadius: '4px', cursor: 'pointer', padding: '4px', fontSize: '0.8rem', color: '#64748B' }}>
              ◀
            </button>
            <div style={{ writingMode: 'vertical-rl', marginTop: '24px', fontSize: '0.75rem', fontWeight: '600', color: '#94A3B8', letterSpacing: '0.05em' }}>
              HELP & DOCS
            </div>
          </div>
        )}

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
      {/* Node Properties Modal for Main Menu edits */}
      {modalNode && (
        <NodePropertiesModal 
          node={modalNode}
          isOpen={!!modalNode}
          onClose={() => setModalNode(null)}
          onSave={handleSaveMenuNodeProperties}
          hasAncestorNamespace={modalNode.k8s_mapping !== 'namespace' && activeProjectTree?.tree?.k8s_mapping === 'namespace'}
          ancestorName={activeProjectTree?.tree?.name}
        />
      )}
    </div>
  );
}
