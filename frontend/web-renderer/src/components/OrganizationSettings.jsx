import React, { useState, useEffect } from 'react';

const Button = ({ children, primary, secondary, danger, onClick, style, disabled }) => {
  let bg = '#FFFFFF', color = '#334155', border = '1px solid #CBD5E1';
  if (primary) { bg = disabled ? '#94A3B8' : '#4F46E5'; color = '#FFFFFF'; border = 'none'; }
  else if (secondary) { bg = '#F1F5F9'; border = 'none'; }
  else if (danger) { bg = '#FEF2F2'; color = '#EF4444'; border = '1px solid #FECACA'; }
  
  return (
    <button onClick={onClick} disabled={disabled} style={{
      background: bg, color, border, padding: '8px 16px', borderRadius: '6px',
      fontWeight: '500', cursor: disabled ? 'not-allowed' : 'pointer',
      fontSize: '0.875rem', transition: 'all 0.2s', ...style
    }}>
      {children}
    </button>
  );
};

const Input = ({ label, value, onChange, placeholder }) => (
  <div style={{ marginBottom: '16px' }}>
    {label && <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: '500', marginBottom: '6px' }}>{label}</label>}
    <input type="text" value={value} onChange={e => onChange(e.target.value)} placeholder={placeholder} style={{
      width: '100%', padding: '10px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box'
    }} />
  </div>
);

const Select = ({ label, value, onChange, options }) => (
  <div style={{ marginBottom: '16px' }}>
    {label && <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: '500', marginBottom: '6px' }}>{label}</label>}
    <select value={value} onChange={e => onChange(e.target.value)} style={{
      width: '100%', padding: '10px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box', background: '#FFF'
    }}>
      {options.map(opt => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
    </select>
  </div>
);

export function OrganizationSettings({ onOrganizationChanged }) {
  const [orgs, setOrgs] = useState([]);
  const [activeOrgId, setActiveOrgId] = useState(null);
  const [selectedNodeId, setSelectedNodeId] = useState(null);
  
  // Load from local storage for prototype
  useEffect(() => {
    const saved = localStorage.getItem('nextkube_orgs');
    if (saved) {
      const parsed = JSON.parse(saved);
      setOrgs(parsed);
      if (parsed.length > 0) setActiveOrgId(parsed[0].id);
    }
  }, []);

  const saveToStorage = (updatedOrgs) => {
    setOrgs(updatedOrgs);
    localStorage.setItem('nextkube_orgs', JSON.stringify(updatedOrgs));
    if (onOrganizationChanged) onOrganizationChanged(updatedOrgs.find(o => o.id === activeOrgId));
  };

  const activeOrg = orgs.find(o => o.id === activeOrgId);

  const handleCreateOrg = () => {
    const newOrg = {
      id: 'org_' + Date.now(),
      name: 'New Organization',
      tree: { id: 'root', name: 'Root', type: 'structural', k8s_mapping: 'namespace', children: [] }
    };
    const newOrgs = [...orgs, newOrg];
    setOrgs(newOrgs);
    setActiveOrgId(newOrg.id);
    setSelectedNodeId('root');
    saveToStorage(newOrgs);
  };

  // Tree manipulation helpers
  const findNode = (node, id) => {
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
    if (node.id === id) return { ...node, ...updates };
    if (node.children) {
      return { ...node, children: node.children.map(c => updateNode(c, id, updates)) };
    }
    return node;
  };

  const removeNode = (node, id) => {
    if (!node.children) return node;
    return {
      ...node,
      children: node.children.filter(c => c.id !== id).map(c => removeNode(c, id))
    };
  };

  const handleUpdateNode = (updates) => {
    if (!activeOrg || !selectedNodeId) return;
    const newTree = updateNode(activeOrg.tree, selectedNodeId, updates);
    const updatedOrgs = orgs.map(o => o.id === activeOrgId ? { ...o, tree: newTree } : o);
    saveToStorage(updatedOrgs);
  };

  const handleAddChild = (parentId) => {
    const parent = findNode(activeOrg.tree, parentId);
    if (parent.type !== 'structural') return; // Cannot add children to leaves
    const newChild = { id: 'node_' + Date.now(), name: 'New Node', type: 'structural', k8s_mapping: 'tags', children: [] };
    const newTree = updateNode(activeOrg.tree, parentId, { children: [...(parent.children || []), newChild] });
    const updatedOrgs = orgs.map(o => o.id === activeOrgId ? { ...o, tree: newTree } : o);
    saveToStorage(updatedOrgs);
    setSelectedNodeId(newChild.id);
  };

  const handleDeleteNode = (id) => {
    if (id === 'root') return; // Cannot delete root
    const newTree = removeNode(activeOrg.tree, id);
    const updatedOrgs = orgs.map(o => o.id === activeOrgId ? { ...o, tree: newTree } : o);
    setSelectedNodeId(null);
    saveToStorage(updatedOrgs);
  };

  const hasNamespaceInLineage = (tree, targetId, currentHasNamespace = false) => {
    const isNamespace = tree.k8s_mapping === 'namespace';
    const effectiveNamespace = currentHasNamespace || isNamespace;
    if (tree.id === targetId) return currentHasNamespace; // check if ancestors have it
    if (tree.children) {
      for (let c of tree.children) {
        if (hasNamespaceInLineage(c, targetId, effectiveNamespace)) return true;
      }
    }
    return false;
  };

  const TreeNode = ({ node, depth = 0 }) => {
    const isSelected = node.id === selectedNodeId;
    return (
      <div style={{ marginLeft: depth > 0 ? '24px' : '0', marginTop: '8px' }}>
        <div 
          onClick={() => setSelectedNodeId(node.id)}
          style={{ 
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            padding: '8px 12px', background: isSelected ? '#EEF2FF' : '#FFF',
            border: `1px solid ${isSelected ? '#4F46E5' : '#E2E8F0'}`,
            borderRadius: '6px', cursor: 'pointer', transition: 'all 0.2s'
          }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span style={{ color: node.type === 'structural' ? '#64748B' : '#4F46E5' }}>
              {node.type === 'structural' ? '📁' : node.type === 'leaf_workflow' ? '⚡️' : '🌐'}
            </span>
            <span style={{ fontWeight: '500', color: '#0F172A' }}>{node.name}</span>
          </div>
          {node.type === 'structural' && (
            <button onClick={(e) => { e.stopPropagation(); handleAddChild(node.id); }} style={{
              background: 'none', border: 'none', color: '#4F46E5', cursor: 'pointer', fontSize: '1.2rem', padding: '0 4px'
            }}>+</button>
          )}
        </div>
        {node.children && node.children.length > 0 && (
          <div style={{ borderLeft: '1px solid #E2E8F0', paddingLeft: '8px', marginLeft: '12px' }}>
            {node.children.map(child => <TreeNode key={child.id} node={child} depth={depth + 1} />)}
          </div>
        )}
      </div>
    );
  };

  const selectedNode = activeOrg ? findNode(activeOrg.tree, selectedNodeId) : null;
  const ancestorHasNamespace = activeOrg && selectedNode ? hasNamespaceInLineage(activeOrg.tree, selectedNodeId) : false;

  return (
    <div style={{ display: 'flex', height: '100%', background: '#F8FAFC' }}>
      {/* Left Pane: Org List */}
      <div style={{ width: '250px', background: '#FFF', borderRight: '1px solid #E2E8F0', display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: '20px', borderBottom: '1px solid #E2E8F0' }}>
          <h2 style={{ margin: '0 0 16px 0', fontSize: '1.1rem', fontWeight: '600' }}>Organizations</h2>
          <Button primary onClick={handleCreateOrg} style={{ width: '100%' }}>+ Create Org</Button>
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: '16px' }}>
          {orgs.length === 0 ? (
            <div style={{ color: '#64748B', fontSize: '0.875rem', textAlign: 'center' }}>No organizations defined.</div>
          ) : (
            orgs.map(org => (
              <div 
                key={org.id} 
                onClick={() => { setActiveOrgId(org.id); setSelectedNodeId('root'); }}
                style={{
                  padding: '10px 12px', borderRadius: '6px', cursor: 'pointer', marginBottom: '8px',
                  background: activeOrgId === org.id ? '#4F46E5' : 'transparent',
                  color: activeOrgId === org.id ? '#FFF' : '#334155',
                  fontWeight: activeOrgId === org.id ? '500' : '400'
                }}>
                {org.name}
              </div>
            ))
          )}
        </div>
      </div>

      {/* Middle Pane: Visual Builder */}
      <div style={{ flex: 1, padding: '32px', overflowY: 'auto' }}>
        {activeOrg ? (
          <div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '24px' }}>
              <h2 style={{ margin: 0, fontSize: '1.5rem', fontWeight: '600' }}>Organization Tree</h2>
            </div>
            <div style={{ maxWidth: '600px', background: '#FFF', padding: '24px', borderRadius: '8px', border: '1px solid #E2E8F0' }}>
              <TreeNode node={activeOrg.tree} />
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', height: '100%', alignItems: 'center', justifyContent: 'center', color: '#94A3B8' }}>
            Select or create an organization to edit its structure.
          </div>
        )}
      </div>

      {/* Right Pane: Node Config */}
      <div style={{ width: '350px', background: '#FFF', borderLeft: '1px solid #E2E8F0', padding: '24px', display: 'flex', flexDirection: 'column' }}>
        <h3 style={{ margin: '0 0 24px 0', fontSize: '1.1rem', fontWeight: '600' }}>Node Configuration</h3>
        {selectedNode ? (
          <div style={{ flex: 1 }}>
            <Input label="Node Name" value={selectedNode.name} onChange={v => handleUpdateNode({ name: v })} />
            
            <div style={{ marginBottom: '24px' }}>
              <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: '500', marginBottom: '8px' }}>Node Type</label>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.875rem' }}>
                  <input type="radio" checked={selectedNode.type === 'structural'} onChange={() => handleUpdateNode({ type: 'structural' })} />
                  📁 Structural Group
                </label>
                <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.875rem' }}>
                  <input type="radio" checked={selectedNode.type === 'leaf_workflow'} onChange={() => {
                    // Prevent changing to leaf if it has children
                    if (selectedNode.children && selectedNode.children.length > 0) {
                      alert("Cannot convert to Leaf Node: This node currently has children. Remove children first.");
                    } else {
                      handleUpdateNode({ type: 'leaf_workflow', children: undefined });
                    }
                  }} />
                  ⚡️ Workflow Leaf
                </label>
                <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.875rem' }}>
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

            <Select 
              label="Kubernetes Mapping" 
              value={selectedNode.k8s_mapping} 
              onChange={v => {
                if (v === 'namespace' && ancestorHasNamespace) {
                  // Reject namespace if an ancestor already is a namespace
                  alert("Invalid Mapping: Kubernetes namespaces are flat and cannot be nested. An ancestor node is already mapped to a namespace. Please use 'Tags' or 'Logical Tenant'.");
                  return;
                }
                handleUpdateNode({ k8s_mapping: v });
              }} 
              options={[
                { value: 'tags', label: 'Tags / Labels' },
                { value: 'logical_tenant', label: 'Logical Tenant' },
                { value: 'namespace', label: 'Namespace' },
                { value: 'pod', label: 'Pod (Leaf Only)' },
                { value: 'container', label: 'Container (Leaf Only)' }
              ]} 
            />

            {selectedNode.k8s_mapping === 'namespace' && ancestorHasNamespace && (
              <div style={{ background: '#FEF2F2', border: '1px solid #FECACA', color: '#991B1B', padding: '12px', borderRadius: '6px', fontSize: '0.8rem', marginTop: '-8px', marginBottom: '16px' }}>
                Warning: Kubernetes namespaces are flat. Nested namespaces are not supported natively.
              </div>
            )}

            {selectedNode.id !== 'root' && (
              <div style={{ marginTop: '48px', paddingTop: '24px', borderTop: '1px solid #E2E8F0' }}>
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
    </div>
  );
}
