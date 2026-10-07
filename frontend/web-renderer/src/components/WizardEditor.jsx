import React, { useState, useEffect } from 'react';

const Card = ({ children, style }) => (
  <div style={{ background: '#FFFFFF', borderRadius: '12px', border: '1px solid #E2E8F0', padding: '32px', boxShadow: '0 4px 6px -1px rgba(0,0,0,0.05), 0 2px 4px -1px rgba(0,0,0,0.03)', ...style }}>
    {children}
  </div>
);

const Button = ({ children, primary, secondary, onClick, style, disabled=false }) => {
  let bg = '#FFFFFF';
  let color = '#334155';
  let border = '1px solid #CBD5E1';
  
  if (primary) {
    bg = disabled ? '#94A3B8' : '#4F46E5';
    color = '#FFFFFF';
    border = 'none';
  } else if (secondary) {
    bg = '#F1F5F9';
    border = 'none';
  }

  return (
    <button onClick={onClick} disabled={disabled} style={{
      background: bg,
      color: color,
      border: border,
      padding: '10px 20px',
      borderRadius: '8px',
      fontWeight: '600',
      cursor: disabled ? 'not-allowed' : 'pointer',
      fontSize: '0.875rem',
      transition: 'all 0.2s ease',
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      ...style
    }}>
      {children}
    </button>
  );
};

const Input = ({ label, value, onChange, placeholder, type="text" }) => (
  <div style={{ marginBottom: '20px' }}>
    <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: '500', color: '#334155', marginBottom: '8px' }}>
      {label}
    </label>
    <input 
      type={type} 
      value={value} 
      onChange={e => onChange(e.target.value)} 
      placeholder={placeholder}
      style={{
        width: '100%', padding: '12px 16px', borderRadius: '8px', 
        border: '1px solid #CBD5E1', fontSize: '0.875rem', 
        boxSizing: 'border-box', outline: 'none', transition: 'border-color 0.2s'
      }}
      onFocus={e => e.target.style.borderColor = '#4F46E5'}
      onBlur={e => e.target.style.borderColor = '#CBD5E1'}
    />
  </div>
);

const SelectableCard = ({ title, description, icon, selected, onClick }) => (
  <div 
    onClick={onClick}
    style={{
      border: `2px solid ${selected ? '#4F46E5' : '#E2E8F0'}`,
      borderRadius: '12px',
      padding: '24px',
      cursor: 'pointer',
      background: selected ? '#EEF2FF' : '#FFFFFF',
      transition: 'all 0.2s ease',
      display: 'flex',
      alignItems: 'flex-start',
      gap: '16px'
    }}
  >
    <div style={{ fontSize: '24px', background: selected ? '#4F46E5' : '#F1F5F9', color: selected ? '#FFF' : '#64748B', width: '48px', height: '48px', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      {icon}
    </div>
    <div>
      <h4 style={{ margin: '0 0 4px 0', fontSize: '1rem', color: '#0F172A' }}>{title}</h4>
      <p style={{ margin: 0, fontSize: '0.875rem', color: '#64748B', lineHeight: '1.4' }}>{description}</p>
    </div>
  </div>
);

const STEPS = [
  { id: 1, label: 'Metadata' },
  { id: 2, label: 'Trigger' },
  { id: 3, label: 'Processing' },
  { id: 4, label: 'Destination' },
  { id: 5, label: 'Review & Deploy' }
];

export function WizardEditor({ appName, onBack }) {
  const [step, setStep] = useState(1);
  const [config, setConfig] = useState({
    name: appName || '',
    description: '',
    trigger: null,
    triggerConfig: {},
    processor: null,
    destination: null,
    destinationConfig: {}
  });

  const [saving, setSaving] = useState(false);

  const updateConfig = (key, value) => {
    setConfig(prev => ({ ...prev, [key]: value }));
  };

  const handleDeploy = async () => {
    setSaving(true);
    // Construct KttmApp YAML equivalent in JSON
    const payload = {
      apiVersion: "flowengine.io/v1alpha1",
      kind: "KttmApp",
      metadata: { name: config.name },
      spec: {
        nodes: []
      }
    };
    
    if (config.trigger) {
      payload.spec.nodes.push({ id: 'source', type: `connector/${config.trigger.toLowerCase()}`, params: config.triggerConfig });
    }
    if (config.destination) {
      payload.spec.nodes.push({ id: 'sink', type: `connector/${config.destination.toLowerCase()}`, params: config.destinationConfig, dependsOn: ['source'] });
    }

    try {
      await fetch(`/api/apps/${encodeURIComponent(config.name)}/sync`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      onBack();
    } catch (err) {
      console.error(err);
      setSaving(false);
    }
  };

  const renderStepContent = () => {
    switch (step) {
      case 1:
        return (
          <div>
            <h2 style={{ margin: '0 0 24px 0', fontSize: '1.5rem' }}>Basic Information</h2>
            <Input label="Workflow Name" value={config.name} onChange={v => updateConfig('name', v)} placeholder="e.g. customer-onboarding" />
            <Input label="Description" value={config.description} onChange={v => updateConfig('description', v)} placeholder="Briefly describe what this workflow does" />
          </div>
        );
      case 2:
        return (
          <div>
            <h2 style={{ margin: '0 0 8px 0', fontSize: '1.5rem' }}>Select Trigger</h2>
            <p style={{ color: '#64748B', marginBottom: '24px' }}>How should this workflow start?</p>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <SelectableCard title="Webhook" description="Trigger via HTTP POST requests" icon="🌐" selected={config.trigger === 'Webhook'} onClick={() => updateConfig('trigger', 'Webhook')} />
              <SelectableCard title="Kafka" description="Consume messages from a Kafka topic" icon="📨" selected={config.trigger === 'Kafka'} onClick={() => updateConfig('trigger', 'Kafka')} />
              <SelectableCard title="S3 Drop" description="Trigger when files are uploaded" icon="📁" selected={config.trigger === 'S3'} onClick={() => updateConfig('trigger', 'S3')} />
              <SelectableCard title="Cron Schedule" description="Run on a timed schedule" icon="⏱️" selected={config.trigger === 'Cron'} onClick={() => updateConfig('trigger', 'Cron')} />
            </div>
            
            {config.trigger === 'Webhook' && (
              <div style={{ marginTop: '24px', padding: '24px', background: '#F8FAFC', borderRadius: '8px' }}>
                <Input label="Webhook Path" value={config.triggerConfig.path || ''} onChange={v => updateConfig('triggerConfig', {...config.triggerConfig, path: v})} placeholder="/my-webhook" />
              </div>
            )}
          </div>
        );
      case 3:
        return (
          <div>
            <h2 style={{ margin: '0 0 8px 0', fontSize: '1.5rem' }}>Processing Steps</h2>
            <p style={{ color: '#64748B', marginBottom: '24px' }}>Add transformations to your data in-flight.</p>
            <div style={{ padding: '40px', border: '2px dashed #CBD5E1', borderRadius: '12px', textAlign: 'center', color: '#64748B' }}>
              <div style={{ fontSize: '32px', marginBottom: '16px' }}>⚙️</div>
              <p>No processing steps defined.</p>
              <Button secondary>+ Add Step</Button>
            </div>
          </div>
        );
      case 4:
        return (
          <div>
            <h2 style={{ margin: '0 0 8px 0', fontSize: '1.5rem' }}>Destination</h2>
            <p style={{ color: '#64748B', marginBottom: '24px' }}>Where should the results go?</p>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <SelectableCard title="PostgreSQL" description="Insert rows into a database" icon="🐘" selected={config.destination === 'Postgres'} onClick={() => updateConfig('destination', 'Postgres')} />
              <SelectableCard title="Kafka" description="Publish to another topic" icon="📨" selected={config.destination === 'Kafka'} onClick={() => updateConfig('destination', 'Kafka')} />
              <SelectableCard title="S3" description="Save payload as object" icon="📁" selected={config.destination === 'S3'} onClick={() => updateConfig('destination', 'S3')} />
            </div>
          </div>
        );
      case 5:
        return (
          <div>
            <h2 style={{ margin: '0 0 8px 0', fontSize: '1.5rem' }}>Review & Deploy</h2>
            <p style={{ color: '#64748B', marginBottom: '24px' }}>Verify your workflow configuration before deploying to the cluster.</p>
            
            <div style={{ background: '#0F172A', color: '#F8FAFC', padding: '24px', borderRadius: '8px', fontFamily: 'monospace', whiteSpace: 'pre', overflowX: 'auto', fontSize: '0.875rem' }}>
{`apiVersion: flowengine.io/v1alpha1
kind: KttmApp
metadata:
  name: ${config.name || 'untitled'}
spec:
  nodes:
${config.trigger ? `  - id: source\n    type: connector/${config.trigger.toLowerCase()}\n    params:\n      path: ${config.triggerConfig.path || ''}` : ''}
${config.destination ? `  - id: sink\n    type: connector/${config.destination.toLowerCase()}\n    dependsOn: [source]` : ''}`}
            </div>
          </div>
        );
      default:
        return null;
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', background: '#F8FAFC' }}>
      {/* Header */}
      <header style={{ background: '#FFFFFF', borderBottom: '1px solid #E2E8F0', padding: '16px 32px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <Button secondary onClick={onBack}>← Back</Button>
          <h1 style={{ margin: 0, fontSize: '1.25rem', fontWeight: '600' }}>
            {config.name ? `Editing: ${config.name}` : 'Create New Workflow'}
          </h1>
        </div>
      </header>

      {/* Main Wizard Area */}
      <main style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        
        {/* Progress Sidebar */}
        <aside style={{ width: '260px', padding: '32px', borderRight: '1px solid #E2E8F0', background: '#FFFFFF' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
            {STEPS.map((s, index) => {
              const isPast = s.id < step;
              const isCurrent = s.id === step;
              return (
                <div key={s.id} style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                  <div style={{ 
                    width: '32px', height: '32px', borderRadius: '16px', 
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    background: isPast || isCurrent ? '#4F46E5' : '#F1F5F9',
                    color: isPast || isCurrent ? '#FFFFFF' : '#94A3B8',
                    fontWeight: '600', fontSize: '0.875rem'
                  }}>
                    {isPast ? '✓' : s.id}
                  </div>
                  <span style={{ 
                    fontWeight: isCurrent ? '600' : '500', 
                    color: isCurrent ? '#0F172A' : (isPast ? '#475569' : '#94A3B8')
                  }}>
                    {s.label}
                  </span>
                </div>
              );
            })}
          </div>
        </aside>

        {/* Content Area */}
        <section style={{ flex: 1, padding: '48px', overflowY: 'auto', display: 'flex', justifyContent: 'center' }}>
          <div style={{ width: '100%', maxWidth: '720px' }}>
            <Card style={{ minHeight: '400px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
              
              <div style={{ flex: 1 }}>
                {renderStepContent()}
              </div>

              {/* Navigation Footer */}
              <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '48px', paddingTop: '24px', borderTop: '1px solid #E2E8F0' }}>
                <Button secondary disabled={step === 1} onClick={() => setStep(step - 1)}>
                  Back
                </Button>
                
                {step < STEPS.length ? (
                  <Button primary onClick={() => setStep(step + 1)} disabled={step === 1 && !config.name}>
                    Next Step →
                  </Button>
                ) : (
                  <Button primary onClick={handleDeploy} disabled={saving}>
                    {saving ? 'Deploying...' : 'Deploy Workflow 🚀'}
                  </Button>
                )}
              </div>
            </Card>
          </div>
        </section>

      </main>
    </div>
  );
}
