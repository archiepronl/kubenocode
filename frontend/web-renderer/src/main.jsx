/**
 * KTTM — Polymorphic Single-Page Application
 *
 * Implements KTTM-REQ-001 through REQ-006:
 *   One SPA bundle, permission-driven rendering — three distinct experiences:
 *   ┌─────────────────┬──────────────────────────────────────────────────┐
 *   │ Plane           │ Permissions required                             │
 *   ├─────────────────┼──────────────────────────────────────────────────┤
 *   │ Developer SDK   │ app:create OR app:debug                          │
 *   │ Admin Console   │ rbac:manage (supersedes Developer)               │
 *   │ End-User Portal │ app:execute only (all others masked out)         │
 *   └─────────────────┴──────────────────────────────────────────────────┘
 *
 * KTTM-REQ-009: Searchable connector catalog (60+ types)
 * KTTM-REQ-011: Schema-driven node introspection sidebar
 * KTTM-REQ-014: Interactive breakpoint debug console
 * KTTM-REQ-015: Drag-and-drop visual-to-backend form binding
 * KTTM-REQ-028: OpenCost financial dashboard
 */
import { StrictMode, useEffect, useState, useCallback, useMemo } from 'react'
import { createRoot } from 'react-dom/client'
import './styles.css'
import { useKttmStore } from './store.js'

// ── Constants ──────────────────────────────────────────────────────────────

const NODE_COLORS = {
  'connector': '#3b82f6', 'trigger': '#8b5cf6', 'script': '#f59e0b',
  'logic': '#06b6d4', 'sink': '#10b981', 'algorithm': '#ec4899',
}
const NODE_EMOJIS = {
  'connector/s3': '☁', 'connector/postgres': '🐘', 'connector/kafka': '🌊',
  'connector/nats': '⚡', 'trigger/webhook': '🔔', 'trigger/cron': '🕐',
  'trigger/file-watch': '📁', 'trigger/mqtt': '📡', 'script/python': '🐍',
  'script/javascript': '🟨', 'script/bash': '🖥', 'script/r': '📊',
  'logic/fanout': '⑃', 'logic/barrier': '⑄', 'logic/branch': '↔',
  'logic/aggregate': '📦', 'sink/postgres': '🐘', 'sink/report-pdf': '📄',
  'sink/slack': '💬', 'sink/email': '📧', 'sink/mongodb': '🍃',
}
const STATUS_COLORS = { succeeded: 'var(--status-ok)', failed: 'var(--status-err)', running: 'var(--status-warn)', idle: 'var(--text-faint)' }

function nodeColor(type) { return NODE_COLORS[type?.split('/')[0]] ?? '#6b7280' }
function nodeEmoji(type) { return NODE_EMOJIS[type] ?? '▪' }

// ─────────────────────────────────────────────
//  Shared primitives
// ─────────────────────────────────────────────

function StatusDot({ phase, size = 8 }) {
  return <span style={{ display: 'inline-block', width: size, height: size, borderRadius: '50%', background: STATUS_COLORS[phase] ?? 'var(--text-faint)', flexShrink: 0 }} aria-label={phase} />
}

function Badge({ children, color = 'var(--accent)' }) {
  return <span style={{ fontSize: 9, fontFamily: 'var(--font-mono)', letterSpacing: '0.08em', textTransform: 'uppercase', padding: '2px 6px', borderRadius: 4, background: color + '22', color, border: `1px solid ${color}44` }}>{children}</span>
}

function Pill({ children, color }) {
  return <span style={{ padding: '1px 8px', borderRadius: 999, fontSize: 10, background: `${color}22`, color, border: `1px solid ${color}44` }}>{children}</span>
}

// ─────────────────────────────────────────────
//  Topbar
// ─────────────────────────────────────────────

const DEV_TABS  = [
  { id: 'form',       label: 'Run' },
  { id: 'canvas',     label: 'Canvas' },
  { id: 'connectors', label: 'Connectors' },
  { id: 'debug',      label: 'Debug' },
  { id: 'history',    label: 'History' },
  { id: 'schema',     label: 'Schema' },
]
const ADMIN_TABS = [
  { id: 'admin-users',     label: 'Users & Roles' },
  { id: 'admin-lifecycle', label: 'Lifecycle' },
  { id: 'admin-bundles',   label: 'Bundles' },
  { id: 'admin-audit',     label: 'Audit Log' },
  { id: 'admin-cost',      label: 'Cost' },
]
const USER_TABS  = [
  { id: 'form',    label: 'Submit' },
  { id: 'history', label: 'My Runs' },
]

function Topbar() {
  const { plane, currentUser, natsStatus, activeTab, setActiveTab, setUser, users } = useKttmStore()
  const tabs = plane === 'admin' ? ADMIN_TABS : plane === 'developer' ? DEV_TABS : USER_TABS
  const statusLabel = { disconnected: 'No cluster', connecting: 'Connecting…', connected: 'Connected', error: 'Cluster error' }

  return (
    <header className="topbar" role="banner">
      <div className="topbar-brand">
        <div className="brand-mark" aria-hidden="true">K</div>
        <span>KTTM</span>
        <Badge color={plane === 'admin' ? 'var(--status-warn)' : plane === 'developer' ? 'var(--accent)' : 'var(--status-ok)'}>{plane}</Badge>
      </div>

      <nav className="topbar-nav" role="navigation" aria-label="Main navigation">
        {tabs.map(tab => (
          <button key={tab.id} id={`nav-tab-${tab.id}`} className={`nav-tab ${activeTab === tab.id ? 'active' : ''}`}
            onClick={() => setActiveTab(tab.id)} aria-current={activeTab === tab.id ? 'page' : undefined}>
            {tab.label}
          </button>
        ))}
      </nav>

      <div className="topbar-right">
        {/* Quick user switcher for demo */}
        <select
          value={currentUser.name}
          onChange={e => setUser(users.find(u => u.name === e.target.value))}
          style={{ fontSize: 11, background: 'var(--surface-2)', color: 'var(--text-main)', border: '1px solid var(--border)', borderRadius: 6, padding: '3px 6px', cursor: 'pointer' }}
          aria-label="Switch demo user"
        >
          {users.map(u => <option key={u.name} value={u.name}>{u.name} ({u.role})</option>)}
        </select>
        <div className="connection-pill" aria-label={`NATS ${statusLabel[natsStatus]}`}>
          <div className={`connection-dot ${natsStatus}`} />
          {statusLabel[natsStatus]}
        </div>
      </div>
    </header>
  )
}

// ─────────────────────────────────────────────
//  DAG Sidebar (shared across planes)
// ─────────────────────────────────────────────

function DagSidebar() {
  const { schema, activeNodeId, setActiveNode, nodeStatuses, metrics } = useKttmStore()
  const dag = schema?.workflowDag ?? []

  return (
    <aside className="sidebar" aria-label="Workflow DAG">
      <div className="sidebar-header">
        <div className="sidebar-label">Workflow</div>
        <div className="sidebar-title">{schema?.title ?? 'No workflow loaded'}</div>
      </div>
      <div className="dag-container" role="list">
        {dag.map((node, i) => {
          const phase = nodeStatuses[node.id] ?? 'idle'
          const m = metrics[node.id]
          const warn = m && (m.mem / m.memLimit > 0.7 || m.cpu > 70)
          return (
            <div key={node.id} role="listitem">
              <div
                className={`dag-node ${activeNodeId === node.id ? 'active' : ''}`}
                style={{ '--node-accent': nodeColor(node.type) }}
                onClick={() => setActiveNode(node.id)}
                tabIndex={0} role="button" aria-pressed={activeNodeId === node.id}
                onKeyDown={e => e.key === 'Enter' && setActiveNode(node.id)}
              >
                <div className="dag-node-icon" aria-hidden="true" style={{ background: nodeColor(node.type) + '22', color: nodeColor(node.type) }}>
                  {nodeEmoji(node.type)}
                </div>
                <div className="dag-node-info">
                  <div className="dag-node-id truncate">{node.label ?? node.id}</div>
                  <div className="dag-node-type">{node.type}</div>
                </div>
                <StatusDot phase={phase} />
                {warn && <span title="Resource warning" style={{ fontSize: 10 }}>⚠</span>}
              </div>
              {i < dag.length - 1 && <div className="dag-edge" aria-hidden="true"><div className="dag-edge-label">▼</div></div>}
            </div>
          )
        })}
      </div>
    </aside>
  )
}

// ─────────────────────────────────────────────
//  PermissionGate
// ─────────────────────────────────────────────

function PermissionGate({ required, children, fallback = null }) {
  const { hasPermission } = useKttmStore()
  if (!hasPermission(required)) return fallback
  return children
}

// ─────────────────────────────────────────────
//  Connector Catalog (KTTM-REQ-009)
// ─────────────────────────────────────────────

const CATEGORIES = ['All', 'Trigger', 'Connector', 'Script', 'Logic', 'Sink', 'Database', 'Messaging']

function ConnectorCatalogView() {
  const { connectorSearch, activeCategory, setConnectorSearch, setActiveCategory, filteredConnectors, activeNodeId } = useKttmStore()
  const connectors = filteredConnectors()

  return (
    <div className="form-view">
      <div className="form-header">
        <div className="form-eyebrow">KTTM-REQ-009 · 20+ connector types</div>
        <h1 className="form-title">Connector Catalog</h1>
        <p className="form-description">Search and drag connectors onto the canvas. Each connector exposes a JSON Schema for auto-generated configuration forms.</p>
      </div>

      <div style={{ display: 'flex', gap: 10, marginBottom: 16, flexWrap: 'wrap' }}>
        <input
          id="connector-search"
          type="text"
          className="field-control"
          placeholder="Search connectors…"
          value={connectorSearch}
          onChange={e => setConnectorSearch(e.target.value)}
          style={{ flex: 1, minWidth: 200 }}
          aria-label="Search connector catalog"
        />
      </div>

      <div style={{ display: 'flex', gap: 6, marginBottom: 20, flexWrap: 'wrap' }}>
        {CATEGORIES.map(cat => (
          <button key={cat} onClick={() => setActiveCategory(cat)}
            style={{ padding: '4px 12px', borderRadius: 20, fontSize: 11, cursor: 'pointer', border: '1px solid', borderColor: activeCategory === cat ? 'var(--accent)' : 'var(--border)', background: activeCategory === cat ? 'var(--accent)22' : 'transparent', color: activeCategory === cat ? 'var(--accent)' : 'var(--text-muted)', fontWeight: activeCategory === cat ? 600 : 400, transition: 'all 0.15s' }}>
            {cat}
          </button>
        ))}
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 12 }} role="list" aria-label="Connector catalog">
        {connectors.map(c => (
          <div key={c.type} className="advice-card info" role="listitem" style={{ cursor: 'grab', userSelect: 'none' }}
            draggable title={`Drag to add ${c.label} node to canvas`}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 6 }}>
              <span style={{ fontSize: 20, width: 32, textAlign: 'center' }}>{c.icon}</span>
              <div>
                <div style={{ fontWeight: 600, fontSize: 13, color: 'var(--text-main)' }}>{c.label}</div>
                <div style={{ fontSize: 10, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)' }}>{c.type}</div>
              </div>
              <div style={{ marginLeft: 'auto' }}><Pill color={nodeColor(c.type)}>{c.category}</Pill></div>
            </div>
            <div style={{ fontSize: 12, color: 'var(--text-muted)', lineHeight: 1.5 }}>{c.description}</div>
          </div>
        ))}
        {connectors.length === 0 && (
          <div style={{ gridColumn: '1/-1', color: 'var(--text-faint)', textAlign: 'center', padding: 40 }}>
            No connectors found for "{connectorSearch}"
          </div>
        )}
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────
//  DAG Canvas (KTTM-REQ-023) — Simplified visual
// ─────────────────────────────────────────────

function CanvasView() {
  const { schema, nodeStatuses, activeNodeId, setActiveNode } = useKttmStore()
  const dag = schema?.workflowDag ?? []

  return (
    <div className="form-view">
      <div className="form-header">
        <div className="form-eyebrow">KTTM-REQ-023 · Visual DAG Editor</div>
        <h1 className="form-title">Workflow Canvas</h1>
        <p className="form-description">Visual node graph. In production, this uses XYFlow for drag-and-drop editing with full parallel fan-out and barrier sync support.</p>
      </div>

      <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: 12, padding: 32, minHeight: 480, position: 'relative', overflow: 'hidden' }}>
        {/* Grid background */}
        <div style={{ position: 'absolute', inset: 0, backgroundImage: 'radial-gradient(circle, var(--border) 1px, transparent 1px)', backgroundSize: '28px 28px', opacity: 0.4 }} aria-hidden="true" />

        <div style={{ position: 'relative', display: 'flex', gap: 16, flexWrap: 'wrap', justifyContent: 'center', alignItems: 'flex-start' }}>
          {dag.map((node, i) => {
            const phase = nodeStatuses[node.id] ?? 'idle'
            const active = activeNodeId === node.id
            return (
              <div key={node.id} onClick={() => setActiveNode(node.id)} tabIndex={0}
                onKeyDown={e => e.key === 'Enter' && setActiveNode(node.id)}
                role="button" aria-pressed={active}
                style={{
                  width: 140, padding: '14px 12px', borderRadius: 12, textAlign: 'center', cursor: 'pointer',
                  background: active ? `${nodeColor(node.type)}22` : 'var(--surface-2)',
                  border: `2px solid ${active ? nodeColor(node.type) : 'var(--border)'}`,
                  boxShadow: active ? `0 0 0 3px ${nodeColor(node.type)}33` : 'none',
                  transition: 'all 0.18s', animation: 'fadeUp 0.3s ease',
                  animationDelay: `${i * 0.05}s`, animationFillMode: 'both',
                }}>
                <div style={{ fontSize: 28, marginBottom: 6 }}>{nodeEmoji(node.type)}</div>
                <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-main)', marginBottom: 4 }}>{node.label ?? node.id}</div>
                <div style={{ fontSize: 9, color: 'var(--text-faint)', fontFamily: 'var(--font-mono)', marginBottom: 8 }}>{node.type}</div>
                <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 4 }}>
                  <StatusDot phase={phase} />
                  <span style={{ fontSize: 9, color: STATUS_COLORS[phase] ?? 'var(--text-faint)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>{phase}</span>
                </div>
              </div>
            )
          })}
        </div>

        <div style={{ position: 'absolute', bottom: 12, right: 16, fontSize: 10, color: 'var(--text-faint)' }}>
          Phase 5: XYFlow drag-and-drop integration
        </div>
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────
//  Debug Console (KTTM-REQ-014)
// ─────────────────────────────────────────────

function DebugView() {
  const { debugEvents, debugPaused, clearDebugEvents, setDebugPaused, addDebugEvent, activeNodeId } = useKttmStore()

  const simulateBreakpoint = () => {
    setDebugPaused(true)
    addDebugEvent({ type: 'paused', nodeId: activeNodeId ?? 'transform-python', timestamp: new Date().toISOString(), message: `Paused at node: ${activeNodeId ?? 'transform-python'}. Send 'continue' to proceed.`, payload: '{"amount":150,"customer":"Alice","phone":"+31612345678"}' })
  }

  const continueExecution = () => {
    setDebugPaused(false)
    addDebugEvent({ type: 'resumed', nodeId: activeNodeId ?? 'transform-python', timestamp: new Date().toISOString(), message: 'Developer resumed execution.' })
    setTimeout(() => addDebugEvent({ type: 'log', nodeId: activeNodeId ?? 'transform-python', timestamp: new Date().toISOString(), message: 'Processing row 4501/5000… chunk complete.' }), 300)
    setTimeout(() => addDebugEvent({ type: 'completed', nodeId: activeNodeId ?? 'transform-python', timestamp: new Date().toISOString(), message: 'Step output stream ended. Forwarding to sink-postgres.' }), 800)
  }

  const typeColors = { log: 'var(--text-muted)', paused: 'var(--status-warn)', resumed: 'var(--status-ok)', completed: 'var(--accent)', payload: 'var(--accent)' }

  return (
    <div className="form-view">
      <div className="form-header">
        <div className="form-eyebrow">KTTM-REQ-014 · Breakpoint Debugger</div>
        <h1 className="form-title">Debug Console</h1>
        <p className="form-description">Interactive step-through debugging. In cluster, the debug sidecar intercepts stdio and streams events here via WebSocket.</p>
      </div>

      <div style={{ display: 'flex', gap: 8, marginBottom: 16, flexWrap: 'wrap' }}>
        <button id="debug-breakpoint" className="btn btn-secondary" onClick={simulateBreakpoint} disabled={debugPaused}>⏸ Simulate Breakpoint</button>
        <button id="debug-continue" className="btn btn-primary" onClick={continueExecution} disabled={!debugPaused} style={{ opacity: debugPaused ? 1 : 0.4 }}>▶ Continue</button>
        <button className="btn btn-secondary" onClick={clearDebugEvents}>Clear Log</button>
        {debugPaused && <Badge color="var(--status-warn)">PAUSED</Badge>}
      </div>

      <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: 10, minHeight: 360, padding: 16, fontFamily: 'var(--font-mono)', fontSize: 11, display: 'flex', flexDirection: 'column', gap: 4 }} role="log" aria-label="Debug event log" aria-live="polite">
        {debugEvents.length === 0 && (
          <div style={{ color: 'var(--text-faint)', margin: 'auto' }}>No debug events yet. Click "Simulate Breakpoint" to begin.</div>
        )}
        {debugEvents.map((ev, i) => (
          <div key={i} style={{ display: 'flex', gap: 10, alignItems: 'flex-start', borderBottom: '1px solid var(--border)', paddingBottom: 4 }}>
            <span style={{ color: 'var(--text-faint)', flexShrink: 0 }}>{new Date(ev.timestamp).toLocaleTimeString()}</span>
            <span style={{ color: typeColors[ev.type] ?? 'var(--text-muted)', flexShrink: 0, textTransform: 'uppercase', letterSpacing: '0.05em' }}>{ev.type.padEnd(9)}</span>
            <span style={{ color: 'var(--accent)', flexShrink: 0 }}>[{ev.nodeId}]</span>
            <span style={{ color: 'var(--text-main)', wordBreak: 'break-all' }}>
              {ev.message}
              {ev.payload && <span style={{ color: 'var(--accent)', display: 'block', marginTop: 2 }}>→ {ev.payload}</span>}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────
//  Form View (End-User Portal) (KTTM-REQ-004, REQ-015)
// ─────────────────────────────────────────────

function FormField({ field, value, error, onChange }) {
  const handleChange = useCallback(e => {
    onChange(field.id, field.type === 'boolean' ? e.target.checked : e.target.value)
  }, [field.id, field.type, onChange])

  return (
    <div className="field-wrapper">
      {field.type !== 'boolean' && (
        <label className="field-label" htmlFor={`field-${field.id}`}>
          {field.label}
          {field.required && <span className="field-required">*</span>}
        </label>
      )}
      {field.description && <div className="field-description">{field.description}</div>}
      {field.type === 'select' && (
        <select id={`field-${field.id}`} className={`field-control ${error ? 'error' : ''}`} value={value ?? ''} onChange={handleChange} aria-invalid={!!error}>
          <option value="">Select…</option>
          {field.options?.map(opt => <option key={opt} value={opt}>{opt}</option>)}
        </select>
      )}
      {field.type === 'textarea' && (
        <textarea id={`field-${field.id}`} className={`field-control ${error ? 'error' : ''}`} value={value ?? ''} onChange={handleChange} placeholder={field.placeholder} rows={field.rows ?? 4} aria-invalid={!!error} />
      )}
      {field.type === 'boolean' && (
        <label className={`field-toggle ${value ? 'checked' : ''}`} htmlFor={`field-${field.id}`}>
          <input id={`field-${field.id}`} type="checkbox" checked={!!value} onChange={handleChange} />
          <div className="toggle-track" aria-hidden="true" />
          <span className="toggle-label">{field.label}</span>
        </label>
      )}
      {!['select', 'textarea', 'boolean'].includes(field.type) && (
        <input id={`field-${field.id}`} type={field.type === 'date' ? 'date' : field.type === 'number' ? 'number' : 'text'}
          className={`field-control ${error ? 'error' : ''}`} value={value ?? ''} onChange={handleChange}
          placeholder={field.placeholder} required={field.required} aria-invalid={!!error} />
      )}
      {error && <div className="field-error" role="alert"><span aria-hidden="true">⚠</span> {error}</div>}
    </div>
  )
}

function PayloadPreview({ values, action }) {
  let preview = { ...values }
  if (action?.payloadTemplate) {
    preview = Object.fromEntries(Object.entries(action.payloadTemplate).map(([k, v]) => [k, typeof v === 'string' ? v.replace(/\{\{(\w+)\}\}/g, (_, id) => values[id] ?? '') : v]))
  }
  const json = JSON.stringify(preview, null, 2)
  return (
    <div className="payload-preview">
      <div className="payload-preview-header">
        <div className="payload-label">Compiled Payload</div>
        <button className="payload-copy-btn" onClick={() => navigator.clipboard?.writeText(json)}>Copy JSON</button>
      </div>
      <pre className="payload-code">{json}</pre>
    </div>
  )
}

function FormView() {
  const { schema, values, errors, submitStatus, submitMessage, setValue, submit } = useKttmStore()
  const primaryAction = schema?.actions?.[0]

  if (!schema) return <div className="form-view"><div style={{ color: 'var(--text-muted)', margin: '60px auto' }}>Loading schema…</div></div>

  return (
    <div className="form-view">
      <div className="form-header">
        <div className="form-eyebrow">Business workspace · intake form</div>
        <h1 className="form-title">{schema.title}</h1>
        {schema.description && <p className="form-description">{schema.description}</p>}
      </div>

      {submitStatus === 'success' && (
        <div className="status-banner success" role="alert">
          <div className="status-banner-icon">✓</div>
          <div><div className="status-banner-title">Workflow Triggered</div><div className="status-banner-message">{submitMessage}</div></div>
        </div>
      )}
      {submitStatus === 'error' && (
        <div className="status-banner error" role="alert">
          <div className="status-banner-icon">✕</div>
          <div><div className="status-banner-title">Submission Failed</div><div className="status-banner-message">{submitMessage}</div></div>
        </div>
      )}

      <form id="workflow-form" noValidate onSubmit={e => { e.preventDefault(); primaryAction && submit(primaryAction) }}>
        <div className="fields-group">
          {schema.fields.map(field => (
            <FormField key={field.id} field={field} value={values[field.id]} error={errors[field.id]} onChange={setValue} />
          ))}
        </div>
        <div className="actions-bar">
          <div className="nats-subject">
            {primaryAction?.natsSubject && <><span>Routes to</span><code>{primaryAction.natsSubject.split('.').slice(-2).join('.')}</code></>}
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            {schema.actions?.map(action => (
              <button key={action.id} id={`action-${action.id}`}
                type={action.style === 'primary' ? 'submit' : 'button'}
                className={`btn btn-${action.style === 'primary' ? 'primary' : 'secondary'} ${submitStatus === 'submitting' ? 'submitting' : ''}`}
                onClick={() => action.style !== 'primary' && submit(action)}
                disabled={submitStatus === 'submitting'}>
                <span className="btn-text">{action.label}</span>
                {action.style === 'primary' && <span className="btn-icon" aria-hidden="true">→</span>}
              </button>
            ))}
          </div>
        </div>
      </form>
      {Object.keys(values).length > 0 && primaryAction && <PayloadPreview values={values} action={primaryAction} />}
    </div>
  )
}

// ─────────────────────────────────────────────
//  History View
// ─────────────────────────────────────────────

function HistoryView() {
  const { history } = useKttmStore()
  return (
    <div className="form-view">
      <div className="form-header">
        <div className="form-eyebrow">Execution log</div>
        <h1 className="form-title">Run History</h1>
      </div>
      <div className="history-list" role="list">
        {history.map((run, i) => (
          <div key={run.id + i} className="history-item" role="listitem">
            <StatusDot phase={run.phase} size={10} />
            <div className="history-meta">
              <div className="history-id">{run.id}</div>
              <div className="history-time">{run.startedAt} · {run.duration}</div>
            </div>
            <Pill color={STATUS_COLORS[run.phase]}>{run.phase}</Pill>
          </div>
        ))}
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────
//  Schema View
// ─────────────────────────────────────────────

function SchemaView() {
  const { schema, schemaSource } = useKttmStore()
  return (
    <div className="form-view">
      <div className="form-header">
        <div className="form-eyebrow">ConfigMap · {schemaSource}</div>
        <h1 className="form-title">UI Layout Schema</h1>
        <p className="form-description">JSON schema loaded at runtime from Kubernetes ConfigMap. No rebuild required (KTTM-REQ-017).</p>
      </div>
      <div className="payload-preview">
        <div className="payload-preview-header">
          <div className="payload-label">kttm-ui-{schema?.workflowId ?? 'schema'}</div>
          <button className="payload-copy-btn" onClick={() => navigator.clipboard?.writeText(JSON.stringify(schema, null, 2))}>Copy JSON</button>
        </div>
        <pre className="payload-code">{JSON.stringify(schema, null, 2)}</pre>
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────
//  Admin — Users & Roles (KTTM-REQ-003, REQ-005, REQ-039)
// ─────────────────────────────────────────────

function RBACMatrix({ user, onSave, onCancel }) {
  const { allPermissions } = useKttmStore()
  const [perms, setPerms] = useState(new Set(user.permissions))
  const toggle = (p) => setPerms(prev => { const n = new Set(prev); n.has(p) ? n.delete(p) : n.add(p); return n })

  return (
    <div style={{ background: 'var(--surface-1)', border: '1px solid var(--accent)', borderRadius: 12, padding: 20 }}>
      <div style={{ fontWeight: 600, marginBottom: 14, color: 'var(--text-main)' }}>
        Editing permissions for <code style={{ color: 'var(--accent)' }}>{user.name}</code>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: 8, marginBottom: 16 }}>
        {allPermissions.map(p => (
          <label key={p} style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', padding: '6px 10px', borderRadius: 8, background: perms.has(p) ? 'var(--accent)18' : 'var(--surface-2)', border: `1px solid ${perms.has(p) ? 'var(--accent)' : 'var(--border)'}`, transition: 'all 0.12s' }}>
            <input type="checkbox" checked={perms.has(p)} onChange={() => toggle(p)} style={{ accentColor: 'var(--accent)' }} />
            <span style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: perms.has(p) ? 'var(--accent)' : 'var(--text-muted)' }}>{p}</span>
          </label>
        ))}
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        <button className="btn btn-primary" onClick={() => onSave([...perms])}>Save Permissions</button>
        <button className="btn btn-secondary" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}

function AdminUsersView() {
  const { users, editingUser, setEditingUser, updateUserPermissions } = useKttmStore()

  return (
    <div className="form-view">
      <div className="form-header">
        <div className="form-eyebrow">KTTM-REQ-003 · KTTM-REQ-005 · KTTM-REQ-039</div>
        <h1 className="form-title">Users & Roles</h1>
        <p className="form-description">Grant atomic permissions to users. Combine any subset to create custom roles (KTTM-REQ-005). Backed by Kubernetes RBAC (KTTM-REQ-039).</p>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 10, marginBottom: 24 }} role="list">
        {users.map(u => (
          <div key={u.name} className="history-item" style={{ alignItems: 'center' }} role="listitem">
            <div style={{ width: 36, height: 36, borderRadius: '50%', background: 'var(--accent)22', color: 'var(--accent)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 14, flexShrink: 0 }}>
              {u.name[0].toUpperCase()}
            </div>
            <div className="history-meta" style={{ flex: 1 }}>
              <div className="history-id">{u.name}</div>
              <div className="history-time" style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginTop: 4 }}>
                {u.permissions.slice(0, 5).map(p => <Pill key={p} color="var(--accent)">{p}</Pill>)}
                {u.permissions.length > 5 && <Pill color="var(--text-faint)">+{u.permissions.length - 5} more</Pill>}
              </div>
            </div>
            <button className="btn btn-secondary" onClick={() => setEditingUser(u)} style={{ flexShrink: 0 }}>Edit Permissions</button>
          </div>
        ))}
      </div>

      {editingUser && (
        <RBACMatrix
          user={editingUser}
          onSave={perms => updateUserPermissions(editingUser.name, perms)}
          onCancel={() => setEditingUser(null)}
        />
      )}
    </div>
  )
}

// ─────────────────────────────────────────────
//  Admin — Lifecycle (KTTM-REQ-003, REQ-032)
// ─────────────────────────────────────────────

function AdminLifecycleView() {
  const { lifecycleOp, lifecycleLog, setLifecycleOp, addLifecycleLog } = useKttmStore()

  const runOp = (op) => {
    setLifecycleOp(op)
    const messages = {
      installing: ['Pulling images from embedded Zot registry…', 'Applying Argo Workflows CRDs…', 'Bootstrapping NATS JetStream…', 'Checking Kyverno policies…', '✓ Installation complete.'],
      upgrading:  ['Fetching new bundle manifest…', 'Rolling update: kttm-operator (0/3 → 3/3)…', 'Rolling update: kttm-bff…', '✓ Zero-downtime upgrade complete.'],
      uninstalling:['Draining active workflow pods…', 'Removing CRDs…', 'Cleaning Zot registry…', '✓ Uninstall complete.'],
    }
    let delay = 0
    messages[op].forEach((msg, i) => {
      delay += 800 + i * 200
      setTimeout(() => addLifecycleLog(msg), delay)
    })
    setTimeout(() => setLifecycleOp(null), delay + 500)
  }

  return (
    <div className="form-view">
      <div className="form-header">
        <div className="form-eyebrow">KTTM-REQ-003 · REQ-032 · Zero-downtime lifecycle</div>
        <h1 className="form-title">Cluster Lifecycle</h1>
        <p className="form-description">Install, upgrade, or uninstall KTTM workloads using rolling zero-downtime deployments. All operations go through the Go operator.</p>
      </div>

      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 20 }}>
        {[['installing', 'Install', 'btn-primary'], ['upgrading', 'Upgrade', 'btn-secondary'], ['uninstalling', 'Uninstall', 'btn-secondary']].map(([op, label, cls]) => (
          <button key={op} id={`lifecycle-${op}`} className={`btn ${cls}`} onClick={() => runOp(op)} disabled={!!lifecycleOp} style={{ opacity: lifecycleOp ? 0.5 : 1 }}>
            {lifecycleOp === op ? `${label}ing…` : label}
          </button>
        ))}
      </div>

      {lifecycleLog.length > 0 && (
        <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border)', borderRadius: 10, padding: 16, fontFamily: 'var(--font-mono)', fontSize: 12 }} role="log">
          {lifecycleLog.map((entry, i) => (
            <div key={i} style={{ color: entry.msg.startsWith('✓') ? 'var(--status-ok)' : 'var(--text-muted)', marginBottom: 4 }}>
              <span style={{ color: 'var(--text-faint)' }}>{new Date(entry.ts).toLocaleTimeString()} </span>{entry.msg}
            </div>
          ))}
          {lifecycleOp && <div style={{ color: 'var(--text-faint)', animation: 'pulse 1s infinite' }}>…</div>}
        </div>
      )}
    </div>
  )
}

// ─────────────────────────────────────────────
//  Admin — Cost Dashboard (KTTM-REQ-028)
// ─────────────────────────────────────────────

const COST_DATA = [
  { node: 'transform-python', daily: '$0.42', monthly: '$12.60', cpu: '1.0 vCPU', mem: '512Mi' },
  { node: 'source-s3',        daily: '$0.08', monthly: '$2.40',  cpu: '0.5 vCPU', mem: '256Mi' },
  { node: 'sink-postgres',    daily: '$0.15', monthly: '$4.50',  cpu: '0.5 vCPU', mem: '256Mi' },
  { node: 'report-pdf',       daily: '$0.12', monthly: '$3.60',  cpu: '0.5 vCPU', mem: '256Mi' },
  { node: 'sink-slack',       daily: '$0.03', monthly: '$0.90',  cpu: '0.1 vCPU', mem: '128Mi' },
]

function AdminCostView() {
  return (
    <div className="form-view">
      <div className="form-header">
        <div className="form-eyebrow">KTTM-REQ-028 · OpenCost Integration</div>
        <h1 className="form-title">Infrastructure Cost</h1>
        <p className="form-description">Granular cost breakdown per workflow node via OpenCost. Helps identify expensive steps and optimize cluster resource allocation.</p>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, marginBottom: 24 }}>
        {[['Total Daily', '$0.80'], ['Total Monthly', '$24.00'], ['Savings Potential', '−$6.50']].map(([label, val]) => (
          <div key={label} style={{ background: 'var(--surface-2)', border: '1px solid var(--border)', borderRadius: 10, padding: '16px 20px' }}>
            <div style={{ fontSize: 11, color: 'var(--text-faint)', marginBottom: 4 }}>{label}</div>
            <div style={{ fontSize: 24, fontWeight: 700, color: label === 'Savings Potential' ? 'var(--status-ok)' : 'var(--text-main)' }}>{val}</div>
          </div>
        ))}
      </div>

      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
          <thead>
            <tr style={{ borderBottom: '2px solid var(--border)' }}>
              {['Node', 'CPU Limit', 'Memory Limit', 'Daily Cost', 'Monthly Cost'].map(h => (
                <th key={h} style={{ textAlign: 'left', padding: '8px 12px', color: 'var(--text-faint)', fontWeight: 500, fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em' }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {COST_DATA.map((row, i) => (
              <tr key={row.node} style={{ borderBottom: '1px solid var(--border)', background: i % 2 === 0 ? 'transparent' : 'var(--surface-1)' }}>
                <td style={{ padding: '10px 12px', color: 'var(--text-main)', fontFamily: 'var(--font-mono)', fontSize: 12 }}>{row.node}</td>
                <td style={{ padding: '10px 12px', color: 'var(--text-muted)' }}>{row.cpu}</td>
                <td style={{ padding: '10px 12px', color: 'var(--text-muted)' }}>{row.mem}</td>
                <td style={{ padding: '10px 12px', color: 'var(--text-main)', fontWeight: 600 }}>{row.daily}</td>
                <td style={{ padding: '10px 12px', color: 'var(--accent)' }}>{row.monthly}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────
//  AI Advisor Panel (KTTM-REQ-026)
// ─────────────────────────────────────────────

function MetricBar({ value, max = 100, pct }) {
  const p = pct ?? Math.min((value / max) * 100, 100)
  const color = p >= 80 ? 'var(--status-err)' : p >= 60 ? 'var(--status-warn)' : 'var(--status-ok)'
  return (
    <div className="metric-bar">
      <div className="metric-bar-fill" style={{ width: `${p}%`, background: color }} role="progressbar" aria-valuenow={p} aria-valuemin={0} aria-valuemax={100} />
    </div>
  )
}

function AdvisorPanel() {
  const { advice, metrics, activeNodeId } = useKttmStore()
  const filteredAdvice = activeNodeId ? advice.filter(a => a.nodeId === activeNodeId) : advice
  const m = activeNodeId ? metrics[activeNodeId] : null

  return (
    <aside className="advisor-panel" aria-label="AI Advisor">
      <div className="advisor-header">
        <div className="advisor-title"><div className="advisor-icon">✦</div> AI Advisor</div>
        <span className="advisor-badge">LIVE</span>
      </div>
      <div className="advisor-body">
        {m && (
          <div style={{ marginBottom: 14 }}>
            <div className="sidebar-label" style={{ marginBottom: 8 }}>Metrics · {activeNodeId}</div>
            <div className="metrics-grid">
              {[
                { label: 'CPU', value: m.cpu, unit: '%', pct: m.cpu },
                { label: 'Memory', value: Math.round(m.mem), unit: 'MB', pct: Math.round(m.mem / m.memLimit * 100) },
                { label: 'Duration', value: m.durationMs, unit: 'ms', pct: Math.min(m.durationMs / 5000 * 100, 100) },
                { label: 'Error Rate', value: m.errorRate, unit: '%', pct: Math.min(m.errorRate * 10, 100) },
              ].map(({ label, value, unit, pct }) => (
                <div key={label} className="metric-tile">
                  <div className="metric-label">{label}</div>
                  <div className="metric-value">{value}<span className="metric-unit">{unit}</span></div>
                  <MetricBar pct={pct} />
                </div>
              ))}
            </div>
          </div>
        )}

        {filteredAdvice.length === 0 ? (
          <div className="advisor-empty">
            <div className="advisor-empty-icon">✦</div>
            <div className="advisor-empty-text">{activeNodeId ? `No issues for ${activeNodeId}` : 'All nodes normal'}</div>
          </div>
        ) : (
          <div role="list">
            {filteredAdvice.map((item, i) => (
              <div key={i} className={`advice-card ${item.severity}`} role="listitem" style={{ animationDelay: `${i * 0.06}s` }}>
                <div className="advice-header">
                  <div className="advice-severity-dot" />
                  <div className="advice-title">{item.title}</div>
                  <div className="advice-node-tag">{item.nodeId}</div>
                </div>
                <div className="advice-message">{item.message}</div>
                <div className="advice-action">{item.action}</div>
              </div>
            ))}
          </div>
        )}
      </div>
      <div className="advisor-footer">
        <div className="advisor-poll-info"><div className="poll-dot" /> Prometheus · 60s</div>
      </div>
    </aside>
  )
}

// ─────────────────────────────────────────────
//  Root App — Plane router (KTTM-REQ-001)
// ─────────────────────────────────────────────

function App() {
  const { plane, activeTab, loadSchema, setNatsStatus } = useKttmStore()

  useEffect(() => {
    loadSchema()
    setNatsStatus('connecting')
    const t = setTimeout(() => setNatsStatus('connected'), 1400)
    return () => clearTimeout(t)
  }, [])

  // Developer SDK — full canvas + debug + connectors (KTTM-REQ-002)
  const renderDevContent = () => {
    switch (activeTab) {
      case 'canvas':     return <CanvasView />
      case 'connectors': return <ConnectorCatalogView />
      case 'debug':      return <PermissionGate required="app:debug" fallback={<div className="form-view"><p>Requires app:debug permission.</p></div>}><DebugView /></PermissionGate>
      case 'history':    return <HistoryView />
      case 'schema':     return <SchemaView />
      default:           return <FormView />
    }
  }

  // Admin Console — users, lifecycle, cost (KTTM-REQ-003)
  const renderAdminContent = () => {
    switch (activeTab) {
      case 'admin-users':     return <AdminUsersView />
      case 'admin-lifecycle': return <AdminLifecycleView />
      case 'admin-cost':      return <PermissionGate required="cost:view"><AdminCostView /></PermissionGate>
      case 'admin-audit':     return (
        <div className="form-view">
          <div className="form-header">
            <div className="form-eyebrow">KTTM-REQ-039 · Immutable audit log</div>
            <h1 className="form-title">Audit Log</h1>
          </div>
          <div style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 11, lineHeight: 1.8 }}>
            {['2026-09-11T20:45:12Z alice@acme.com  app:execute    daily-ledger-etl  → triggered', '2026-09-11T19:30:00Z alice@acme.com  app:execute    daily-ledger-etl  → failed', '2026-09-11T14:22:08Z bob@acme.com    infra:upgrade  kttm-operator     → v1.3.0'].map((l, i) => <div key={i}>{l}</div>)}
          </div>
        </div>
      )
      case 'admin-bundles': return (
        <div className="form-view">
          <div className="form-header">
            <div className="form-eyebrow">KTTM-REQ-020 · KTTM-REQ-042</div>
            <h1 className="form-title">Bundle Manager</h1>
            <p className="form-description">Import .tar.gz bundles for air-gapped deployment. Bundles are extracted into the embedded Zot OCI registry.</p>
          </div>
          <div style={{ border: '2px dashed var(--border)', borderRadius: 12, padding: 40, textAlign: 'center', color: 'var(--text-muted)' }}>
            <div style={{ fontSize: 32, marginBottom: 12 }}>📦</div>
            <div>Drop bundle.tar.gz here or <button className="btn btn-secondary" style={{ display: 'inline-block' }}>Browse</button></div>
          </div>
        </div>
      )
      default: return <AdminUsersView />
    }
  }

  // End-User Portal — forms and dashboards only (KTTM-REQ-004)
  const renderUserContent = () => {
    switch (activeTab) {
      case 'history': return <HistoryView />
      default:        return <FormView />
    }
  }

  return (
    <div className="app-shell">
      <Topbar />
      <DagSidebar />
      <main className="main-content" id="main-content" role="main">
        {plane === 'developer' && renderDevContent()}
        {plane === 'admin' && renderAdminContent()}
        {plane === 'enduser' && renderUserContent()}
      </main>
      {/* Advisor panel shown for developers only — admins have cost panel, users see nothing */}
      {plane === 'developer' && <AdvisorPanel />}
      {plane === 'admin' && activeTab !== 'admin-cost' && <AdvisorPanel />}
    </div>
  )
}

createRoot(document.getElementById('root')).render(<StrictMode><App /></StrictMode>)