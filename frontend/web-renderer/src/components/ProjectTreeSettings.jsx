import React, { useState, useEffect, useRef } from 'react';
import { NodePropertiesModal } from './NodePropertiesModal.jsx';

const Button = ({ children, primary, secondary, danger, onClick, style, disabled, title }) => {
  let bg = '#FFFFFF', color = '#334155', border = '1px solid #CBD5E1';
  if (primary) { bg = disabled ? '#94A3B8' : '#4F46E5'; color = '#FFFFFF'; border = 'none'; }
  else if (secondary) { bg = '#F1F5F9'; border = 'none'; }
  else if (danger) { bg = '#FEF2F2'; color = '#EF4444'; border = '1px solid #FECACA'; }
  
  return (
    <button onClick={onClick} disabled={disabled} title={title} style={{
      background: bg, color, border, padding: '8px 14px', borderRadius: '6px',
      fontWeight: '500', cursor: disabled ? 'not-allowed' : 'pointer',
      fontSize: '0.875rem', transition: 'all 0.15s ease', display: 'inline-flex',
      alignItems: 'center', justifyContent: 'center', gap: '6px', ...style
    }}>
      {children}
    </button>
  );
};

const IconButton = ({ children, onClick, title, active, style }) => (
  <button 
    onClick={onClick} 
    title={title}
    style={{
      background: active ? '#EEF2FF' : '#F8FAFC',
      border: `1px solid ${active ? '#C7D2FE' : '#E2E8F0'}`,
      color: active ? '#4F46E5' : '#64748B',
      borderRadius: '6px',
      cursor: 'pointer',
      padding: '4px 8px',
      fontSize: '0.8rem',
      fontWeight: '600',
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      transition: 'all 0.15s ease',
      ...style
    }}
  >
    {children}
  </button>
);

const Input = ({ label, value, onChange, placeholder }) => (
  <div style={{ marginBottom: '16px' }}>
    {label && <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: '500', marginBottom: '6px', color: '#1E293B' }}>{label}</label>}
    <input type="text" value={value} onChange={e => onChange(e.target.value)} placeholder={placeholder} style={{
      width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box'
    }} />
  </div>
);

const Select = ({ label, value, onChange, options }) => (
  <div style={{ marginBottom: '16px' }}>
    {label && <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: '500', marginBottom: '6px', color: '#1E293B' }}>{label}</label>}
    <select value={value} onChange={e => onChange(e.target.value)} style={{
      width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box', background: '#FFF'
    }}>
      {options.map(opt => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
    </select>
  </div>
);

// Helpers to trace path & hierarchy
const findPathToNode = (current, targetId, currentPath = []) => {
  if (!current) return null;
  const path = [...currentPath, current];
  if (current.id === targetId) return path;
  if (current.children) {
    for (const child of current.children) {
      const result = findPathToNode(child, targetId, path);
      if (result) return result;
    }
  }
  return null;
};

const hasDescendantWithNamespace = (node) => {
  if (!node || !node.children) return false;
  return node.children.some(child => 
    child.k8s_mapping === 'namespace' || hasDescendantWithNamespace(child)
  );
};

export function ProjectTreeSettings({ onProjectTreeChanged, onOpenStudio }) {
  const [projectTrees, setProjectTrees] = useState([]);
  const [activeTreeId, setActiveTreeId] = useState(null);
  const [selectedNodeId, setSelectedNodeId] = useState(null);
  
  // Resizable & Collapsible Panel States with defaults
  const [leftWidth, setLeftWidth] = useState(() => {
    const saved = localStorage.getItem('nextkube_panel_left_width');
    return saved ? parseInt(saved, 10) : 260;
  });
  const [rightWidth, setRightWidth] = useState(() => {
    const saved = localStorage.getItem('nextkube_panel_right_width');
    return saved ? parseInt(saved, 10) : 340;
  });

  const [leftCollapsed, setLeftCollapsed] = useState(false);
  const [rightCollapsed, setRightCollapsed] = useState(false);
  const [middleMaximized, setMiddleMaximized] = useState(false);

  // Modal State for Advanced Kubernetes Properties
  const [modalNode, setModalNode] = useState(null);

  // Dragging state for splitters
  const isDraggingLeft = useRef(false);
  const isDraggingRight = useRef(false);

  useEffect(() => {
    const handleMouseMove = (e) => {
      if (isDraggingLeft.current) {
        const newW = Math.max(160, Math.min(500, e.clientX - 240));
        setLeftWidth(newW);
        localStorage.setItem('nextkube_panel_left_width', newW);
      }
      if (isDraggingRight.current) {
        const containerRight = window.innerWidth - 280;
        const newW = Math.max(220, Math.min(600, containerRight - e.clientX));
        setRightWidth(newW);
        localStorage.setItem('nextkube_panel_right_width', newW);
      }
    };

    const handleMouseUp = () => {
      if (isDraggingLeft.current || isDraggingRight.current) {
        isDraggingLeft.current = false;
        isDraggingRight.current = false;
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

  const startResizeLeft = (e) => {
    e.preventDefault();
    isDraggingLeft.current = true;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  };

  const startResizeRight = (e) => {
    e.preventDefault();
    isDraggingRight.current = true;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  };

  // Load from local storage
  useEffect(() => {
    const saved = localStorage.getItem('nextkube_project_trees') || localStorage.getItem('nextkube_orgs');
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        if (Array.isArray(parsed) && parsed.length > 0) {
          const cleaned = parsed.map((t, idx) => ({
            ...t,
            name: (!t.name || t.name === 'New Organization') ? `Project Tree ${idx + 1}` : t.name
          }));
          setProjectTrees(cleaned);
          setActiveTreeId(cleaned[0].id);
          setSelectedNodeId('root');
          return;
        }
      } catch (e) {
        console.error('Failed to parse saved project trees', e);
      }
    }
    
    // Default initial project tree if empty
    const defaultTree = {
      id: 'tree_' + Date.now(),
      name: 'Default Project Tree',
      tree: {
        id: 'root',
        name: 'Root',
        type: 'structural',
        k8s_mapping: 'namespace',
        k8s_namespace: 'default',
        resources: { requests: { cpu: '100m', memory: '128Mi' }, limits: { cpu: '500m', memory: '512Mi' } },
        children: [
          {
            id: 'node_billing',
            name: 'Billing Platform',
            type: 'structural',
            k8s_mapping: 'tags',
            k8s_labels: 'app=billing, tier=api',
            resources: { requests: { cpu: '200m', memory: '256Mi' }, limits: { cpu: '1', memory: '1Gi' } },
            children: [
              { id: 'node_invoice_wf', name: 'Invoice Generator', type: 'leaf_workflow', k8s_mapping: 'pod', resources: { requests: { cpu: '250m', memory: '512Mi' }, limits: { cpu: '1', memory: '2Gi' } } },
              { id: 'node_portal_web', name: 'Customer Portal', type: 'leaf_webapp', k8s_mapping: 'pod', resources: { requests: { cpu: '100m', memory: '128Mi' }, limits: { cpu: '500m', memory: '512Mi' } } }
            ]
          }
        ]
      }
    };
    setProjectTrees([defaultTree]);
    setActiveTreeId(defaultTree.id);
    setSelectedNodeId('root');
    saveToStorage([defaultTree], defaultTree.id);
  }, []);

  const saveToStorage = (updatedTrees, targetId = activeTreeId) => {
    setProjectTrees(updatedTrees);
    localStorage.setItem('nextkube_project_trees', JSON.stringify(updatedTrees));
    localStorage.setItem('nextkube_orgs', JSON.stringify(updatedTrees));
    if (onProjectTreeChanged) {
      const active = updatedTrees.find(t => t.id === targetId) || updatedTrees[0] || null;
      onProjectTreeChanged(active);
    }
  };

  const activeTree = projectTrees.find(t => t.id === activeTreeId);

  const handleCreateTree = () => {
    const newTree = {
      id: 'tree_' + Date.now(),
      name: `Project Tree ${projectTrees.length + 1}`,
      tree: { 
        id: 'root', 
        name: 'Root', 
        type: 'structural', 
        k8s_mapping: 'namespace',
        k8s_namespace: 'default',
        resources: { requests: { cpu: '100m', memory: '128Mi' }, limits: { cpu: '500m', memory: '512Mi' } },
        children: [] 
      }
    };
    const updated = [...projectTrees, newTree];
    setActiveTreeId(newTree.id);
    setSelectedNodeId('root');
    saveToStorage(updated, newTree.id);
  };

  const handleDeleteTree = (treeId) => {
    if (projectTrees.length <= 1) {
      alert("At least one Project Tree must be maintained.");
      return;
    }
    if (!confirm("Are you sure you want to delete this Project Tree?")) return;
    const updated = projectTrees.filter(t => t.id !== treeId);
    const nextActive = updated[0]?.id || null;
    setActiveTreeId(nextActive);
    setSelectedNodeId('root');
    saveToStorage(updated, nextActive);
  };

  const handleUpdateTreeName = (newName) => {
    if (!activeTree) return;
    const updated = projectTrees.map(t => t.id === activeTreeId ? { ...t, name: newName } : t);
    saveToStorage(updated);
  };

  // Tree manipulation helpers
  const findNode = (node, id) => {
    if (!node) return null;
    if (node.id === id) return node;
    if (node.children) {
      for (let child of node.children) {
        const found = findNode(child, id);
        if (found) return found;
      }
    }
    return null;
  };

  const updateNode = (node, id, updates) => {
    if (!node) return node;
    if (node.id === id) return { ...node, ...updates };
    if (node.children) {
      return { ...node, children: node.children.map(c => updateNode(c, id, updates)) };
    }
    return node;
  };

  const removeNode = (node, id) => {
    if (!node || !node.children) return node;
    return {
      ...node,
      children: node.children.filter(c => c.id !== id).map(c => removeNode(c, id))
    };
  };

  const handleUpdateNode = (updates) => {
    if (!activeTree || !selectedNodeId) return;
    const newTreeRoot = updateNode(activeTree.tree, selectedNodeId, updates);
    const updated = projectTrees.map(t => t.id === activeTreeId ? { ...t, tree: newTreeRoot } : t);
    saveToStorage(updated);
  };

  const handleSaveModalProperties = (nodeId, updates) => {
    if (!activeTree) return;
    const newTreeRoot = updateNode(activeTree.tree, nodeId, updates);
    const updated = projectTrees.map(t => t.id === activeTreeId ? { ...t, tree: newTreeRoot } : t);
    saveToStorage(updated);
  };

  const handleAddChild = (parentId) => {
    if (!activeTree) return;
    const parent = findNode(activeTree.tree, parentId);
    if (!parent || parent.type !== 'structural') return;

    const lineage = findPathToNode(activeTree.tree, parentId) || [];
    const parentOrAncestorHasNamespace = lineage.some(n => n.k8s_mapping === 'namespace');
    const defaultMapping = parentOrAncestorHasNamespace ? 'tags' : 'namespace';

    const newChild = { 
      id: 'node_' + Date.now(), 
      name: 'New Node', 
      type: 'structural', 
      k8s_mapping: defaultMapping, 
      k8s_namespace: defaultMapping === 'namespace' ? 'new-namespace' : undefined,
      resources: { requests: { cpu: '100m', memory: '128Mi' }, limits: { cpu: '500m', memory: '512Mi' } },
      children: [] 
    };

    const newTreeRoot = updateNode(activeTree.tree, parentId, { children: [...(parent.children || []), newChild] });
    const updated = projectTrees.map(t => t.id === activeTreeId ? { ...t, tree: newTreeRoot } : t);
    saveToStorage(updated);
    setSelectedNodeId(newChild.id);
  };

  const handleDeleteNode = (id) => {
    if (!activeTree || id === 'root') return;
    const newTreeRoot = removeNode(activeTree.tree, id);
    const updated = projectTrees.map(t => t.id === activeTreeId ? { ...t, tree: newTreeRoot } : t);
    setSelectedNodeId('root');
    saveToStorage(updated);
  };

  // Node selection analysis
  const selectedNode = activeTree ? findNode(activeTree.tree, selectedNodeId) : null;
  const lineagePath = (activeTree && selectedNodeId) ? findPathToNode(activeTree.tree, selectedNodeId) : null;
  const ancestors = lineagePath ? lineagePath.slice(0, -1) : [];
  const ancestorWithNamespace = ancestors.find(a => a.k8s_mapping === 'namespace');
  const hasAncestorNamespace = !!ancestorWithNamespace;
  const hasDescendantsWithNamespace = selectedNode ? hasDescendantWithNamespace(selectedNode) : false;

  const canSelectNamespace = !hasAncestorNamespace && !hasDescendantsWithNamespace;

  const k8sMappingOptions = [
    { value: 'tags', label: 'Tags / Labels' },
    { value: 'logical_tenant', label: 'Logical Tenant' },
    ...(canSelectNamespace ? [{ value: 'namespace', label: 'Namespace' }] : []),
    ...(selectedNode?.type !== 'structural' ? [
      { value: 'pod', label: 'Pod (Leaf Only)' },
      { value: 'container', label: 'Container (Leaf Only)' }
    ] : [])
  ];

  const TreeNode = ({ node, depth = 0 }) => {
    const isSelected = node.id === selectedNodeId;
    return (
      <div style={{ marginLeft: depth > 0 ? '20px' : '0', marginTop: '8px' }}>
        <div 
          onClick={() => setSelectedNodeId(node.id)}
          style={{ 
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            padding: '8px 12px', background: isSelected ? '#EEF2FF' : '#FFF',
            border: `1px solid ${isSelected ? '#4F46E5' : '#E2E8F0'}`,
            borderRadius: '6px', cursor: 'pointer', transition: 'all 0.15s ease'
          }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0, flex: 1 }}>
            <span style={{ color: node.type === 'structural' ? '#64748B' : '#4F46E5', flexShrink: 0 }}>
              {node.type === 'structural' ? '📁' : node.type === 'leaf_workflow' ? '⚡️' : '🌐'}
            </span>
            <span style={{ fontWeight: '500', color: '#0F172A', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {node.name}
            </span>
            <span style={{ 
              fontSize: '0.7rem', 
              padding: '2px 6px', 
              borderRadius: '4px', 
              background: node.k8s_mapping === 'namespace' ? '#EFF6FF' : '#F1F5F9',
              color: node.k8s_mapping === 'namespace' ? '#1D4ED8' : '#64748B',
              border: node.k8s_mapping === 'namespace' ? '1px solid #BFDBFE' : '1px solid #E2E8F0',
              flexShrink: 0
            }}>
              {node.k8s_mapping === 'namespace' ? 'K8s Namespace' : node.k8s_mapping}
            </span>
            {node.resources?.limits?.cpu && (
              <span style={{ fontSize: '0.65rem', padding: '1px 5px', borderRadius: '3px', background: '#FEF3C7', color: '#92400E', border: '1px solid #FDE68A' }}>
                {node.resources.limits.cpu} CPU / {node.resources.limits.memory}
              </span>
            )}
          </div>
          <div style={{ display: 'flex', gap: '6px', alignItems: 'center' }}>
            {node.type !== 'structural' && (
              <button 
                title={`Open Studio for ${node.type === 'leaf_workflow' ? 'Workflow' : 'Web App'}`}
                onClick={(e) => { 
                  e.stopPropagation(); 
                  if (onOpenStudio) onOpenStudio({ name: node.name, type: node.type, node }); 
                }}
                style={{
                  background: '#EEF2FF', border: '1px solid #C7D2FE', color: '#4F46E5',
                  borderRadius: '4px', cursor: 'pointer', fontSize: '0.75rem', padding: '2px 8px', fontWeight: '600'
                }}
              >
                🚀 Studio
              </button>
            )}
            <button 
              title="Configure Kubernetes Properties & Resources"
              onClick={(e) => { e.stopPropagation(); setModalNode(node); }}
              style={{
                background: '#F1F5F9', border: '1px solid #CBD5E1', color: '#475569',
                borderRadius: '4px', cursor: 'pointer', fontSize: '0.75rem', padding: '2px 6px', fontWeight: '500'
              }}
            >
              ⚙ Edit K8s
            </button>
            {node.type === 'structural' && (
              <button 
                title="Add child node"
                onClick={(e) => { e.stopPropagation(); handleAddChild(node.id); }} 
                style={{
                  background: '#EEF2FF', border: '1px solid #C7D2FE', color: '#4F46E5', 
                  borderRadius: '4px', cursor: 'pointer', fontSize: '0.85rem', padding: '2px 8px', fontWeight: 'bold', flexShrink: 0
                }}>+ Add</button>
            )}
          </div>
        </div>
        {node.children && node.children.length > 0 && (
          <div style={{ borderLeft: '2px dashed #CBD5E1', paddingLeft: '8px', marginLeft: '10px' }}>
            {node.children.map(child => <TreeNode key={child.id} node={child} depth={depth + 1} />)}
          </div>
        )}
      </div>
    );
  };

  const showLeft = !middleMaximized && !leftCollapsed;
  const showRight = !middleMaximized && !rightCollapsed;

  return (
    <div style={{ display: 'flex', height: '100%', width: '100%', background: '#F8FAFC', overflow: 'hidden', position: 'relative' }}>
      
      {/* Node Properties Modal */}
      {modalNode && (
        <NodePropertiesModal 
          node={modalNode}
          isOpen={!!modalNode}
          onClose={() => setModalNode(null)}
          onSave={handleSaveModalProperties}
          hasAncestorNamespace={findPathToNode(activeTree?.tree, modalNode.id)?.slice(0, -1)?.some(a => a.k8s_mapping === 'namespace')}
          ancestorName={findPathToNode(activeTree?.tree, modalNode.id)?.slice(0, -1)?.find(a => a.k8s_mapping === 'namespace')?.name}
        />
      )}

      {/* LEFT PANE: Project Tree List */}
      {showLeft ? (
        <div style={{ 
          width: `${leftWidth}px`, 
          minWidth: '180px', 
          maxWidth: '500px', 
          background: '#FFF', 
          borderRight: '1px solid #E2E8F0', 
          display: 'flex', 
          flexDirection: 'column', 
          flexShrink: 0,
          position: 'relative'
        }}>
          <div style={{ padding: '16px 20px', borderBottom: '1px solid #E2E8F0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <h2 style={{ margin: 0, fontSize: '1rem', fontWeight: '600', color: '#1E293B' }}>Project Trees</h2>
            <div style={{ display: 'flex', gap: '4px' }}>
              <IconButton title="Collapse Panel" onClick={() => setLeftCollapsed(true)}>◀</IconButton>
            </div>
          </div>

          <div style={{ padding: '12px 16px', borderBottom: '1px solid #F1F5F9' }}>
            <Button primary onClick={handleCreateTree} style={{ width: '100%' }}>+ Create Project Tree</Button>
          </div>

          <div style={{ flex: 1, overflowY: 'auto', padding: '12px 16px' }}>
            {projectTrees.length === 0 ? (
              <div style={{ color: '#64748B', fontSize: '0.875rem', textAlign: 'center' }}>No project trees defined.</div>
            ) : (
              projectTrees.map(tree => (
                <div 
                  key={tree.id} 
                  onClick={() => { setActiveTreeId(tree.id); setSelectedNodeId('root'); }}
                  style={{
                    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                    padding: '9px 12px', borderRadius: '6px', cursor: 'pointer', marginBottom: '6px',
                    background: activeTreeId === tree.id ? '#EEF2FF' : 'transparent',
                    border: activeTreeId === tree.id ? '1px solid #C7D2FE' : '1px solid transparent',
                    color: activeTreeId === tree.id ? '#4F46E5' : '#334155',
                    fontWeight: activeTreeId === tree.id ? '600' : '400',
                    transition: 'all 0.15s ease'
                  }}>
                  <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: '0.875rem' }}>
                    {tree.name}
                  </span>
                  {projectTrees.length > 1 && (
                    <button 
                      onClick={(e) => { e.stopPropagation(); handleDeleteTree(tree.id); }}
                      title="Delete Project Tree"
                      style={{ background: 'none', border: 'none', color: '#94A3B8', cursor: 'pointer', fontSize: '1rem', padding: '0 4px', lineHeight: 1 }}>
                      ×
                    </button>
                  )}
                </div>
              ))
            )}
          </div>

          <div style={{ padding: '8px 12px', borderTop: '1px solid #F1F5F9', background: '#FAFAFA', display: 'flex', gap: '6px', justifyContent: 'center' }}>
            <span style={{ fontSize: '0.7rem', color: '#94A3B8', alignSelf: 'center' }}>Width:</span>
            <IconButton style={{ fontSize: '0.7rem', padding: '2px 6px' }} onClick={() => setLeftWidth(200)}>Compact</IconButton>
            <IconButton style={{ fontSize: '0.7rem', padding: '2px 6px' }} onClick={() => setLeftWidth(280)}>Default</IconButton>
            <IconButton style={{ fontSize: '0.7rem', padding: '2px 6px' }} onClick={() => setLeftWidth(380)}>Wide</IconButton>
          </div>
        </div>
      ) : (
        <div style={{ width: '36px', background: '#FFF', borderRight: '1px solid #E2E8F0', display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '12px 0', flexShrink: 0 }}>
          <IconButton title="Expand Project Trees Panel" onClick={() => { setLeftCollapsed(false); setMiddleMaximized(false); }} style={{ width: '28px', height: '28px', padding: 0 }}>▶</IconButton>
          <div style={{ writingMode: 'vertical-rl', transform: 'rotate(180deg)', marginTop: '20px', fontSize: '0.75rem', fontWeight: '600', color: '#64748B', letterSpacing: '0.05em' }}>
            PROJECT TREES
          </div>
        </div>
      )}

      {/* LEFT SPLITTER RESIZER */}
      {showLeft && (
        <div 
          onMouseDown={startResizeLeft}
          title="Drag to resize panel"
          style={{ width: '6px', cursor: 'col-resize', background: 'transparent', zIndex: 10, flexShrink: 0, transition: 'background 0.2s' }}
          onMouseEnter={e => e.currentTarget.style.background = '#818CF8'}
          onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
        />
      )}

      {/* MIDDLE PANE: Visual Tree Builder */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: '320px', overflow: 'hidden' }}>
        <div style={{ padding: '12px 24px', background: '#FFFFFF', borderBottom: '1px solid #E2E8F0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexShrink: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            {leftCollapsed && (
              <IconButton title="Show Project Trees" onClick={() => setLeftCollapsed(false)}>▶ Show Trees</IconButton>
            )}
            <span style={{ fontSize: '0.85rem', fontWeight: '600', color: '#64748B' }}>
              CANVAS: {activeTree?.name || 'No Project Tree'}
            </span>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <IconButton active={middleMaximized} title={middleMaximized ? "Restore Panels" : "Maximize Tree Canvas"} onClick={() => setMiddleMaximized(!middleMaximized)}>
              {middleMaximized ? '🗗 Restore Layout' : '⛶ Maximize Canvas'}
            </IconButton>
            {rightCollapsed && !middleMaximized && (
              <IconButton title="Show Node Config" onClick={() => setRightCollapsed(false)}>Config ◀</IconButton>
            )}
          </div>
        </div>

        <div style={{ flex: 1, padding: '24px 32px', overflowY: 'auto' }}>
          {activeTree ? (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
                <div>
                  <input 
                    type="text" 
                    value={activeTree.name} 
                    onChange={e => handleUpdateTreeName(e.target.value)} 
                    style={{ fontSize: '1.4rem', fontWeight: '700', color: '#0F172A', border: '1px solid transparent', padding: '4px 8px', borderRadius: '4px', background: 'transparent' }}
                    onFocus={e => e.target.style.background = '#FFF'}
                    onBlur={e => e.target.style.background = 'transparent'}
                    title="Click to rename Project Tree"
                  />
                  <p style={{ margin: '4px 0 0 8px', color: '#64748B', fontSize: '0.85rem' }}>
                    Define hierarchical navigation & Kubernetes mappings. Nested namespaces are strictly restricted.
                  </p>
                </div>
              </div>
              
              <div style={{ maxWidth: '720px', background: '#FFF', padding: '24px', borderRadius: '8px', border: '1px solid #E2E8F0', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
                <TreeNode node={activeTree.tree} />
              </div>
            </div>
          ) : (
            <div style={{ display: 'flex', height: '100%', alignItems: 'center', justifyContent: 'center', color: '#94A3B8' }}>
              Select or create a project tree to edit its structure.
            </div>
          )}
        </div>
      </div>

      {/* RIGHT SPLITTER RESIZER */}
      {showRight && (
        <div 
          onMouseDown={startResizeRight}
          title="Drag to resize panel"
          style={{ width: '6px', cursor: 'col-resize', background: 'transparent', zIndex: 10, flexShrink: 0, transition: 'background 0.2s' }}
          onMouseEnter={e => e.currentTarget.style.background = '#818CF8'}
          onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
        />
      )}

      {/* RIGHT PANE: Node Configuration */}
      {showRight ? (
        <div style={{ 
          width: `${rightWidth}px`, 
          minWidth: '220px', 
          maxWidth: '600px', 
          background: '#FFF', 
          borderLeft: '1px solid #E2E8F0', 
          display: 'flex', 
          flexDirection: 'column', 
          flexShrink: 0,
          position: 'relative'
        }}>
          <div style={{ padding: '16px 20px', borderBottom: '1px solid #E2E8F0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <h3 style={{ margin: 0, fontSize: '1rem', fontWeight: '600', color: '#1E293B' }}>Node Configuration</h3>
            <div style={{ display: 'flex', gap: '4px' }}>
              <IconButton title="Collapse Panel" onClick={() => setRightCollapsed(true)}>▶</IconButton>
            </div>
          </div>

          <div style={{ flex: 1, padding: '20px', overflowY: 'auto' }}>
            {selectedNode ? (
              <div>
                <Input label="Node Name" value={selectedNode.name} onChange={v => handleUpdateNode({ name: v })} />
                
                <div style={{ marginBottom: '20px' }}>
                  <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: '500', marginBottom: '8px', color: '#1E293B' }}>Node Role</label>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.875rem', cursor: 'pointer' }}>
                      <input type="radio" checked={selectedNode.type === 'structural'} onChange={() => handleUpdateNode({ type: 'structural' })} />
                      📁 Structural Group (Can have children)
                    </label>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.875rem', cursor: 'pointer' }}>
                      <input type="radio" checked={selectedNode.type === 'leaf_workflow'} onChange={() => {
                        if (selectedNode.children && selectedNode.children.length > 0) {
                          alert("Cannot convert to Leaf Node: This node currently has children. Remove children first.");
                        } else {
                          handleUpdateNode({ type: 'leaf_workflow', children: undefined });
                        }
                      }} />
                      ⚡️ Workflow Leaf
                    </label>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.875rem', cursor: 'pointer' }}>
                      <input type="radio" checked={selectedNode.type === 'leaf_webapp'} onChange={() => {
                         if (selectedNode.children && selectedNode.children.length > 0) {
                          alert("Cannot convert to Leaf Node: This node currently has children. Remove children first.");
                        } else {
                          handleUpdateNode({ type: 'leaf_webapp', children: undefined });
                        }
                      }} />
                      🌐 Web App Leaf
                    </label>
                  </div>
                </div>

                {selectedNode.type !== 'structural' && (
                  <div style={{ marginBottom: '20px' }}>
                    <Button 
                      primary 
                      onClick={() => {
                        if (onOpenStudio) onOpenStudio({ name: selectedNode.name, type: selectedNode.type, node: selectedNode });
                      }}
                      style={{ width: '100%', background: '#4F46E5', padding: '10px 16px' }}
                    >
                      🚀 Open Studio ({selectedNode.type === 'leaf_workflow' ? 'Workflow Pipeline' : 'Web App Builder'})
                    </Button>
                  </div>
                )}

                <Select 
                  label="Kubernetes Mapping" 
                  value={selectedNode.k8s_mapping} 
                  onChange={v => {
                    if (v === 'namespace' && hasAncestorNamespace) {
                      alert(`Kubernetes Restriction: Ancestor "${ancestorWithNamespace.name}" is already mapped to a Namespace. Kubernetes does not allow nested namespaces.`);
                      return;
                    }
                    if (v === 'namespace' && hasDescendantsWithNamespace) {
                      alert(`Kubernetes Restriction: One of this node's child/descendant nodes is already mapped to a Namespace. Kubernetes does not allow nested namespaces.`);
                      return;
                    }
                    handleUpdateNode({ k8s_mapping: v });
                  }} 
                  options={k8sMappingOptions} 
                />

                {/* Direct Inline K8s properties */}
                {selectedNode.k8s_mapping === 'namespace' && (
                  <div style={{ marginBottom: '16px' }}>
                    <Input 
                      label="Namespace Name" 
                      value={selectedNode.k8s_namespace || selectedNode.name?.toLowerCase().replace(/[^a-z0-9-]/g, '-')} 
                      onChange={v => handleUpdateNode({ k8s_namespace: v })}
                      placeholder="e.g. prod-apps" 
                    />
                  </div>
                )}

                {selectedNode.k8s_mapping === 'logical_tenant' && (
                  <div style={{ marginBottom: '16px' }}>
                    <Input 
                      label="Tenant Name" 
                      value={selectedNode.k8s_tenant || ''} 
                      onChange={v => handleUpdateNode({ k8s_tenant: v })}
                      placeholder="e.g. tenant-alpha" 
                    />
                  </div>
                )}

                <div style={{ marginBottom: '16px' }}>
                  <Input 
                    label="Tags & Labels (Comma-separated)" 
                    value={selectedNode.k8s_labels || ''} 
                    onChange={v => handleUpdateNode({ k8s_labels: v })}
                    placeholder="team=analytics, tier=backend" 
                  />
                </div>

                {/* Resource limits summary & edit trigger */}
                <div style={{ background: '#F8FAFC', border: '1px solid #E2E8F0', padding: '12px', borderRadius: '6px', marginBottom: '16px' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                    <span style={{ fontSize: '0.85rem', fontWeight: '600', color: '#1E293B' }}>Kubernetes Resources</span>
                    <button 
                      onClick={() => setModalNode(selectedNode)}
                      style={{ background: '#EEF2FF', border: '1px solid #C7D2FE', color: '#4F46E5', borderRadius: '4px', cursor: 'pointer', fontSize: '0.75rem', padding: '2px 8px', fontWeight: '600' }}
                    >
                      ⚙ Configure All
                    </button>
                  </div>
                  <div style={{ fontSize: '0.78rem', color: '#475569', display: 'flex', flexDirection: 'column', gap: '4px' }}>
                    <div><strong>CPU:</strong> {selectedNode.resources?.requests?.cpu || '100m'} (Req) / {selectedNode.resources?.limits?.cpu || '500m'} (Lim)</div>
                    <div><strong>Memory:</strong> {selectedNode.resources?.requests?.memory || '128Mi'} (Req) / {selectedNode.resources?.limits?.memory || '512Mi'} (Lim)</div>
                  </div>
                </div>

                {/* Explanatory validation notices */}
                {hasAncestorNamespace && (
                  <div style={{ background: '#F8FAFC', border: '1px solid #E2E8F0', color: '#475569', padding: '10px 12px', borderRadius: '6px', fontSize: '0.8rem', marginTop: '-4px', marginBottom: '16px', lineHeight: '1.4' }}>
                    <span style={{ fontWeight: '600', color: '#0F172A', display: 'block', marginBottom: '2px' }}>🔒 Namespace Option Hidden</span>
                    Kubernetes does not permit nested namespaces. Ancestor node <strong>"{ancestorWithNamespace.name}"</strong> is already assigned as a Namespace, so child nodes cannot select Namespace.
                  </div>
                )}

                {!hasAncestorNamespace && hasDescendantsWithNamespace && (
                  <div style={{ background: '#FFFBEB', border: '1px solid #FDE68A', color: '#92400E', padding: '10px 12px', borderRadius: '6px', fontSize: '0.8rem', marginTop: '-4px', marginBottom: '16px', lineHeight: '1.4' }}>
                    <span style={{ fontWeight: '600', color: '#78350F', display: 'block', marginBottom: '2px' }}>⚠️ Descendant Namespace Detected</span>
                    A descendant of this node is already mapped as a Namespace. This node cannot be configured as a Namespace unless the descendant mapping is changed.
                  </div>
                )}

                {selectedNode.k8s_mapping === 'namespace' && (
                  <div style={{ background: '#EFF6FF', border: '1px solid #BFDBFE', color: '#1E40AF', padding: '10px 12px', borderRadius: '6px', fontSize: '0.8rem', marginTop: '-4px', marginBottom: '16px' }}>
                    ✓ <strong>Primary K8s Namespace:</strong> Workflows beneath this node will deploy into this Kubernetes Namespace.
                  </div>
                )}

                {selectedNode.id !== 'root' && (
                  <div style={{ marginTop: '36px', paddingTop: '20px', borderTop: '1px solid #E2E8F0' }}>
                    <Button danger onClick={() => handleDeleteNode(selectedNode.id)} style={{ width: '100%' }}>Delete Node</Button>
                  </div>
                )}
              </div>
            ) : (
              <div style={{ color: '#94A3B8', fontSize: '0.875rem', textAlign: 'center', marginTop: '40px' }}>
                Select a node to edit its properties.
              </div>
            )}
          </div>

          <div style={{ padding: '8px 12px', borderTop: '1px solid #F1F5F9', background: '#FAFAFA', display: 'flex', gap: '6px', justifyContent: 'center' }}>
            <span style={{ fontSize: '0.7rem', color: '#94A3B8', alignSelf: 'center' }}>Width:</span>
            <IconButton style={{ fontSize: '0.7rem', padding: '2px 6px' }} onClick={() => setRightWidth(260)}>Compact</IconButton>
            <IconButton style={{ fontSize: '0.7rem', padding: '2px 6px' }} onClick={() => setRightWidth(340)}>Default</IconButton>
            <IconButton style={{ fontSize: '0.7rem', padding: '2px 6px' }} onClick={() => setRightWidth(460)}>Wide</IconButton>
          </div>
        </div>
      ) : (
        <div style={{ width: '36px', background: '#FFF', borderLeft: '1px solid #E2E8F0', display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '12px 0', flexShrink: 0 }}>
          <IconButton title="Expand Node Configuration Panel" onClick={() => { setRightCollapsed(false); setMiddleMaximized(false); }} style={{ width: '28px', height: '28px', padding: 0 }}>◀</IconButton>
          <div style={{ writingMode: 'vertical-rl', marginTop: '20px', fontSize: '0.75rem', fontWeight: '600', color: '#64748B', letterSpacing: '0.05em' }}>
            NODE CONFIG
          </div>
        </div>
      )}

    </div>
  );
}

// Backwards compatibility export
export const OrganizationSettings = ProjectTreeSettings;
