import React, { useState } from 'react';

export function NodePropertiesModal({ node, isOpen, onClose, onSave, hasAncestorNamespace, ancestorName }) {
  if (!isOpen || !node) return null;

  // Local editing state
  const [name, setName] = useState(node.name || '');
  const [k8sMapping, setK8sMapping] = useState(node.k8s_mapping || 'tags');
  const [namespaceName, setNamespaceName] = useState(node.k8s_namespace || (node.k8s_mapping === 'namespace' ? node.name?.toLowerCase().replace(/[^a-z0-9-]/g, '-') : 'default'));
  const [tenantName, setTenantName] = useState(node.k8s_tenant || '');
  const [labels, setLabels] = useState(node.k8s_labels || '');
  const [annotations, setAnnotations] = useState(node.k8s_annotations || '');
  
  // Resources
  const [cpuRequest, setCpuRequest] = useState(node.resources?.requests?.cpu || '100m');
  const [memRequest, setMemRequest] = useState(node.resources?.requests?.memory || '128Mi');
  const [cpuLimit, setCpuLimit] = useState(node.resources?.limits?.cpu || '500m');
  const [memLimit, setMemLimit] = useState(node.resources?.limits?.memory || '512Mi');
  
  // Advanced K8s
  const [nodeSelector, setNodeSelector] = useState(node.k8s_node_selector || '');
  const [serviceAccount, setServiceAccount] = useState(node.k8s_service_account || 'default');
  const [restartPolicy, setRestartPolicy] = useState(node.k8s_restart_policy || 'Always');

  const [activeTab, setActiveTab] = useState('general');

  const handleSave = () => {
    onSave(node.id, {
      name,
      k8s_mapping: k8sMapping,
      k8s_namespace: namespaceName,
      k8s_tenant: tenantName,
      k8s_labels: labels,
      k8s_annotations: annotations,
      k8s_node_selector: nodeSelector,
      k8s_service_account: serviceAccount,
      k8s_restart_policy: restartPolicy,
      resources: {
        requests: { cpu: cpuRequest, memory: memRequest },
        limits: { cpu: cpuLimit, memory: memLimit }
      }
    });
    onClose();
  };

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
      background: 'rgba(15, 23, 42, 0.65)', backdropFilter: 'blur(3px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      zIndex: 9999, padding: '20px'
    }} onClick={onClose}>
      <div style={{
        background: '#FFFFFF', borderRadius: '12px', width: '640px', maxWidth: '95vw',
        maxHeight: '88vh', display: 'flex', flexDirection: 'column',
        boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.2), 0 8px 10px -6px rgba(0, 0, 0, 0.1)',
        overflow: 'hidden', border: '1px solid #E2E8F0'
      }} onClick={e => e.stopPropagation()}>
        
        {/* Header */}
        <div style={{ padding: '18px 24px', borderBottom: '1px solid #E2E8F0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: '#F8FAFC' }}>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span style={{ fontSize: '1.25rem' }}>{node.type === 'structural' ? '📁' : node.type === 'leaf_workflow' ? '⚡️' : '🌐'}</span>
              <h3 style={{ margin: 0, fontSize: '1.15rem', fontWeight: '700', color: '#0F172A' }}>Configure Node: {node.name}</h3>
            </div>
            <p style={{ margin: '4px 0 0 0', fontSize: '0.8rem', color: '#64748B' }}>
              Define Kubernetes namespace, tenant isolation, tags/labels, and resource quotas.
            </p>
          </div>
          <button onClick={onClose} style={{ background: 'none', border: 'none', fontSize: '1.4rem', color: '#94A3B8', cursor: 'pointer', lineHeight: 1 }}>×</button>
        </div>

        {/* Tab Navigation */}
        <div style={{ display: 'flex', borderBottom: '1px solid #E2E8F0', background: '#FFF', padding: '0 24px' }}>
          {[
            { id: 'general', label: 'General & K8s Mapping' },
            { id: 'resources', label: 'Resource Limits & Quotas' },
            { id: 'advanced', label: 'Labels, RBAC & Pod Specs' },
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              style={{
                padding: '12px 16px',
                border: 'none',
                background: 'transparent',
                borderBottom: activeTab === tab.id ? '2px solid #4F46E5' : '2px solid transparent',
                color: activeTab === tab.id ? '#4F46E5' : '#64748B',
                fontWeight: activeTab === tab.id ? '600' : '500',
                fontSize: '0.875rem',
                cursor: 'pointer'
              }}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Body */}
        <div style={{ padding: '24px', overflowY: 'auto', flex: 1, display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {activeTab === 'general' && (
            <>
              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px', color: '#334155' }}>Node Display Name</label>
                <input 
                  type="text" 
                  value={name} 
                  onChange={e => setName(e.target.value)} 
                  placeholder="e.g. Payments Service"
                  style={{ width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box' }}
                />
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px', color: '#334155' }}>Kubernetes Mapping Mode</label>
                <select 
                  value={k8sMapping} 
                  onChange={e => setK8sMapping(e.target.value)}
                  style={{ width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box', background: '#FFF' }}
                >
                  <option value="tags">Tags / Labels (Inherits parent namespace)</option>
                  <option value="logical_tenant">Logical Tenant (Multi-tenancy grouping)</option>
                  {!hasAncestorNamespace && <option value="namespace">Kubernetes Namespace (Isolated K8s boundary)</option>}
                  {node.type !== 'structural' && <option value="pod">Kubernetes Pod (Workload Leaf)</option>}
                  {node.type !== 'structural' && <option value="container">Kubernetes Container</option>}
                </select>
                {hasAncestorNamespace && (
                  <p style={{ margin: '6px 0 0 0', fontSize: '0.78rem', color: '#64748B' }}>
                    🔒 <em>Kubernetes Namespace is disabled because ancestor "{ancestorName || 'Parent'}" is already mapped to a Namespace. Kubernetes does not allow nested namespaces.</em>
                  </p>
                )}
              </div>

              {k8sMapping === 'namespace' ? (
                <div style={{ background: '#EFF6FF', border: '1px solid #BFDBFE', padding: '16px', borderRadius: '8px' }}>
                  <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px', color: '#1E40AF' }}>Kubernetes Namespace Name</label>
                  <input 
                    type="text" 
                    value={namespaceName} 
                    onChange={e => setNamespaceName(e.target.value)} 
                    placeholder="e.g. prod-payments"
                    style={{ width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #93C5FD', fontSize: '0.875rem', boxSizing: 'border-box', background: '#FFF' }}
                  />
                  <span style={{ fontSize: '0.75rem', color: '#1E40AF', marginTop: '4px', display: 'block' }}>
                    A dedicated Kubernetes Namespace will be created/bound for this node.
                  </span>
                </div>
              ) : k8sMapping === 'logical_tenant' ? (
                <div style={{ background: '#F8FAFC', border: '1px solid #E2E8F0', padding: '16px', borderRadius: '8px' }}>
                  <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px', color: '#334155' }}>Tenant Identifier</label>
                  <input 
                    type="text" 
                    value={tenantName} 
                    onChange={e => setTenantName(e.target.value)} 
                    placeholder="e.g. tenant-alpha, enterprise-customer"
                    style={{ width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box' }}
                  />
                  <span style={{ fontSize: '0.75rem', color: '#64748B', marginTop: '4px', display: 'block' }}>
                    Workloads will be tagged with tenant context metadata for security and quota isolation.
                  </span>
                </div>
              ) : (
                <div style={{ background: '#F8FAFC', border: '1px solid #E2E8F0', padding: '14px', borderRadius: '8px', fontSize: '0.8rem', color: '#475569' }}>
                  ℹ️ Workloads will deploy into the parent's resolved Kubernetes Namespace with custom tags and label selectors applied.
                </div>
              )}
            </>
          )}

          {activeTab === 'resources' && (
            <>
              <div style={{ background: '#F8FAFC', border: '1px solid #E2E8F0', padding: '12px 16px', borderRadius: '8px', marginBottom: '8px' }}>
                <span style={{ fontSize: '0.85rem', fontWeight: '600', color: '#1E293B', display: 'block', marginBottom: '4px' }}>
                  ⚡ Kubernetes Compute Resource Quotas
                </span>
                <span style={{ fontSize: '0.78rem', color: '#64748B' }}>
                  Specify resource requests and limits enforced by the Kubernetes scheduler and container runtime.
                </span>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                <div style={{ border: '1px solid #E2E8F0', padding: '16px', borderRadius: '8px' }}>
                  <h4 style={{ margin: '0 0 12px 0', fontSize: '0.85rem', fontWeight: '700', color: '#4F46E5', textTransform: 'uppercase' }}>Requests (Guaranteed)</h4>
                  <div style={{ marginBottom: '12px' }}>
                    <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: '500', marginBottom: '4px', color: '#475569' }}>CPU Request</label>
                    <input 
                      type="text" 
                      value={cpuRequest} 
                      onChange={e => setCpuRequest(e.target.value)} 
                      placeholder="e.g. 100m, 0.5"
                      style={{ width: '100%', padding: '8px 10px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.85rem', boxSizing: 'border-box' }}
                    />
                  </div>
                  <div>
                    <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: '500', marginBottom: '4px', color: '#475569' }}>Memory Request</label>
                    <input 
                      type="text" 
                      value={memRequest} 
                      onChange={e => setMemRequest(e.target.value)} 
                      placeholder="e.g. 128Mi, 1Gi"
                      style={{ width: '100%', padding: '8px 10px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.85rem', boxSizing: 'border-box' }}
                    />
                  </div>
                </div>

                <div style={{ border: '1px solid #E2E8F0', padding: '16px', borderRadius: '8px' }}>
                  <h4 style={{ margin: '0 0 12px 0', fontSize: '0.85rem', fontWeight: '700', color: '#EF4444', textTransform: 'uppercase' }}>Limits (Ceiling)</h4>
                  <div style={{ marginBottom: '12px' }}>
                    <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: '500', marginBottom: '4px', color: '#475569' }}>CPU Limit</label>
                    <input 
                      type="text" 
                      value={cpuLimit} 
                      onChange={e => setCpuLimit(e.target.value)} 
                      placeholder="e.g. 500m, 2"
                      style={{ width: '100%', padding: '8px 10px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.85rem', boxSizing: 'border-box' }}
                    />
                  </div>
                  <div>
                    <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: '500', marginBottom: '4px', color: '#475569' }}>Memory Limit</label>
                    <input 
                      type="text" 
                      value={memLimit} 
                      onChange={e => setMemLimit(e.target.value)} 
                      placeholder="e.g. 512Mi, 4Gi"
                      style={{ width: '100%', padding: '8px 10px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.85rem', boxSizing: 'border-box' }}
                    />
                  </div>
                </div>
              </div>
            </>
          )}

          {activeTab === 'advanced' && (
            <>
              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px', color: '#334155' }}>Kubernetes Labels (Key=Value, comma-separated)</label>
                <input 
                  type="text" 
                  value={labels} 
                  onChange={e => setLabels(e.target.value)} 
                  placeholder="env=production, team=billing, tier=backend"
                  style={{ width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box' }}
                />
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px', color: '#334155' }}>Annotations (Key=Value, comma-separated)</label>
                <input 
                  type="text" 
                  value={annotations} 
                  onChange={e => setAnnotations(e.target.value)} 
                  placeholder="prometheus.io/scrape=true, nextkube.io/managed=true"
                  style={{ width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box' }}
                />
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                <div>
                  <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px', color: '#334155' }}>K8s ServiceAccount</label>
                  <input 
                    type="text" 
                    value={serviceAccount} 
                    onChange={e => setServiceAccount(e.target.value)} 
                    placeholder="default"
                    style={{ width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box' }}
                  />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px', color: '#334155' }}>Node Selector (Optional)</label>
                  <input 
                    type="text" 
                    value={nodeSelector} 
                    onChange={e => setNodeSelector(e.target.value)} 
                    placeholder="kubernetes.io/arch=arm64"
                    style={{ width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box' }}
                  />
                </div>
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px', color: '#334155' }}>Restart Policy</label>
                <select 
                  value={restartPolicy} 
                  onChange={e => setRestartPolicy(e.target.value)}
                  style={{ width: '100%', padding: '9px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '0.875rem', boxSizing: 'border-box', background: '#FFF' }}
                >
                  <option value="Always">Always</option>
                  <option value="OnFailure">OnFailure</option>
                  <option value="Never">Never</option>
                </select>
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div style={{ padding: '16px 24px', borderTop: '1px solid #E2E8F0', display: 'flex', justifyContent: 'flex-end', gap: '12px', background: '#F8FAFC' }}>
          <button 
            type="button" 
            onClick={onClose} 
            style={{ padding: '8px 16px', borderRadius: '6px', border: '1px solid #CBD5E1', background: '#FFF', color: '#475569', fontWeight: '500', cursor: 'pointer', fontSize: '0.875rem' }}
          >
            Cancel
          </button>
          <button 
            type="button" 
            onClick={handleSave} 
            style={{ padding: '8px 20px', borderRadius: '6px', border: 'none', background: '#4F46E5', color: '#FFF', fontWeight: '600', cursor: 'pointer', fontSize: '0.875rem' }}
          >
            Save Kubernetes Settings
          </button>
        </div>

      </div>
    </div>
  );
}
