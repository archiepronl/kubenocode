import React, { useState, useEffect } from 'react';
import * as yaml from 'js-yaml';

// Studio UI Sub-Components
const StudioButton = ({ children, primary, secondary, danger, success, onClick, style, disabled }) => {
  let bg = '#FFFFFF', color = '#334155', border = '1px solid #CBD5E1';
  if (primary) { bg = disabled ? '#94A3B8' : '#4F46E5'; color = '#FFFFFF'; border = 'none'; }
  else if (secondary) { bg = '#F1F5F9'; border = '1px solid #E2E8F0'; }
  else if (success) { bg = '#10B981'; color = '#FFFFFF'; border = 'none'; }
  else if (danger) { bg = '#FEF2F2'; color = '#EF4444'; border = '1px solid #FECACA'; }

  return (
    <button onClick={onClick} disabled={disabled} style={{
      background: bg, color, border, padding: '8px 16px', borderRadius: '6px',
      fontWeight: '600', cursor: disabled ? 'not-allowed' : 'pointer',
      fontSize: '0.85rem', transition: 'all 0.15s ease', display: 'inline-flex',
      alignItems: 'center', justifyContent: 'center', gap: '6px', ...style
    }}>
      {children}
    </button>
  );
};

export function Studio({ item, onBack }) {
  // item can be string (legacy appName) or object { name, type: 'leaf_workflow' | 'leaf_webapp', node }
  const itemName = typeof item === 'string' ? item : (item?.name || 'My Workload');
  const initialType = typeof item === 'object' && item?.type ? item.type : 'leaf_workflow';

  // Studio Mode: 'workflow' (Backend Pipeline) vs 'webapp' (Frontend UI)
  const [studioMode, setStudioMode] = useState(initialType === 'leaf_webapp' ? 'webapp' : 'workflow');
  
  // View mode inside current studio: 'visual' | 'yaml' | 'preview'
  const [viewMode, setViewMode] = useState('visual');

  // Name & Description
  const [name, setName] = useState(itemName);
  const [description, setDescription] = useState(
    initialType === 'leaf_webapp' 
      ? 'Interactive low-code Web App connected to backend Kubernetes workflows.'
      : 'Automated data orchestration pipeline deployed on Kubernetes.'
  );

  // Workflow Pipeline State
  const [workflowNodes, setWorkflowNodes] = useState([
    { id: 'step-1', type: 'trigger', name: 'HTTP Webhook Ingress', icon: '⚡️', details: 'POST /api/v1/intake', k8sRole: 'Service/Ingress' },
    { id: 'step-2', type: 'processor', name: 'Python Transform', icon: '🐍', details: 'normalize_payload.py (Micro-batch)', k8sRole: 'Worker Pod' },
    { id: 'step-3', type: 'database', name: 'PostgreSQL Store', icon: '🗄️', details: 'INSERT INTO transactions', k8sRole: 'DB Connector' },
    { id: 'step-4', type: 'webapp_target', name: 'Notify Web App', icon: '🌐', details: 'Emit event to Customer Portal', k8sRole: 'Event Bus' }
  ]);
  const [selectedWorkflowStep, setSelectedWorkflowStep] = useState('step-1');

  // Web App UI Components State (for Leaf Node as Web App)
  const [webAppComponents, setWebAppComponents] = useState([
    { id: 'c-1', type: 'header', label: 'Order Intake & Analytics', subtitle: 'Submit new tasks to the execution engine' },
    { id: 'c-2', type: 'input', label: 'Customer Name', placeholder: 'e.g. Acme Corp', required: true, fieldName: 'customer' },
    { id: 'c-3', type: 'select', label: 'Priority Tier', options: ['Standard', 'Expedited', 'Critical'], fieldName: 'priority' },
    { id: 'c-4', type: 'file', label: 'Batch Upload CSV', fieldName: 'batch_file' },
    { id: 'c-5', type: 'button', label: 'Dispatch Workflow', action: 'trigger_workflow', workflowTarget: itemName }
  ]);
  const [selectedComponentId, setSelectedComponentId] = useState('c-1');

  // Live Test Run / Preview Execution States
  const [formValues, setFormValues] = useState({ customer: 'Acme Global', priority: 'Critical' });
  const [executionLogs, setExecutionLogs] = useState([]);
  const [isExecuting, setIsExecuting] = useState(false);

  // Sync to YAML/JSON
  const [yamlContent, setYamlContent] = useState('');

  useEffect(() => {
    const payload = studioMode === 'workflow' ? {
      apiVersion: 'flowengine.io/v1alpha1',
      kind: 'KttmApp',
      metadata: { name: name.toLowerCase().replace(/[^a-z0-9-]/g, '-'), namespace: item?.node?.k8s_namespace || 'default' },
      spec: {
        type: 'workflow',
        steps: workflowNodes.map(n => ({ id: n.id, type: n.type, name: n.name, details: n.details }))
      }
    } : {
      apiVersion: 'flowengine.io/v1alpha1',
      kind: 'KttmWebApp',
      metadata: { name: name.toLowerCase().replace(/[^a-z0-9-]/g, '-'), namespace: item?.node?.k8s_namespace || 'default' },
      spec: {
        type: 'webapp',
        boundWorkflow: itemName,
        components: webAppComponents
      }
    };
    setYamlContent(yaml.dump(payload));
  }, [studioMode, workflowNodes, webAppComponents, name, item, itemName]);

  // Workflow Catalog item addition
  const addWorkflowNode = (type, nodeName, icon, details, k8sRole) => {
    const newNode = {
      id: 'step-' + Date.now(),
      type,
      name: nodeName,
      icon,
      details,
      k8sRole
    };
    setWorkflowNodes([...workflowNodes, newNode]);
    setSelectedWorkflowStep(newNode.id);
  };

  // Web App Component addition
  const addWebAppComponent = (type, label) => {
    const newComp = {
      id: 'c-' + Date.now(),
      type,
      label,
      placeholder: `Enter ${label}...`,
      fieldName: label.toLowerCase().replace(/[^a-z0-9_]/g, '_')
    };
    setWebAppComponents([...webAppComponents, newComp]);
    setSelectedComponentId(newComp.id);
  };

  // Execute workflow from live preview
  const handleRunExecution = () => {
    setIsExecuting(true);
    setExecutionLogs([
      `[${new Date().toLocaleTimeString()}] Initializing execution context for ${name}...`,
      `[${new Date().toLocaleTimeString()}] Resolved K8s Namespace: "${item?.node?.k8s_namespace || 'default'}"`,
      `[${new Date().toLocaleTimeString()}] Submitting form payload: ${JSON.stringify(formValues)}`
    ]);

    setTimeout(() => {
      setExecutionLogs(prev => [
        ...prev,
        `[${new Date().toLocaleTimeString()}] Step 1 [HTTP Webhook Ingress]: Accepted payload HTTP 202`,
        `[${new Date().toLocaleTimeString()}] Step 2 [Python Transform]: Executed normalize_payload.py (18ms)`
      ]);
    }, 600);

    setTimeout(() => {
      setExecutionLogs(prev => [
        ...prev,
        `[${new Date().toLocaleTimeString()}] Step 3 [PostgreSQL Store]: Record committed with ID #TX-9841`,
        `[${new Date().toLocaleTimeString()}] Step 4 [Notify Web App]: Event dispatched successfully!`,
        `[${new Date().toLocaleTimeString()}] ✅ Pipeline Execution Completed with Status: SUCCESS`
      ]);
      setIsExecuting(false);
    }, 1200);
  };

  const handleSaveApp = async () => {
    try {
      const payload = {
        name,
        studioMode,
        workflowNodes,
        webAppComponents
      };
      localStorage.setItem(`nextkube_studio_${name}`, JSON.stringify(payload));
      alert(`Studio configuration for "${name}" saved successfully!`);
    } catch (e) {
      alert("Error saving studio configuration: " + e.message);
    }
  };

  const activeStep = workflowNodes.find(n => n.id === selectedWorkflowStep);
  const activeComp = webAppComponents.find(c => c.id === selectedComponentId);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', width: '100vw', background: '#F8FAFC', color: '#0F172A', overflow: 'hidden' }}>
      
      {/* STUDIO TOP HEADER */}
      <header style={{
        height: '60px', background: '#FFFFFF', borderBottom: '1px solid #E2E8F0',
        padding: '0 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        flexShrink: 0, zIndex: 20
      }}>
        {/* Left: Back + Name */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <button 
            onClick={onBack} 
            title="Return to Dashboard"
            style={{ background: '#F1F5F9', border: '1px solid #CBD5E1', borderRadius: '6px', padding: '6px 12px', fontSize: '0.85rem', fontWeight: '600', color: '#475569', cursor: 'pointer' }}
          >
            &larr; Dashboard
          </button>

          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <span style={{ fontSize: '1.25rem' }}>{studioMode === 'workflow' ? '⚡️' : '🌐'}</span>
            <div>
              <input 
                type="text" 
                value={name} 
                onChange={e => setName(e.target.value)} 
                style={{ fontWeight: '700', fontSize: '1.1rem', border: '1px solid transparent', padding: '2px 6px', borderRadius: '4px', background: 'transparent', color: '#0F172A' }}
                onFocus={e => e.target.style.background = '#F8FAFC'}
                onBlur={e => e.target.style.background = 'transparent'}
              />
            </div>
            <span style={{ 
              fontSize: '0.75rem', padding: '2px 8px', borderRadius: '12px', 
              background: studioMode === 'workflow' ? '#EEF2FF' : '#ECFDF5', 
              color: studioMode === 'workflow' ? '#4F46E5' : '#059669',
              fontWeight: '600', border: studioMode === 'workflow' ? '1px solid #C7D2FE' : '1px solid #A7F3D0'
            }}>
              Leaf: {studioMode === 'workflow' ? 'Workflow Pipeline' : 'Web App'}
            </span>
          </div>
        </div>

        {/* Center: Leaf Node Mode Switcher (Workflow <-> Web App) */}
        <div style={{ display: 'flex', background: '#F1F5F9', borderRadius: '8px', padding: '3px', border: '1px solid #E2E8F0' }}>
          <button
            onClick={() => setStudioMode('workflow')}
            style={{
              padding: '6px 16px', border: 'none', borderRadius: '6px', cursor: 'pointer',
              background: studioMode === 'workflow' ? '#FFFFFF' : 'transparent',
              color: studioMode === 'workflow' ? '#4F46E5' : '#64748B',
              fontWeight: '600', fontSize: '0.85rem',
              boxShadow: studioMode === 'workflow' ? '0 1px 3px rgba(0,0,0,0.1)' : 'none',
              transition: 'all 0.15s ease'
            }}
          >
            ⚡️ Backend Workflow
          </button>
          <button
            onClick={() => setStudioMode('webapp')}
            style={{
              padding: '6px 16px', border: 'none', borderRadius: '6px', cursor: 'pointer',
              background: studioMode === 'webapp' ? '#FFFFFF' : 'transparent',
              color: studioMode === 'webapp' ? '#4F46E5' : '#64748B',
              fontWeight: '600', fontSize: '0.85rem',
              boxShadow: studioMode === 'webapp' ? '0 1px 3px rgba(0,0,0,0.1)' : 'none',
              transition: 'all 0.15s ease'
            }}
          >
            🌐 Frontend Web App
          </button>
        </div>

        {/* Right: View mode & Action Buttons */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{ display: 'flex', background: '#F8FAFC', borderRadius: '6px', border: '1px solid #E2E8F0', padding: '2px' }}>
            <button 
              onClick={() => setViewMode('visual')} 
              style={{ padding: '4px 10px', border: 'none', background: viewMode === 'visual' ? '#EEF2FF' : 'transparent', color: viewMode === 'visual' ? '#4F46E5' : '#64748B', borderRadius: '4px', fontSize: '0.8rem', fontWeight: '500', cursor: 'pointer' }}
            >
              Visual Studio
            </button>
            <button 
              onClick={() => setViewMode('preview')} 
              style={{ padding: '4px 10px', border: 'none', background: viewMode === 'preview' ? '#EEF2FF' : 'transparent', color: viewMode === 'preview' ? '#4F46E5' : '#64748B', borderRadius: '4px', fontSize: '0.8rem', fontWeight: '500', cursor: 'pointer' }}
            >
              ▶ Live Preview
            </button>
            <button 
              onClick={() => setViewMode('yaml')} 
              style={{ padding: '4px 10px', border: 'none', background: viewMode === 'yaml' ? '#EEF2FF' : 'transparent', color: viewMode === 'yaml' ? '#4F46E5' : '#64748B', borderRadius: '4px', fontSize: '0.8rem', fontWeight: '500', cursor: 'pointer' }}
            >
              YAML Spec
            </button>
          </div>

          <StudioButton primary onClick={handleSaveApp}>Save Manifest</StudioButton>
        </div>
      </header>

      {/* SUB-BAR: Kubernetes Context & Bindings */}
      <div style={{
        height: '38px', background: '#F8FAFC', borderBottom: '1px solid #E2E8F0',
        padding: '0 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        fontSize: '0.8rem', color: '#64748B'
      }}>
        <div style={{ display: 'flex', gap: '20px', alignItems: 'center' }}>
          <span><strong>K8s Namespace:</strong> <code style={{ color: '#1D4ED8', background: '#EFF6FF', padding: '1px 6px', borderRadius: '4px' }}>{item?.node?.k8s_namespace || 'default'}</code></span>
          <span><strong>Tenant:</strong> {item?.node?.k8s_tenant || 'shared'}</span>
          <span><strong>Resources:</strong> {item?.node?.resources?.limits?.cpu || '500m'} CPU / {item?.node?.resources?.limits?.memory || '512Mi'} RAM</span>
        </div>
        <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
          <span>🔗 <strong>Integrated Architecture:</strong> Web App triggers &rarr; Kubernetes Workflow Execution</span>
        </div>
      </div>

      {/* MAIN STUDIO WORKSPACE */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        
        {viewMode === 'visual' && studioMode === 'workflow' && (
          <>
            {/* WORKFLOW CATALOG SIDEBAR */}
            <div style={{ width: '260px', background: '#FFFFFF', borderRight: '1px solid #E2E8F0', padding: '20px', overflowY: 'auto' }}>
              <h3 style={{ margin: '0 0 16px 0', fontSize: '0.9rem', fontWeight: '700', color: '#1E293B', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                Connector Catalog
              </h3>
              
              <div style={{ marginBottom: '20px' }}>
                <span style={{ fontSize: '0.75rem', fontWeight: '600', color: '#94A3B8', textTransform: 'uppercase', display: 'block', marginBottom: '8px' }}>
                  1. Triggers & Ingress
                </span>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                  <button onClick={() => addWorkflowNode('trigger', 'Webhook Endpoint', '⚡️', 'POST /webhook', 'Ingress')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    ⚡️ HTTP Webhook
                  </button>
                  <button onClick={() => addWorkflowNode('trigger', 'NATS Event Consumer', '📨', 'Subject: orders.*', 'NATS Daemon')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    📨 NATS JetStream
                  </button>
                  <button onClick={() => addWorkflowNode('trigger', 'Cron Scheduler', '⏰', 'Every 5m (Cron)', 'K8s CronJob')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    ⏰ Cron Schedule
                  </button>
                </div>
              </div>

              <div style={{ marginBottom: '20px' }}>
                <span style={{ fontSize: '0.75rem', fontWeight: '600', color: '#94A3B8', textTransform: 'uppercase', display: 'block', marginBottom: '8px' }}>
                  2. Compute & Transforms
                </span>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                  <button onClick={() => addWorkflowNode('processor', 'Python Script', '🐍', 'Pandas ETL Step', 'Pod Container')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    🐍 Python Script (Pandas)
                  </button>
                  <button onClick={() => addWorkflowNode('processor', 'Data Mapping Matrix', '🔄', 'JSON to Arrow stream', 'Zero-copy Engine')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    🔄 Zero-Copy Transform
                  </button>
                  <button onClick={() => addWorkflowNode('processor', 'AI LLM Inference', '🤖', 'Gemini / Claude Step', 'AI Worker')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    🤖 AI Reasoning Step
                  </button>
                </div>
              </div>

              <div>
                <span style={{ fontSize: '0.75rem', fontWeight: '600', color: '#94A3B8', textTransform: 'uppercase', display: 'block', marginBottom: '8px' }}>
                  3. Destinations & Output
                </span>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                  <button onClick={() => addWorkflowNode('database', 'PostgreSQL / MySQL', '🗄️', 'Database Upsert', 'SQL Client')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    🗄️ Database Record
                  </button>
                  <button onClick={() => addWorkflowNode('storage', 'Amazon S3 / MinIO', '🪣', 's3://app-data-bucket', 'Object Store')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    🪣 S3 Object Bucket
                  </button>
                  <button onClick={() => addWorkflowNode('webapp_target', 'Update Web App UI', '🌐', 'Push to Web App View', 'Reactive State')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    🌐 Web App Target (Leaf)
                  </button>
                </div>
              </div>
            </div>

            {/* WORKFLOW PIPELINE CANVAS */}
            <div style={{ flex: 1, padding: '32px', overflowY: 'auto', background: '#F8FAFC' }}>
              <div style={{ maxWidth: '780px', margin: '0 auto' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
                  <div>
                    <h2 style={{ margin: 0, fontSize: '1.25rem', fontWeight: '700' }}>Workflow Graph Sequence</h2>
                    <p style={{ margin: '4px 0 0 0', color: '#64748B', fontSize: '0.85rem' }}>
                      Linear and branch execution graph executed by Kubernetes Argo engine.
                    </p>
                  </div>
                  <StudioButton secondary onClick={() => addWorkflowNode('processor', 'New Pipeline Step', '⚙️', 'Processing task', 'Container')}>
                    + Add Step
                  </StudioButton>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', position: 'relative' }}>
                  {workflowNodes.map((node, index) => {
                    const isSelected = selectedWorkflowStep === node.id;
                    return (
                      <div key={node.id} style={{ position: 'relative' }}>
                        {/* Connecting Arrow */}
                        {index > 0 && (
                          <div style={{ display: 'flex', justifyContent: 'center', margin: '-8px 0', height: '16px', position: 'relative', zIndex: 1 }}>
                            <div style={{ width: '2px', height: '100%', background: '#CBD5E1' }}></div>
                          </div>
                        )}
                        <div 
                          onClick={() => setSelectedWorkflowStep(node.id)}
                          style={{
                            background: '#FFFFFF', borderRadius: '10px', border: `2px solid ${isSelected ? '#4F46E5' : '#E2E8F0'}`,
                            padding: '16px 20px', cursor: 'pointer', boxShadow: '0 2px 4px rgba(0,0,0,0.03)',
                            display: 'flex', alignItems: 'center', justifyContent: 'space-between', transition: 'all 0.15s ease'
                          }}
                        >
                          <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
                            <span style={{ fontSize: '1.6rem' }}>{node.icon}</span>
                            <div>
                              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <span style={{ fontSize: '0.95rem', fontWeight: '700', color: '#0F172A' }}>{node.name}</span>
                                <span style={{ fontSize: '0.7rem', padding: '1px 6px', borderRadius: '4px', background: '#F1F5F9', color: '#475569' }}>{node.k8sRole}</span>
                              </div>
                              <span style={{ fontSize: '0.8rem', color: '#64748B' }}>{node.details}</span>
                            </div>
                          </div>

                          <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                            <span style={{ fontSize: '0.75rem', color: '#94A3B8' }}>Step #{index + 1}</span>
                            {workflowNodes.length > 1 && (
                              <button 
                                onClick={(e) => { e.stopPropagation(); setWorkflowNodes(workflowNodes.filter(n => n.id !== node.id)); }}
                                style={{ background: 'none', border: 'none', color: '#EF4444', cursor: 'pointer', fontSize: '1rem', padding: '2px 6px' }}
                              >
                                ×
                              </button>
                            )}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>

            {/* WORKFLOW INSPECTOR SIDEBAR */}
            <div style={{ width: '320px', background: '#FFFFFF', borderLeft: '1px solid #E2E8F0', padding: '24px', overflowY: 'auto' }}>
              <h3 style={{ margin: '0 0 16px 0', fontSize: '0.95rem', fontWeight: '700', color: '#1E293B' }}>
                Step Configuration
              </h3>
              {activeStep ? (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                  <div>
                    <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: '600', marginBottom: '6px' }}>Step Name</label>
                    <input 
                      type="text" 
                      value={activeStep.name} 
                      onChange={e => {
                        const updated = workflowNodes.map(n => n.id === activeStep.id ? { ...n, name: e.target.value } : n);
                        setWorkflowNodes(updated);
                      }}
                      style={{ width: '100%', padding: '8px 12px', border: '1px solid #CBD5E1', borderRadius: '6px', fontSize: '0.85rem', boxSizing: 'border-box' }}
                    />
                  </div>

                  <div>
                    <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: '600', marginBottom: '6px' }}>Execution Parameter / Detail</label>
                    <input 
                      type="text" 
                      value={activeStep.details} 
                      onChange={e => {
                        const updated = workflowNodes.map(n => n.id === activeStep.id ? { ...n, details: e.target.value } : n);
                        setWorkflowNodes(updated);
                      }}
                      style={{ width: '100%', padding: '8px 12px', border: '1px solid #CBD5E1', borderRadius: '6px', fontSize: '0.85rem', boxSizing: 'border-box' }}
                    />
                  </div>

                  <div>
                    <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: '600', marginBottom: '6px' }}>Kubernetes Workload Target</label>
                    <input 
                      type="text" 
                      value={activeStep.k8sRole} 
                      onChange={e => {
                        const updated = workflowNodes.map(n => n.id === activeStep.id ? { ...n, k8sRole: e.target.value } : n);
                        setWorkflowNodes(updated);
                      }}
                      style={{ width: '100%', padding: '8px 12px', border: '1px solid #CBD5E1', borderRadius: '6px', fontSize: '0.85rem', boxSizing: 'border-box' }}
                    />
                  </div>

                  <div style={{ background: '#F8FAFC', border: '1px solid #E2E8F0', padding: '12px', borderRadius: '6px' }}>
                    <span style={{ fontSize: '0.75rem', fontWeight: '600', color: '#1E293B', display: 'block', marginBottom: '4px' }}>Security & Quota</span>
                    <span style={{ fontSize: '0.75rem', color: '#64748B' }}>Runs under namespace <strong>{item?.node?.k8s_namespace || 'default'}</strong> with isolated RBAC tokens.</span>
                  </div>
                </div>
              ) : (
                <div style={{ color: '#94A3B8', fontSize: '0.85rem' }}>Select a step to configure parameters.</div>
              )}
            </div>
          </>
        )}

        {viewMode === 'visual' && studioMode === 'webapp' && (
          <>
            {/* WEB APP PALETTE SIDEBAR */}
            <div style={{ width: '260px', background: '#FFFFFF', borderRight: '1px solid #E2E8F0', padding: '20px', overflowY: 'auto' }}>
              <h3 style={{ margin: '0 0 16px 0', fontSize: '0.9rem', fontWeight: '700', color: '#1E293B', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                UI Component Palette
              </h3>
              
              <div style={{ marginBottom: '20px' }}>
                <span style={{ fontSize: '0.75rem', fontWeight: '600', color: '#94A3B8', textTransform: 'uppercase', display: 'block', marginBottom: '8px' }}>
                  Form & Inputs
                </span>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                  <button onClick={() => addWebAppComponent('input', 'Text Input')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    📝 Text Input Field
                  </button>
                  <button onClick={() => addWebAppComponent('select', 'Dropdown Selector')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    🔽 Dropdown / Select
                  </button>
                  <button onClick={() => addWebAppComponent('file', 'File Upload Box')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    📁 Document Upload
                  </button>
                  <button onClick={() => addWebAppComponent('button', 'Submit Button')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    🔘 Action Button (Calls Workflow)
                  </button>
                </div>
              </div>

              <div>
                <span style={{ fontSize: '0.75rem', fontWeight: '600', color: '#94A3B8', textTransform: 'uppercase', display: 'block', marginBottom: '8px' }}>
                  Layout & Displays
                </span>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                  <button onClick={() => addWebAppComponent('header', 'Section Header')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    🔤 Section Title
                  </button>
                  <button onClick={() => addWebAppComponent('table', 'Data Grid Table')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    📊 Paginated Data Grid
                  </button>
                  <button onClick={() => addWebAppComponent('kpi', 'KPI Metrics Card')} style={{ textAlign: 'left', padding: '8px 10px', background: '#F8FAFC', border: '1px solid #E2E8F0', borderRadius: '6px', fontSize: '0.8rem', cursor: 'pointer' }}>
                    📈 KPI Metric Card
                  </button>
                </div>
              </div>
            </div>

            {/* WEB APP WYSIWYG CANVAS */}
            <div style={{ flex: 1, padding: '32px', overflowY: 'auto', background: '#F1F5F9' }}>
              <div style={{ maxWidth: '680px', margin: '0 auto', background: '#FFFFFF', borderRadius: '12px', border: '1px solid #CBD5E1', padding: '32px', boxShadow: '0 4px 6px -1px rgba(0,0,0,0.05)' }}>
                <div style={{ borderBottom: '1px solid #E2E8F0', paddingBottom: '16px', marginBottom: '24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div>
                    <span style={{ fontSize: '0.7rem', fontWeight: '700', color: '#4F46E5', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                      Web App Live Canvas Preview
                    </span>
                    <h2 style={{ margin: '4px 0 0 0', fontSize: '1.4rem', fontWeight: '700' }}>{name}</h2>
                  </div>
                  <span style={{ fontSize: '0.75rem', background: '#DCFCE7', color: '#166534', padding: '3px 8px', borderRadius: '12px', fontWeight: '600' }}>
                    Linked to Workflow
                  </span>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
                  {webAppComponents.map(comp => {
                    const isSelected = selectedComponentId === comp.id;
                    return (
                      <div 
                        key={comp.id} 
                        onClick={() => setSelectedComponentId(comp.id)}
                        style={{
                          padding: '16px', borderRadius: '8px', cursor: 'pointer',
                          border: `2px dashed ${isSelected ? '#4F46E5' : '#E2E8F0'}`,
                          background: isSelected ? '#F5F7FF' : '#FAFAFA',
                          position: 'relative', transition: 'all 0.15s ease'
                        }}
                      >
                        {comp.type === 'header' && (
                          <div>
                            <h3 style={{ margin: '0 0 4px 0', fontSize: '1.2rem', color: '#0F172A' }}>{comp.label}</h3>
                            <p style={{ margin: 0, fontSize: '0.85rem', color: '#64748B' }}>{comp.subtitle || 'Configure input parameters below'}</p>
                          </div>
                        )}

                        {comp.type === 'input' && (
                          <div>
                            <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px' }}>{comp.label}</label>
                            <input 
                              type="text" 
                              placeholder={comp.placeholder} 
                              disabled 
                              style={{ width: '100%', padding: '10px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', background: '#FFF', boxSizing: 'border-box' }}
                            />
                          </div>
                        )}

                        {comp.type === 'select' && (
                          <div>
                            <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px' }}>{comp.label}</label>
                            <select disabled style={{ width: '100%', padding: '10px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', background: '#FFF', boxSizing: 'border-box' }}>
                              {(comp.options || ['Option A', 'Option B']).map(opt => <option key={opt}>{opt}</option>)}
                            </select>
                          </div>
                        )}

                        {comp.type === 'file' && (
                          <div>
                            <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px' }}>{comp.label}</label>
                            <div style={{ border: '2px dashed #CBD5E1', padding: '20px', borderRadius: '6px', textAlign: 'center', background: '#FFF', color: '#64748B', fontSize: '0.85rem' }}>
                              Drag and drop files here to stream directly to Kubernetes PVC
                            </div>
                          </div>
                        )}

                        {comp.type === 'button' && (
                          <div>
                            <button style={{ width: '100%', padding: '12px 20px', background: '#4F46E5', color: '#FFF', border: 'none', borderRadius: '6px', fontWeight: '600', cursor: 'pointer' }}>
                              {comp.label} &rarr; Triggers Workflow: "{comp.workflowTarget || itemName}"
                            </button>
                          </div>
                        )}

                        {comp.type === 'table' && (
                          <div>
                            <div style={{ fontSize: '0.85rem', fontWeight: '600', marginBottom: '8px' }}>{comp.label}</div>
                            <div style={{ background: '#FFF', border: '1px solid #E2E8F0', borderRadius: '6px', padding: '12px', fontSize: '0.8rem', color: '#64748B' }}>
                              [Live Data Table with pagination streaming from ClickHouse/PostgreSQL]
                            </div>
                          </div>
                        )}

                        {comp.type === 'kpi' && (
                          <div style={{ background: '#FFF', border: '1px solid #E2E8F0', borderRadius: '6px', padding: '16px', display: 'flex', justifyContent: 'space-between' }}>
                            <div>
                              <div style={{ fontSize: '0.8rem', color: '#64748B' }}>{comp.label}</div>
                              <div style={{ fontSize: '1.5rem', fontWeight: '700', color: '#0F172A' }}>99.98%</div>
                            </div>
                            <span style={{ color: '#10B981', fontWeight: '600', fontSize: '0.8rem' }}>+4.2%</span>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>

            {/* WEB APP COMPONENT INSPECTOR */}
            <div style={{ width: '320px', background: '#FFFFFF', borderLeft: '1px solid #E2E8F0', padding: '24px', overflowY: 'auto' }}>
              <h3 style={{ margin: '0 0 16px 0', fontSize: '0.95rem', fontWeight: '700', color: '#1E293B' }}>
                Component Properties
              </h3>
              {activeComp ? (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                  <div>
                    <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: '600', marginBottom: '6px' }}>Label Text</label>
                    <input 
                      type="text" 
                      value={activeComp.label} 
                      onChange={e => {
                        const updated = webAppComponents.map(c => c.id === activeComp.id ? { ...c, label: e.target.value } : c);
                        setWebAppComponents(updated);
                      }}
                      style={{ width: '100%', padding: '8px 12px', border: '1px solid #CBD5E1', borderRadius: '6px', fontSize: '0.85rem', boxSizing: 'border-box' }}
                    />
                  </div>

                  <div>
                    <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: '600', marginBottom: '6px' }}>Payload Field Name (JSON Key)</label>
                    <input 
                      type="text" 
                      value={activeComp.fieldName || ''} 
                      onChange={e => {
                        const updated = webAppComponents.map(c => c.id === activeComp.id ? { ...c, fieldName: e.target.value } : c);
                        setWebAppComponents(updated);
                      }}
                      style={{ width: '100%', padding: '8px 12px', border: '1px solid #CBD5E1', borderRadius: '6px', fontSize: '0.85rem', boxSizing: 'border-box' }}
                    />
                  </div>

                  {activeComp.type === 'button' && (
                    <div style={{ background: '#EFF6FF', border: '1px solid #BFDBFE', padding: '14px', borderRadius: '8px' }}>
                      <label style={{ display: 'block', fontSize: '0.8rem', fontWeight: '700', color: '#1E40AF', marginBottom: '6px' }}>
                        🔗 Bound Workflow Target
                      </label>
                      <input 
                        type="text" 
                        value={activeComp.workflowTarget || itemName} 
                        onChange={e => {
                          const updated = webAppComponents.map(c => c.id === activeComp.id ? { ...c, workflowTarget: e.target.value } : c);
                          setWebAppComponents(updated);
                        }}
                        style={{ width: '100%', padding: '8px 10px', border: '1px solid #93C5FD', borderRadius: '6px', fontSize: '0.85rem', boxSizing: 'border-box' }}
                      />
                      <span style={{ fontSize: '0.72rem', color: '#1E40AF', marginTop: '4px', display: 'block' }}>
                        Clicking this button packages form inputs and dispatches the execution pipeline.
                      </span>
                    </div>
                  )}

                  <button 
                    onClick={() => setWebAppComponents(webAppComponents.filter(c => c.id !== activeComp.id))}
                    style={{ marginTop: '16px', padding: '8px', background: '#FEF2F2', border: '1px solid #FECACA', color: '#EF4444', borderRadius: '6px', cursor: 'pointer', fontSize: '0.85rem', fontWeight: '600' }}
                  >
                    Delete Component
                  </button>
                </div>
              ) : (
                <div style={{ color: '#94A3B8', fontSize: '0.85rem' }}>Select a component to customize.</div>
              )}
            </div>
          </>
        )}

        {/* LIVE PREVIEW TAB */}
        {viewMode === 'preview' && (
          <div style={{ flex: 1, display: 'flex', background: '#F8FAFC' }}>
            {/* Interactive Form Panel */}
            <div style={{ flex: 1, padding: '32px', overflowY: 'auto', borderRight: '1px solid #E2E8F0' }}>
              <div style={{ maxWidth: '600px', margin: '0 auto', background: '#FFFFFF', padding: '32px', borderRadius: '12px', border: '1px solid #E2E8F0', boxShadow: '0 4px 6px -1px rgba(0,0,0,0.05)' }}>
                <div style={{ marginBottom: '24px' }}>
                  <span style={{ fontSize: '0.75rem', fontWeight: '700', color: '#4F46E5', textTransform: 'uppercase' }}>End-User Interactive View</span>
                  <h2 style={{ margin: '4px 0', fontSize: '1.5rem', fontWeight: '700' }}>{name}</h2>
                  <p style={{ margin: 0, color: '#64748B', fontSize: '0.85rem' }}>{description}</p>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                  <div>
                    <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px' }}>Customer Name</label>
                    <input 
                      type="text" 
                      value={formValues.customer} 
                      onChange={e => setFormValues({ ...formValues, customer: e.target.value })}
                      style={{ width: '100%', padding: '10px 12px', border: '1px solid #CBD5E1', borderRadius: '6px', fontSize: '0.9rem', boxSizing: 'border-box' }}
                    />
                  </div>

                  <div>
                    <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: '600', marginBottom: '6px' }}>Priority Level</label>
                    <select 
                      value={formValues.priority} 
                      onChange={e => setFormValues({ ...formValues, priority: e.target.value })}
                      style={{ width: '100%', padding: '10px 12px', border: '1px solid #CBD5E1', borderRadius: '6px', fontSize: '0.9rem', boxSizing: 'border-box', background: '#FFF' }}
                    >
                      <option value="Standard">Standard</option>
                      <option value="Expedited">Expedited</option>
                      <option value="Critical">Critical</option>
                    </select>
                  </div>

                  <div style={{ marginTop: '12px' }}>
                    <StudioButton 
                      primary 
                      disabled={isExecuting}
                      onClick={handleRunExecution}
                      style={{ width: '100%', padding: '12px', fontSize: '0.95rem' }}
                    >
                      {isExecuting ? '⏳ Executing Kubernetes Pipeline...' : '▶ Submit to Workflow Engine'}
                    </StudioButton>
                  </div>
                </div>
              </div>
            </div>

            {/* Live Terminal & Stream Logs */}
            <div style={{ width: '480px', background: '#0F172A', color: '#38BDF8', padding: '24px', display: 'flex', flexDirection: 'column', fontFamily: 'monospace', overflowY: 'auto' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid #1E293B', paddingBottom: '12px', marginBottom: '16px' }}>
                <span style={{ color: '#F1F5F9', fontWeight: '700', fontSize: '0.9rem' }}>Live Execution Stream</span>
                <span style={{ fontSize: '0.75rem', color: '#10B981', display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <span style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#10B981' }}></span> Live Pod Link
                </span>
              </div>
              <div style={{ flex: 1, overflowY: 'auto', fontSize: '0.8rem', lineHeight: '1.6' }}>
                {executionLogs.length === 0 ? (
                  <div style={{ color: '#64748B' }}>Submit the form on the left to trace real-time workflow pipeline execution...</div>
                ) : (
                  executionLogs.map((log, i) => (
                    <div key={i} style={{ color: log.includes('✅') ? '#4ADE80' : log.includes('Step') ? '#E2E8F0' : '#94A3B8' }}>
                      {log}
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        )}

        {/* YAML TAB */}
        {viewMode === 'yaml' && (
          <div style={{ flex: 1, padding: '24px', background: '#0F172A', display: 'flex', flexDirection: 'column' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
              <span style={{ color: '#94A3B8', fontSize: '0.85rem', fontFamily: 'monospace' }}>Generated Declarative Kubernetes Manifest</span>
              <StudioButton secondary onClick={() => alert("Manifest copied to clipboard!")} style={{ fontSize: '0.8rem', padding: '4px 10px' }}>
                Copy YAML
              </StudioButton>
            </div>
            <textarea 
              value={yamlContent} 
              onChange={e => setYamlContent(e.target.value)}
              style={{
                flex: 1, width: '100%', background: '#020617', color: '#38BDF8',
                border: '1px solid #1E293B', borderRadius: '8px', padding: '20px',
                fontFamily: 'monospace', fontSize: '0.9rem', lineHeight: '1.5', resize: 'none',
                boxSizing: 'border-box'
              }}
            />
          </div>
        )}

      </div>
    </div>
  );
}
