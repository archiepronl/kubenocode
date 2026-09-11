/**
 * KTTM — Zustand State Store
 *
 * Implements all client-side reactive state for the KTTM platform:
 * - Authentication & permission resolution (KTTM-REQ-001, REQ-039)
 * - Schema loading from K8s ConfigMap (KTTM-REQ-011)
 * - NATS WebSocket bridge for live updates (KTTM-REQ-031)
 * - AI Advisor metric polling (KTTM-REQ-026)
 * - Form values + validation (KTTM-REQ-015)
 * - Workflow execution history (KTTM-REQ-032)
 */
import { create } from 'zustand'
import { subscribeWithSelector } from 'zustand/middleware'

// ── Permission helpers ─────────────────────────────────────────────────────

const ALL_PERMISSIONS = [
  'app:create', 'app:modify', 'app:delete', 'app:debug',
  'app:export', 'app:execute', 'rbac:manage', 'infra:install',
  'infra:upgrade', 'bundle:import', 'audit:view', 'cost:view',
]

const DEFAULT_ROLES = {
  developer: ['app:create', 'app:modify', 'app:delete', 'app:debug', 'app:export', 'app:execute', 'audit:view'],
  admin: ['rbac:manage', 'infra:install', 'infra:upgrade', 'bundle:import', 'app:export', 'app:execute', 'audit:view', 'cost:view'],
  enduser: ['app:execute'],
}

/**
 * Resolves the UI plane based on current permissions.
 * KTTM-REQ-001: Single SPA, permission-driven slot masking.
 */
function resolvePlane(permissions = []) {
  const ps = new Set(permissions)
  if (ps.has('rbac:manage')) return 'admin'
  if (ps.has('app:create') || ps.has('app:debug')) return 'developer'
  return 'enduser'
}

// ── Mock schema (replaced at runtime by ConfigMap fetch) ──────────────────

const DEMO_SCHEMA = {
  workflowId: 'daily-ledger-etl',
  title: 'Daily Ledger ETL',
  description: 'Process and reconcile the daily financial ledger against upstream S3 data.',
  fields: [
    { id: 'date', type: 'date', label: 'Processing Date', required: true, placeholder: '2024-01-15' },
    { id: 'bucket', type: 'text', label: 'S3 Bucket', required: true, placeholder: 'acme-lake', description: 'Source S3 bucket containing ledger CSVs.' },
    { id: 'env', type: 'select', label: 'Environment', options: ['production', 'staging', 'development'], required: true },
    { id: 'batchSize', type: 'number', label: 'Batch Size', placeholder: '5000', description: 'Rows per micro-batch (KTTM-REQ-031).' },
    { id: 'enableValidation', type: 'boolean', label: 'Enable Schema Validation', description: 'Run JSON schema check before transformation.' },
    { id: 'outputTable', type: 'text', label: 'Postgres Target Table', placeholder: 'public.ledger_daily' },
    { id: 'notes', type: 'textarea', label: 'Run Notes', placeholder: 'Optional operator notes…', rows: 3 },
  ],
  actions: [
    {
      id: 'trigger',
      label: 'Launch Pipeline',
      style: 'primary',
      natsSubject: 'kttm.apps.daily-ledger-etl.trigger',
      payloadTemplate: { date: '{{date}}', bucket: '{{bucket}}', env: '{{env}}', batchSize: '{{batchSize}}' },
    },
    { id: 'validate', label: 'Validate Only', style: 'secondary', natsSubject: 'kttm.apps.daily-ledger-etl.validate' },
  ],
  workflowDag: [
    { id: 'source-s3', type: 'connector/s3', label: 'Read Ledger CSV' },
    { id: 'validate-schema', type: 'logic/function', label: 'Schema Validate' },
    { id: 'transform-python', type: 'script/python', label: 'Filter & Normalize' },
    { id: 'fan-out', type: 'logic/fanout', label: 'Parallel Fan-Out' },
    { id: 'sink-postgres', type: 'sink/postgres', label: 'Write to Postgres' },
    { id: 'sink-slack', type: 'sink/slack', label: 'Notify Slack' },
    { id: 'report-pdf', type: 'sink/report-pdf', label: 'Generate PDF' },
  ],
}

const DEMO_CONNECTORS = [
  { type: 'connector/s3', label: 'S3 / MinIO', category: 'Storage', icon: '☁', description: 'Read or write files from S3, GCS, or MinIO object storage.' },
  { type: 'connector/postgres', label: 'PostgreSQL', category: 'Database', icon: '🐘', description: 'Zero-copy Arrow IPC streaming to/from PostgreSQL.' },
  { type: 'connector/kafka', label: 'Apache Kafka', category: 'Messaging', icon: '🌊', description: 'Consume or produce Kafka topics with SASL_SSL support.' },
  { type: 'connector/nats', label: 'NATS', category: 'Messaging', icon: '⚡', description: 'Publish or subscribe to NATS JetStream subjects.' },
  { type: 'trigger/webhook', label: 'Webhook', category: 'Trigger', icon: '🔔', description: 'Expose an HTTP endpoint with HMAC-SHA256 verification.' },
  { type: 'trigger/cron', label: 'Cron Scheduler', category: 'Trigger', icon: '🕐', description: 'Trigger workflows on time-based schedules.' },
  { type: 'trigger/file-watch', label: 'File Watcher', category: 'Trigger', icon: '📁', description: 'Watch S3 buckets or filesystem paths for file events.' },
  { type: 'trigger/mqtt', label: 'MQTT / IoT', category: 'Trigger', icon: '📡', description: 'Listen to MQTT topics from IoT devices and edge sensors.' },
  { type: 'script/python', label: 'Python Script', category: 'Script', icon: '🐍', description: 'Run arbitrary Python with pandas, sklearn, or any pip package.' },
  { type: 'script/javascript', label: 'JavaScript', category: 'Script', icon: '🟨', description: 'Execute Node.js scripts with npm package support.' },
  { type: 'script/bash', label: 'Bash Script', category: 'Script', icon: '🖥', description: 'Run shell scripts inside an isolated Alpine container.' },
  { type: 'script/r', label: 'R Analytics', category: 'Script', icon: '📊', description: 'Statistical computing with R and CRAN packages.' },
  { type: 'logic/fanout', label: 'Parallel Fan-Out', category: 'Logic', icon: '⑃', description: 'Split workflow into parallel execution branches.' },
  { type: 'logic/barrier', label: 'Barrier / Join', category: 'Logic', icon: '⑄', description: 'Wait for all parallel branches before continuing.' },
  { type: 'logic/branch', label: 'Branch Filter', category: 'Logic', icon: '↔', description: 'Route data along different paths using CEL expressions.' },
  { type: 'logic/aggregate', label: 'Micro-Batch Aggregator', category: 'Logic', icon: '📦', description: 'Group streaming records into configurable micro-batches.' },
  { type: 'sink/report-pdf', label: 'PDF Report', category: 'Sink', icon: '📄', description: 'Generate structured PDF reports from processed data.' },
  { type: 'sink/slack', label: 'Slack Alert', category: 'Sink', icon: '💬', description: 'Send alerts and results to Slack channels.' },
  { type: 'sink/email', label: 'Email', category: 'Sink', icon: '📧', description: 'Send emails via SMTP with attachment support.' },
  { type: 'sink/mongodb', label: 'MongoDB', category: 'Database', icon: '🍃', description: 'Write processed data to MongoDB collections.' },
]

const DEMO_ADVICE = [
  { nodeId: 'transform-python', severity: 'critical', title: 'Memory Near Limit', message: 'Container is using 92% of its 512Mi limit. At current trajectory, OOMKill imminent.', action: 'Increase memory limit to 1Gi or enable micro-batching (batchSize=1000).' },
  { nodeId: 'source-s3', severity: 'warning', title: 'High Read Latency', message: 'S3 GetObject p99 = 2.3s. This is 3× higher than baseline.', action: 'Enable S3 Transfer Acceleration or switch to an in-region endpoint.' },
  { nodeId: 'sink-postgres', severity: 'info', title: 'Batch Insert Optimization Available', message: 'Single-row INSERTs detected. COPY protocol is 40× faster for bulk loads.', action: 'Use COPY FROM STDIN — already supported by the postgres connector.' },
]

const DEMO_METRICS = {
  'source-s3':       { cpu: 12, mem: 120, memLimit: 256, durationMs: 2300, errorRate: 0 },
  'validate-schema': { cpu: 5,  mem: 45,  memLimit: 128, durationMs: 80,   errorRate: 0.2 },
  'transform-python':{ cpu: 78, mem: 470, memLimit: 512, durationMs: 4200, errorRate: 1.1 },
  'fan-out':         { cpu: 2,  mem: 30,  memLimit: 128, durationMs: 5,    errorRate: 0 },
  'sink-postgres':   { cpu: 22, mem: 180, memLimit: 256, durationMs: 650,  errorRate: 0.4 },
  'sink-slack':      { cpu: 3,  mem: 40,  memLimit: 128, durationMs: 320,  errorRate: 0 },
  'report-pdf':      { cpu: 45, mem: 220, memLimit: 256, durationMs: 1800, errorRate: 0 },
}

const DEMO_HISTORY = [
  { id: 'kttm-etl-x7k2p', phase: 'succeeded', startedAt: '2026-09-11 20:45', duration: '4m 12s' },
  { id: 'kttm-etl-n3q9r', phase: 'failed',    startedAt: '2026-09-11 19:30', duration: '1m 08s' },
  { id: 'kttm-etl-m8w4t', phase: 'succeeded', startedAt: '2026-09-11 18:15', duration: '3m 55s' },
  { id: 'kttm-etl-b1z6v', phase: 'succeeded', startedAt: '2026-09-11 17:00', duration: '4m 33s' },
]

const DEMO_USERS = [
  { name: 'alice@acme.com', role: 'developer', permissions: DEFAULT_ROLES.developer },
  { name: 'bob@acme.com',   role: 'admin',     permissions: DEFAULT_ROLES.admin },
  { name: 'carol@acme.com', role: 'enduser',   permissions: DEFAULT_ROLES.enduser },
]

// ── Store ──────────────────────────────────────────────────────────────────

export const useKttmStore = create(
  subscribeWithSelector((set, get) => ({

    // ── Auth / RBAC ─────────────────────────────────────────────────────────
    currentUser: { name: 'alice@acme.com', role: 'developer', permissions: DEFAULT_ROLES.developer },
    plane: 'developer', // developer | admin | enduser
    allPermissions: ALL_PERMISSIONS,
    allRoles: DEFAULT_ROLES,
    users: DEMO_USERS,

    setUser: (user) => set({ currentUser: user, plane: resolvePlane(user.permissions) }),

    hasPermission: (perm) => {
      const { currentUser } = get()
      return currentUser.permissions.includes(perm)
    },

    // ── Schema ──────────────────────────────────────────────────────────────
    schema: DEMO_SCHEMA,
    schemaSource: 'configmap:fsa-ui-daily-ledger-etl',
    connectors: DEMO_CONNECTORS,
    connectorSearch: '',
    activeCategory: 'All',

    loadSchema: async () => {
      try {
        const res = await fetch('/api/schema')
        if (res.ok) {
          const schema = await res.json()
          set({ schema, schemaSource: 'configmap:live' })
        }
      } catch {
        // Remain on demo schema — offline / dev mode
      }
    },

    setConnectorSearch: (q) => set({ connectorSearch: q }),
    setActiveCategory: (cat) => set({ activeCategory: cat }),

    filteredConnectors: () => {
      const { connectors, connectorSearch, activeCategory } = get()
      return connectors.filter(c => {
        const matchSearch = !connectorSearch || c.label.toLowerCase().includes(connectorSearch.toLowerCase()) || c.type.toLowerCase().includes(connectorSearch.toLowerCase())
        const matchCat = activeCategory === 'All' || c.category === activeCategory
        return matchSearch && matchCat
      })
    },

    // ── NATS / connection ────────────────────────────────────────────────────
    natsStatus: 'disconnected', // disconnected | connecting | connected | error
    setNatsStatus: (s) => set({ natsStatus: s }),

    // ── DAG / canvas ─────────────────────────────────────────────────────────
    activeNodeId: null,
    nodeStatuses: { 'source-s3': 'succeeded', 'validate-schema': 'succeeded', 'transform-python': 'running', 'fan-out': 'idle', 'sink-postgres': 'idle', 'sink-slack': 'idle', 'report-pdf': 'idle' },
    metrics: DEMO_METRICS,
    advice: DEMO_ADVICE,
    advisorLastPoll: new Date().toISOString(),

    setActiveNode: (id) => set({ activeNodeId: id }),

    // ── Form (End-User Portal) ───────────────────────────────────────────────
    values: {},
    errors: {},
    submitStatus: null, // null | 'submitting' | 'success' | 'error'
    submitMessage: '',

    setValue: (id, val) => set(s => ({
      values: { ...s.values, [id]: val },
      errors: { ...s.errors, [id]: undefined },
    })),

    submit: async (action) => {
      const { schema, values } = get()
      // Validate required fields
      const errs = {}
      schema.fields.forEach(f => {
        if (f.required && !values[f.id]) errs[f.id] = 'This field is required.'
      })
      if (Object.keys(errs).length) {
        set({ errors: errs })
        return
      }
      set({ submitStatus: 'submitting', submitMessage: '' })
      try {
        await fetch('/api/trigger', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: action.id, values }),
        })
        set({ submitStatus: 'success', submitMessage: `Workflow triggered successfully. Check History for live status.` })
      } catch {
        set({ submitStatus: 'error', submitMessage: 'Network error — NATS bridge unavailable. Check cluster connectivity.' })
      }
      setTimeout(() => set({ submitStatus: null }), 6000)
    },

    // ── History ──────────────────────────────────────────────────────────────
    history: DEMO_HISTORY,

    // ── Tabs ─────────────────────────────────────────────────────────────────
    activeTab: 'form', // form | canvas | history | schema | connectors | debug | admin
    setActiveTab: (t) => set({ activeTab: t }),

    // ── Debug ─────────────────────────────────────────────────────────────────
    debugEvents: [],
    debugPaused: false,
    addDebugEvent: (ev) => set(s => ({ debugEvents: [ev, ...s.debugEvents].slice(0, 200) })),
    clearDebugEvents: () => set({ debugEvents: [] }),
    setDebugPaused: (v) => set({ debugPaused: v }),

    // ── Admin: users & roles ──────────────────────────────────────────────────
    editingUser: null,
    setEditingUser: (u) => set({ editingUser: u }),
    updateUserPermissions: (username, perms) => set(s => ({
      users: s.users.map(u => u.name === username ? { ...u, permissions: perms } : u),
      editingUser: null,
    })),

    // ── Admin: lifecycle ──────────────────────────────────────────────────────
    lifecycleOp: null, // null | 'installing' | 'upgrading' | 'uninstalling'
    lifecycleLog: [],
    setLifecycleOp: (op) => set({ lifecycleOp: op, lifecycleLog: [] }),
    addLifecycleLog: (msg) => set(s => ({ lifecycleLog: [...s.lifecycleLog, { msg, ts: new Date().toISOString() }] })),
  }))
)
