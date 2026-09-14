# KTTM — Full Product Execution Plan

**Project:** KTTM (கட்டும் — *Kattum*) · Kubernetes-Native Workflow & No-Code Platform  
**Planning Model:** Agile · Vertical Feature Slices · Living Document  
**Last Updated:** September 2026

> [!IMPORTANT]
> **How to read this plan:** Every listed feature is a **vertical slice** — a complete, runnable, user-testable piece of functionality. No mocking, no placeholder UIs. Real data, real Kubernetes, real users. Scope evolves; this document is the authoritative tracker.

---

## Quick Reference: Milestones

| Milestone | Target | Phases Covered | Key Capability Unlocked |
|---|---|---|---|
| **M0 · Dev Foundation** | Month 1 | Phase 0 | CI/CD, local cluster, testing harness — any engineer can onboard in 1 command |
| **M1 · ETL MVP** | Month 3 | Phase 1–2 | Real pipeline from S3 → transform → Postgres, live on k3d |
| **M2 · Visual Flow Benchmark** | Month 5 | Phase 3–4 | Node-RED & n8n feature parity on canvas, public demo ready |
| **M3 · No-Code App Builder** | Month 7 | Phase 5–6 | End-user builds a working web app with zero code |
| **M4 · Enterprise Security** | Month 9 | Phase 7 | RBAC, audit, CVE scan, CIS benchmark, SOC 2 path |
| **M5 · Platform Scale** | Month 11 | Phase 8–9 | KEDA auto-scale, HA, multi-cluster GitOps, algorithm engine |
| **M6 · CNCF Sandbox** | Month 14 | Phase 10–11 | SPIFFE/SPIRE, OPA, Artifact Hub listing, governance |
| **M7 · Marketplace & GA** | Month 18+ | Phase 12–13 | AWS/GCP/Azure marketplace, one-click install, community |

---

## Phase 0 — Development Foundation & DevOps Baseline
**Theme:** Any engineer can clone the repo and have a running KTTM cluster in under 10 minutes.  
**Milestone:** M0 · Dev Foundation

### Features

#### F0.1 — Local Multi-Node K3d Cluster Bootstrap Script
- One-command setup: `make setup` executes `scripts/setup-cluster.sh` to spin up a k3d cluster (1 server, 2 agents) optimized for memory on Apple Silicon.
- Activates vertical in-place auto-scaling feature gates.
- Installs: KEDA Event-Driven Horizontal Scaler natively.
- `make clean` tears down cleanly.
- **User test:** Developer runs `make setup`, opens k3d cluster info and verifies KEDA operator pod is ready.
- **Reqs:** KTTM-NFR-001, KTTM-REQ-044

#### F0.2 — Continuous Live-Sync Pipeline (Skaffold)
- Trigger: `make dev` launches `skaffold dev` for reactive live-code compilation.
- Builds native ARM64 development containers locally without pushing to slow external registries.
- Hot-swaps Go code and React workspace changes into the local node registry in under 3 seconds.
- Auto-hydrates Custom Resource manifests inside the cluster.
- **User test:** Edit `cmd/operator/main.go` → Skaffold rebuilds and restarts the operator pod almost instantly.
- **Reqs:** KTTM-NFR-003, KTTM-NFR-004

#### F0.3 — Release Pipeline & Versioning
- Semantic versioning via `git tag v0.1.0`
- GitHub Actions release job: builds binaries, Docker images, Helm chart tarball
- Images pushed to `ghcr.io/kubeworkflow/kttm-operator` and `ghcr.io/kubeworkflow/kttm-bff`
- `goreleaser` for CLI binaries (Linux/macOS/Windows AMD64+ARM64)
- **User test:** Create `v0.1.0` tag → GitHub Releases page shows downloadable binaries + Docker images
- **Reqs:** KTTM-NFR-004

#### F0.4 — Go Module & Dependency Structure
- Single `go.mod` root with proper module path `github.com/kubeworkflow/kttm`
- `Makefile` targets: `generate` (deepcopy), `manifests` (CRD YAML), `fmt`, `vet`, `lint`
- Pre-commit hooks via `lefthook`: auto-runs `fmt`, `vet`, `lint` before every commit
- **User test:** Run `make generate && make manifests` → CRD YAML files updated in `deploy/`
- **Reqs:** KTTM-NFR-005

#### F0.5 — Universal Local Task Master (Makefile)
- Bridges Antigravity IDE straight to cluster lifecycle tasks.
- Targets:
  - `make test-unit`: fast in-memory linter and schema graph tests.
  - `make test-integration`: real-time controller runtime integration validations.
  - `make build`: Compiles standalone local workstation Go binary (M2 ARM64).
- **User test:** Run `make test-unit` → all tests pass quickly in local memory.
- **Reqs:** KTTM-NFR-003

#### F0.6 — Developer Documentation & Onboarding
- `README.md`: project overview, quick-start (3 commands), architecture diagram link
- `CONTRIBUTING.md`: fork → feature branch → PR workflow, lint rules, commit message format
- `docs/local-dev.md`: step-by-step k3d setup with troubleshooting
- `docs/architecture.md`: links to `specs/design.md` with ASCII architecture
- **User test:** New engineer follows README → running cluster in < 10 minutes
- **Reqs:** KTTM-NFR-005

#### F0.7 — Helm Chart Skeleton (Dev Profile)
- `deploy/helm/kttm/` with Chart.yaml, values.yaml, templates/
- Dev profile: single-node, no HA, embedded NATS + BuntDB + Zot
- `make helm-install` deploys KTTM to local k3d cluster in < 2 minutes
- `make helm-uninstall` cleans up cleanly
- **User test:** `make helm-install` → `kubectl get pods -n kttm-system` → all pods Running
- **Reqs:** KTTM-REQ-037, KTTM-REQ-038, KTTM-REQ-044

---

**M0 Requirements met:** NFR-001, NFR-003, NFR-004, NFR-005, REQ-037 (partial), REQ-038 (partial), REQ-044 (partial)

---

## Phase 1 — Core Operator & First Real Pipeline
**Theme:** A real S3 → transform → Postgres pipeline runs end-to-end via `KttmApp` CRD.  
**Milestone:** M1 · ETL MVP

### Features

#### F1.1 — KttmApp CRD Registration & Operator Boot
- `KttmApp` CRD registered in cluster: `kubectl apply -f deploy/crds/kttmapp.yaml`
- Operator starts, watches `KttmApp` resources, logs reconcile events
- `kubectl get kttmapps -A` works
- `kubectl describe kttmapp my-app` shows Status conditions
- **User test:** `kubectl apply -f examples/hello-world.yaml` → operator logs show "Reconciled KttmApp hello-world"
- **Reqs:** REQ-007, REQ-032

#### F1.2 — GraphLinter: Static DAG Validation
- Operator calls GraphLinter before any deployment
- Checks: DAG cycle detection (DFS), disconnected nodes, missing SecretRef names
- On error: `KttmApp.status.phase = Linting`, status condition `LintError=True` with message
- On success: `LintPassed=True`, proceed to deploy
- **User test:** Apply a CRD with a cycle in edges → `kubectl describe` shows lint error, Argo Workflow NOT created
- **Reqs:** REQ-025

#### F1.3 — Argo Workflow Compilation & Execution
- Operator compiles `KttmApp.spec.workflowDag` → `Workflow` CRD and applies it
- Single-node DAGs work end-to-end
- Multi-node linear DAGs work (A → B → C)
- Operator sets `ownerReference` on Argo Workflow (cascade delete)
- `KttmApp.status.argoWorkflowName` filled after deploy
- **User test:** Apply `examples/s3-ledger-pipeline.yaml` → Argo UI at `http://localhost:8080/argo` shows workflow running
- **Reqs:** REQ-007, REQ-023, REQ-043

#### F1.4 — S3 Connector (Source)
- Real implementation: connects to MinIO (local) or AWS S3
- Reads files matching `bucket` + `prefix` params
- Streams bytes to NATS envelope → emptyDir handoff
- SecretRef for AWS credentials (never in CRD)
- **User test:** Upload a CSV to MinIO → trigger pipeline → file bytes arrive at next node
- **Reqs:** REQ-031, NFR-002

#### F1.5 — PostgreSQL Connector (Sink)
- Real implementation using `pgx` + COPY protocol for bulk insert
- Reads from NATS envelope + emptyDir volume
- Writes rows to target table
- SecretRef for DB credentials
- **User test:** Run pipeline → `SELECT COUNT(*) FROM ledger` in Postgres → row count matches CSV
- **Reqs:** REQ-031, NFR-002

#### F1.6 — Webhook Trigger Connector
- HTTP endpoint exposed via KTTM BFF: `POST /api/trigger/{app-name}`
- HMAC-SHA256 signature verification (optional, configurable)
- Triggers `KttmApp` execution via NATS `kttm.apps.{name}.trigger`
- Returns `202 Accepted` with run ID
- **User test:** `curl -X POST http://localhost:8080/api/trigger/my-app` → pipeline starts in Argo
- **Reqs:** REQ-023

#### F1.7 — NATS JetStream Dual-Channel Transport
- NATS JetStream stream: `KTTM_APPS` (all workflow traffic)
- Metadata envelope: `{envelopeId, mimeType, payloadBytes, storageRef}` on NATS
- Raw bytes: `emptyDir` volume for same-pod, MinIO for cross-pod
- Routing logic: < 100MB → shm, < 1GB → emptyDir, > 1GB or fan-out → MinIO
- **User test:** Send a 500MB CSV through pipeline → NATS messages are < 500 bytes each, file transferred via emptyDir
- **Reqs:** REQ-031

#### F1.8 — KttmApp Status & Phase Tracking
- Operator updates `status.phase` at each lifecycle stage: `Pending → Linting → Building → Deploying → Running → Succeeded/Failed`
- Per-node status in `status.nodeStatuses`
- Kubernetes Events emitted per phase transition
- **User test:** Watch `kubectl get kttmapp -w` → phase transitions visible in real time as pipeline runs
- **Reqs:** REQ-032

#### F1.9 — `kttm` CLI: run, validate
- `kttm validate my-app.yaml` → runs GraphLinter offline (no cluster needed), prints errors
- `kttm run my-app.yaml` → applies CRD to cluster, streams logs, exits with code 0/1
- `kttm status my-app` → prints current phase + node statuses
- **User test:** `kttm validate examples/s3-ledger-pipeline.yaml` → "✓ DAG valid, 3 nodes, 2 edges"
- **Reqs:** REQ-025

---

**M1 Requirements met (cumulative):** NFR-001 to 005, REQ-007, REQ-023, REQ-025, REQ-031, REQ-032, REQ-034 (partial), REQ-035 (partial), REQ-043

---

## Phase 2 — Scripting Engine & More Connectors
**Theme:** Developers write custom Python/Bash scripts that run inside secure containers and process real data.  
**Milestone:** M1 · ETL MVP (completing)

### Features

#### F2.1 — Python Script Node (Real Container Execution)
- `type: script/python` in DAG spec
- Operator uses NodeBuilderFactory → BuildKit → builds real container image from inline script
- `params.packages` → installs pip packages at build time (air-gap safe)
- Container reads from stdin (emptyDir), writes to stdout (emptyDir)
- **User test:** Write a pandas transformation script in `spec.script` → run pipeline → Postgres shows transformed rows
- **Reqs:** REQ-012, REQ-024, REQ-041

#### F2.2 — Bash Script Node
- `type: script/bash` with Alpine base image
- Same stdin/stdout stdio contract as Python
- **User test:** Bash script that filters CSV rows runs inside cluster, real output verified
- **Reqs:** REQ-012, REQ-024

#### F2.3 — JavaScript (Node.js) Script Node
- `type: script/javascript` with Node 22 Alpine base
- npm package installation from `params.packages`
- **User test:** JS script that reformats dates in JSON runs inside cluster
- **Reqs:** REQ-012, REQ-024

#### F2.4 — Kafka Consumer Trigger
- Real Kafka connection: SASL_SSL + plaintext
- Subscribes to topic, each message triggers a workflow execution
- Configurable batch size and consumer group
- **User test:** Publish 10 messages to Kafka topic → 10 Argo Workflows start in cluster
- **Reqs:** REQ-023

#### F2.5 — NATS Consumer Trigger
- Subscribe to NATS subject, trigger workflow per message
- **User test:** `nats pub kttm.test "hello"` → workflow fires
- **Reqs:** REQ-023

#### F2.6 — Retry Policy & Fault Tolerance
- `spec.nodes[].retryPolicy.maxRetries` + `backoff: fixed | exponential | linear`
- Failed node → Argo retries → status reflects retry count
- `retryOn: error | transientError`
- **User test:** Node fails 2x then succeeds on 3rd try → pipeline succeeds with retry events in status
- **Reqs:** REQ-035

#### F2.7 — Resource Limits & Requests per Node
- `spec.nodes[].resources.requests/limits` maps to pod spec
- Operator validates limits are set (warning if missing)
- **User test:** Node with `limits.memory: 100Mi` processing 200MB file → OOMKill detected, retry triggered
- **Reqs:** REQ-034

#### F2.8 — Parallel Fan-Out + Barrier Aggregation
- `spec.parallelGroups[].nodes` → compiled to Argo `dag.tasks` (parallel execution)
- `barrierAfter: true` → Argo waits for ALL parallel nodes before continuing
- **User test:** Pipeline splits into 3 parallel sinks, all must finish before final reporting node runs
- **Reqs:** REQ-023

#### F2.9 — OpenTelemetry Traces & Prometheus Metrics
- OTel SDK in operator + BFF: traces exported to Jaeger/Tempo
- 6 Prometheus metrics: NodeDuration, NodeMemoryRatio, NodeCPUThrottleRatio, EnvelopePayloadBytes, NATSQueueDepth, WorkflowExecutions
- Prometheus scrape endpoint: `/metrics` on operator + BFF
- **User test:** Run pipeline → Prometheus at `http://localhost:9090` shows node metrics, Jaeger shows traces
- **Reqs:** REQ-027

---

**M1 Requirements met (cumulative):** + REQ-012, REQ-024, REQ-027, REQ-034, REQ-035, REQ-041

---

## Phase 3 — Visual DAG Canvas (Developer SDK)
**Theme:** Developers drag nodes onto a canvas, see real-time execution state, without writing YAML.  
**Milestone:** M2 · Visual Flow Benchmark

### Features

#### F3.1 — XYFlow DAG Canvas (Live Editor)
- React + `@xyflow/react` (ReactFlow v12) visual node graph
- Drag-and-drop nodes from sidebar connector catalog onto canvas
- Draw edges by connecting node ports (output → input)
- Canvas state syncs to `KttmApp` YAML in real time (bidirectional)
- **User test:** Drag S3 → Python → Postgres nodes, draw edges → YAML panel shows valid KttmApp spec
- **Reqs:** REQ-009, REQ-011, REQ-023

#### F3.2 — Node Config Sidebar (Schema-Driven Forms)
- Click any node → right sidebar shows a JSON Schema-driven config form
- Form fields rendered from `Connector.Schema()` (real connector schema, not hardcoded)
- Changes in form update `KttmApp` spec live
- SecretRef field: dropdown showing real cluster secrets
- **User test:** Click S3 node → form shows bucket, prefix, region fields → fill in → YAML updates
- **Reqs:** REQ-009, REQ-011

#### F3.3 — Real-Time Node Execution State on Canvas
- Nodes glow/pulse with status colors: grey (idle) → blue (running) → green (succeeded) → red (failed)
- Updates via NATS WebSocket → browser: `kttm.apps.{name}.nodes.{id}.status`
- **User test:** Trigger pipeline from canvas → watch nodes light up in sequence as pipeline runs
- **Reqs:** REQ-032

#### F3.4 — Node Grouping UI
- Multi-select nodes → "Group into shared pod" → sets same `groupId`
- Grouped nodes rendered with shared container border on canvas
- **User test:** Group 2 nodes → run pipeline → Argo shows 1 step (not 2) for grouped nodes
- **Reqs:** REQ-010

#### F3.5 — Connector Catalog with Live Search
- Left sidebar: searchable, filterable grid of all registered connectors
- Categories: Trigger, Storage, Database, Messaging, Script, Logic, Sink
- Click connector → added to canvas at cursor position
- **User test:** Type "kafka" in search → Kafka trigger card appears → click → node appears on canvas
- **Reqs:** REQ-009, REQ-018

#### F3.6 — YAML / JSON Schema Tab (Live Preview)
- Tab in developer SDK: raw YAML view of the current `KttmApp` spec
- Edit YAML directly → canvas updates (round-trip sync)
- Syntax highlighting via CodeMirror
- **User test:** Edit YAML to change a param → canvas node config sidebar reflects change immediately
- **Reqs:** REQ-011

#### F3.7 — Run & Stop Controls on Canvas
- "Trigger" button: sends `POST /api/trigger/{app}` → starts Argo Workflow
- "Stop" button: sends delete to Argo Workflow → terminates running pipeline
- Live progress bar showing % nodes completed
- **User test:** Click Trigger → pipeline starts → click Stop → Argo Workflow terminated
- **Reqs:** REQ-032

---

**M2 Requirements met (cumulative):** + REQ-009, REQ-010, REQ-011, REQ-018

---

## Phase 4 — AI Advisor & Developer Observability
**Theme:** The system automatically diagnoses problems and explains them in plain English.  
**Milestone:** M2 · Visual Flow Benchmark (completing)

### Features

#### F4.1 — Rule-Based Pre-Deploy Advisor
- GraphLinter extended with resource/performance rules:
  - Missing resource limits → WARNING
  - Memory limit < payload size estimate → ERROR
  - No retry policy on network nodes → WARNING
- Results displayed in canvas sidebar before trigger
- **User test:** Add S3 node with no memory limit → advisor shows warning before you can run
- **Reqs:** REQ-025, REQ-026

#### F4.2 — Runtime AI Advisor (Rule Engine)
- Prometheus metrics polled every 60s by advisor goroutine
- Rules engine: memory > 90% → CRITICAL, error rate > 20% → CRITICAL, CPU throttle > 80% → WARNING
- Advice published to NATS `kttm.advisor.{ns}.{app}.advice`
- BFF subscribes → WebSocket → browser toast notification + Advisor panel card
- **User test:** Node uses > 90% memory → orange advice card appears in Advisor panel without page refresh
- **Reqs:** REQ-026

#### F4.3 — LLM-Assisted Advisor (Optional)
- When Prometheus threshold breached → builds structured prompt
- Sends to Google Gemini API (if API key configured in Helm values)
- Falls back to rule-only text if no API key / offline
- LLM response shows in Advisor panel with "AI Generated" badge
- **User test:** Configure Gemini API key → trigger memory spike → plain-English recommendation appears in panel
- **Reqs:** REQ-026

#### F4.4 — Execution History & Audit Trail
- Run history per `KttmApp`: list of past Argo Workflow runs with phase + duration
- `GET /api/apps/{name}/history` returns last 50 runs
- UI: History tab showing sortable table of runs
- Click run → drill into per-node timing + errors
- **User test:** After 3 pipeline runs → History tab shows 3 rows with correct phases and durations
- **Reqs:** REQ-032

#### F4.5 — Node-Level Log Streaming
- BFF WebSocket: `ws://kttm/ws/apps/{name}/nodes/{nodeId}/logs`
- Streams Argo step pod logs in real time to the canvas "Debug" drawer
- No polling — pushed via NATS `kttm.apps.{name}.nodes.{id}.log`
- **User test:** Running node → click node → log drawer opens → live log lines appear as they're generated
- **Reqs:** REQ-014 (partial), REQ-032

---

**M2 Requirements met (cumulative):** + REQ-026, REQ-027 (extended), REQ-028 (partial)

---

## Phase 5 — No-Code Form Builder & End-User Portal
**Theme:** A business user fills a form and triggers a real pipeline — no code, no Kubernetes knowledge.  
**Milestone:** M3 · No-Code App Builder

### Features

#### F5.1 — Visual Form Builder (Drag & Drop)
- Developer SDK: "UI Builder" tab next to canvas
- Drag form components: Text, Number, Date, Select, File Upload, Toggle, Textarea
- Place into a layout grid
- Each component binds to a workflow node input field
- Output: JSON Schema saved to `KttmApp.spec.uiLayout.schema` (via ConfigMap)
- **User test:** Drag 3 fields → "Preview as End-User" → form renders correctly with real fields
- **Reqs:** REQ-015

#### F5.2 — End-User Portal (Schema-Driven, No Rebuild)
- SPA reads `uiLayout.schema` from ConfigMap at runtime
- Renders form: field types, labels, required validation, placeholder text
- No frontend rebuild required when schema changes
- **User test:** Developer changes form field label in builder → end-user refreshes browser → new label appears (no deployment)
- **Reqs:** REQ-001, REQ-004, REQ-017

#### F5.3 — Form Submission → Workflow Trigger
- Submit button: validates required fields → sends payload to `kttm.apps.{name}.trigger` via NATS
- NATS message includes form values as the workflow's input parameters
- Success/error feedback to end-user
- **User test:** End-user fills form → clicks Submit → Argo Workflow starts with form values as node inputs
- **Reqs:** REQ-004, REQ-015

#### F5.4 — Real-Time Dashboard Renderer
- `sink/ui-state` connector pushes processed results back to the SPA
- SPA subscribes via WebSocket: `kttm.apps.{name}.ui.state`
- Dashboard renders: tables, numbers, status indicators, text
- **User test:** Pipeline finishes → end-user's browser shows result table without page refresh
- **Reqs:** REQ-004

#### F5.5 — File Upload via Form
- Form field type: `file`
- File uploaded to MinIO (internal S3) via BFF `/api/upload`
- StorageRef written to workflow input envelope
- Pipeline processes the uploaded file
- **User test:** End-user uploads Excel file via form → Python node reads it → processed data written to Postgres
- **Reqs:** REQ-004, REQ-015, REQ-031

#### F5.6 — Custom React Bundle Injection
- Developers upload a custom React component bundle (Webpack Module Federation remote)
- Stored in ConfigMap `kttm-ui-{app-name}`
- BFF serves bundle from `GET /ui-bundles/{app-name}/remoteEntry.js`
- SPA loads it via dynamic `import()` — no restart
- **User test:** Upload a custom chart component → end-user portal shows the custom chart
- **Reqs:** REQ-016

#### F5.7 — Cron / Scheduled Trigger
- `type: trigger/cron` with `params.schedule: "0 8 * * 1-5"` (cron expression)
- Operator creates a Kubernetes `CronJob` or Argo `CronWorkflow`
- Workflow runs automatically at scheduled time
- **User test:** Set schedule for "every minute" → watch 5 Argo Workflows start automatically
- **Reqs:** REQ-023

---

**M3 Requirements met (cumulative):** + REQ-001, REQ-004, REQ-015, REQ-016, REQ-017

---

## Phase 6 — More Connectors: Database Sinks & File Watchers
**Theme:** KTTM can talk to every major database and react to file system events.  
**Milestone:** M3 · No-Code App Builder (extending connector library)

### Features

#### F6.1 — MySQL Sink Connector
- Real `go-sql-driver/mysql` implementation
- Bulk insert via `LOAD DATA INFILE` or multi-value INSERT
- **User test:** Pipeline writes 10,000 rows → verify in MySQL client
- **Reqs:** REQ-009 (connector matrix)

#### F6.2 — MongoDB Sink Connector
- `mongo-go-driver` with `InsertMany` for bulk writes
- Connection string from SecretRef
- **User test:** Pipeline writes documents → `db.collection.find({})` shows results
- **Reqs:** REQ-009

#### F6.3 — Redis Connector (Source + Sink)
- Read: `LRANGE`, `HGETALL`, `GET` operations
- Write: `SET`, `LPUSH`, `HSET` operations
- **User test:** Pipeline reads from Redis list → transforms → writes back to Redis hash
- **Reqs:** REQ-009

#### F6.4 — ClickHouse Sink Connector
- `clickhouse-go` driver with native TCP protocol
- Bulk insert via `clickhouse-go` batch API
- **User test:** Pipeline writes 1M rows → ClickHouse query verifies count in < 1s
- **Reqs:** REQ-009

#### F6.5 — File Watcher Trigger (S3 Events)
- Polls MinIO/S3 for new objects matching prefix pattern
- Configurable polling interval (default 30s)
- Triggers workflow per new file with file ref as input
- **User test:** Drop CSV into MinIO bucket → pipeline starts automatically within 30 seconds
- **Reqs:** REQ-023

#### F6.6 — Email Trigger (IMAP)
- IMAP listener polling inbox for new emails
- Triggers workflow with email subject, body, attachment ref as inputs
- Configurable filter: sender, subject regex
- **User test:** Send email to configured inbox → pipeline fires with email metadata
- **Reqs:** REQ-023

#### F6.7 — WebSocket Live Trigger
- BFF maintains WebSocket server: `ws://kttm/ws/trigger/{app}`
- Each message received → workflow triggered with message payload
- **User test:** `wscat -c ws://localhost:8080/ws/trigger/my-app` → send message → Argo Workflow starts
- **Reqs:** REQ-023

#### F6.8 — `kttm export` & `kttm import` CLI
- `kttm export my-app --output bundle.tar.gz` → creates scrubbed bundle (no secrets)
- `kttm export my-app --include-images --output bundle.tar.gz` → embeds Docker images
- `kttm import bundle.tar.gz --registry registry.local:5000` → loads into air-gapped cluster
- **User test:** Export app → copy tarball to a machine with no internet → import and run successfully
- **Reqs:** REQ-020, REQ-021, REQ-033, REQ-042, NFR-002

---

**M3 Requirements met (cumulative):** + REQ-020, REQ-021, REQ-033, REQ-042

---

## Phase 7 — Security, RBAC & Compliance
**Theme:** Enterprises can trust KTTM in regulated environments — zero secrets in code, full audit trail.  
**Milestone:** M4 · Enterprise Security

### Features

#### F7.1 — Polymorphic Permission-Driven UI
- Full `PermissionGate` enforcement: 12 atomic permissions gate every UI section
- `usePermissions()` hook queries BFF → K8s SubjectAccessReview → permission array
- No permission → component not rendered + API call returns 403
- **User test:** Log in as "enduser" → canvas tab missing → try to POST to `/api/apps/create` → 403
- **Reqs:** REQ-001, REQ-002, REQ-003, REQ-004

#### F7.2 — Admin Console: RBAC Matrix UI
- Admin sees all users + their current permission sets
- Checkbox matrix: assign/revoke any of the 12 atomic permissions per user
- Changes applied to K8s RBAC (creates/updates ClusterRoleBindings)
- **User test:** Admin checks "app:debug" for a user → user can see Debug tab without relogin
- **Reqs:** REQ-003, REQ-005, REQ-039

#### F7.3 — K8s RBAC SubjectAccessReview Integration
- BFF queries K8s `SubjectAccessReview` API for each KTTM permission
- Falls back to embedded default role table if K8s RBAC not configured
- OIDC/JWT token from browser → BFF → `TokenReview` API
- **User test:** Revoke `app:create` from user in K8s → user logs out and back in → no Create button
- **Reqs:** REQ-039

#### F7.4 — Secret Isolation: SecretRef-Only Architecture
- All connector configs use `secretRef: k8s-secret-name` — no inline values
- `kttm export` scrubs: strips params matching password/token/key/cert patterns
- Integration test: export CRD, grep for any secret value → must find none
- **User test:** Export app bundle → inspect YAML → no passwords/tokens/keys visible
- **Reqs:** NFR-002

#### F7.5 — Trivy CVE Scanner Integration
- NodeBuilderFactory triggers Trivy scan after every image build
- CRITICAL CVEs → block deployment, status condition `ScanFailed=True`
- HIGH CVEs → WARNING in Advisor panel, deployment proceeds
- CVE report available at `GET /api/apps/{name}/scan-report`
- **User test:** Build an image with a known CVE → deployment blocked, admin console shows CVE report
- **Reqs:** REQ-046

#### F7.6 — Kyverno Policy Engine
- Kyverno deployed as part of Helm chart (toggleable)
- Default policies baked in: no privileged containers, no root, resource limits required, no latest tag
- `KttmApp.spec.policy.policyRefs` links to additional cluster policies
- Operator runs `OPAGate.Validate()` before every Argo Workflow creation
- **User test:** Apply a CRD with a privileged container → Kyverno blocks it, status shows policy violation
- **Reqs:** REQ-036

#### F7.7 — Immutable Audit Log
- Every API call with auth context logged to NATS `kttm.audit.{namespace}`
- Format: `{timestamp, principal, action, resource, result, ip}`
- NATS JetStream with `LIMITS` retention policy (immutable, max 90 days)
- Admin Console: Audit Log tab shows last 1000 events, searchable by user/action
- **User test:** Perform 5 actions as admin → Audit Log tab shows all 5 with correct timestamps and principals
- **Reqs:** REQ-039 (extended)

#### F7.8 — CIS Kubernetes Benchmark Kyverno Policies
- Helm chart includes `kyverno-policies.yaml` with CIS L1 + L2 rules:
  - No privileged containers, no `hostPID/hostNetwork`, seccompProfile required
  - `runAsNonRoot: true`, read-only root filesystem where possible
- `kttm check-compliance` CLI command: runs audit against all policies
- **User test:** `kttm check-compliance` → reports pass/fail for each CIS control
- **Reqs:** REQ-047

#### F7.9 — Admin Lifecycle Controls (Install/Upgrade/Uninstall)
- Admin Console: Lifecycle tab with real Helm install/upgrade/uninstall
- BFF: `POST /api/infra/install`, `POST /api/infra/upgrade`, `POST /api/infra/uninstall`
- Rolling upgrade: zero-downtime (operator leader election, BFF rolling update)
- **User test:** Click Upgrade in Admin Console → operator updates from v0.1 to v0.2 with no downtime
- **Reqs:** REQ-032, REQ-033

#### F7.10 — Bundle Import / Air-Gap Loader
- Admin Console: Bundles tab with file drop zone
- Upload `bundle.tar.gz` → BFF unpacks → images loaded into Zot registry → CRD applied
- Progress shown in real time
- **User test:** Import a bundle on a machine with no internet access → pipeline runs successfully
- **Reqs:** REQ-020, REQ-021, REQ-042

---

**M4 Requirements met (cumulative):** + REQ-001–006, REQ-020, REQ-021, REQ-033, REQ-036, REQ-039, REQ-042, REQ-046, REQ-047, NFR-002

---

## Phase 8 — Auto-Scaling & High Availability
**Theme:** Workflows scale to zero overnight and to 100 pods under load. Production cluster never goes down.  
**Milestone:** M5 · Platform Scale

### Features

#### F8.1 — KEDA Integration: Scale-to-Zero
- Operator auto-creates KEDA `ScaledObject` for each `KttmApp` with `execution.scaling.type: keda`
- NATS JetStream scaler: scales based on consumer lag
- `minReplicas: 0` → pods disappear when queue is empty
- `maxReplicas` from CRD spec
- **User test:** Send 1000 messages to NATS queue → watch `kubectl get pods` scale from 0 → 50 → 0
- **Reqs:** REQ-029

#### F8.2 — In-Place Vertical Pod Autoscaling
- Operator monitors node `metrics.cpu` and `metrics.mem` via Prometheus
- When mem > 80% sustained for 5 minutes → patch Pod resource requests/limits (no eviction)
- Requires K8s 1.27+ feature gate `InPlacePodVerticalScaling=true`
- **User test:** Node under memory pressure → `kubectl describe pod` shows updated limits without restart
- **Reqs:** REQ-030

#### F8.3 — Operator HA: Leader Election
- Deploy operator with 3 replicas + `leaderElection: true`
- Kill the leader pod → another takes over within 15 seconds, no reconcile loop lost
- **User test:** `kubectl delete pod kttm-operator-0` → new leader elected, workflows continue
- **Reqs:** REQ-044

#### F8.4 — BFF HA: Horizontal Pod Autoscaler
- Deploy BFF with HPA: scale on CPU > 70%
- Session-less: any BFF pod can handle any request
- NATS WebSocket connections load-balanced via sticky session (or re-connect on failover)
- **User test:** Load test BFF at 1000 req/s → HPA scales to multiple pods, no errors
- **Reqs:** REQ-044

#### F8.5 — NATS JetStream Clustering (3-node)
- Helm chart deploys NATS in clustered mode (3 replicas, quorum = 2)
- JetStream persistence: `emptyDir` (dev) → `PVC` (prod)
- Kill 1 NATS node → system continues operating without message loss
- **User test:** Kill NATS pod mid-pipeline → pipeline completes successfully, no messages lost
- **Reqs:** REQ-044

#### F8.6 — Zot Registry Clustering
- Zot OCI registry in HA mode: 2 replicas + shared PVC
- Image push/pull works when 1 replica is down
- **User test:** Kill Zot replica during image build → push succeeds via remaining replica
- **Reqs:** REQ-044

#### F8.7 — Multi-Cluster GitOps Replication
- After every reconcile: operator pushes scrubbed YAML to Git (existing F7.4 gitops bridge)
- Argo CD in downstream cluster pulls from Git branch
- Promote app from staging to production: `git merge staging → production`
- **User test:** Deploy app to staging cluster → merge to production branch → Argo CD applies to prod cluster automatically
- **Reqs:** REQ-008

---

**M5 Requirements met (cumulative):** + REQ-008, REQ-029, REQ-030, REQ-044

---

## Phase 9 — Embedded Algorithm Engine (Layer 10)
**Theme:** Data transformation using 33 built-in algorithm implementations — no external dependencies.  
**Milestone:** M5 · Platform Scale (completing)

### Features

#### F9.1 — Algorithm Engine Core Dispatcher
- `AlgorithmEngine.Dispatch(algName, input io.Reader) io.Writer`
- Node type: `algorithm/{name}` in `workflowDag`
- Input: streaming JSON/CSV from envelope; output: transformed stream
- **User test:** DAG node `algorithm/sort-merge` → input unsorted CSV → output sorted CSV
- **Reqs:** Layer 10 (ALG-001–033)

#### F9.2 — Level 1–4 Algorithms (Pure Go, Embedded)
- Math basics: GCD, Fibonacci, Prime sieve, Palindrome
- Arrays & Sort: Merge, Quick, Heap, Binary Search, Sliding Window
- String algorithms: KMP, Aho-Corasick, Rabin-Karp, Boyer-Moore
- DP: Knapsack, LCS, LIS, Edit Distance
- All as Go packages: `core/algorithms/level{1..4}/`
- **User test:** Send unstructured data through `algorithm/kmp-search` → returns match positions
- **Reqs:** ALG-001–015

#### F9.3 — Level 5–9 Algorithms (Pure Go, Embedded)
- Graph: BFS, DFS, Dijkstra, Bellman-Ford, Floyd-Warshall, Prim, Kruskal
- Greedy: Huffman Coding, Activity Selection
- Backtracking: N-Queens, Sudoku solver
- Advanced trees: Segment Tree, Fenwick Tree, B-Tree, LSM Tree
- Compression: LZW, RLE, Huffman
- **User test:** `algorithm/huffman-compress` node → compresses CSV → decompresses → identical bytes
- **Reqs:** ALG-016–021

#### F9.4 — Level 10: ML Kernels (Containerized, Optional)
- Containerized as separate images (heavy deps: numpy, sklearn, torch)
- Algorithms: Linear/Logistic Regression, Decision Tree, Random Forest, CNN, LSTM
- Consensus: Raft simulation, Two-Phase Commit, Saga Orchestration
- Node type: `algorithm/ml/{name}` → spawns a separate container
- **User test:** `algorithm/ml/linear-regression` node trains model on CSV data, outputs predictions
- **Reqs:** ALG-029–033

---

**M5 Requirements met (cumulative):** + Layer 10 ALG-001–033

---

## Phase 10 — CNCF Ecosystem Integration & Identity
**Theme:** KTTM is a first-class CNCF citizen. Every workload has a cryptographic identity.  
**Milestone:** M6 · CNCF Sandbox

### Features

#### F10.1 — SPIFFE/SPIRE Workload Identity
- SPIRE Server deployed via Helm (toggleable)
- Every workflow pod gets a SPIFFE SVID (X.509 cert) via SPIRE Agent
- Connector auth uses SVID-based mTLS instead of static tokens where possible
- **User test:** `kubectl exec` into a workflow pod → `/run/spiffe/certs/svid.pem` exists
- **Reqs:** REQ-036 (extended), REQ-039 (extended)

#### F10.2 — OPA Gatekeeper: Extended Policy Suite
- OPA Gatekeeper as alternative to Kyverno (toggle in Helm)
- `ConstraintTemplate` CRDs shipped with Helm chart
- Enterprise policies: mandatory cost labels, workload identity required, no internet egress
- **User test:** Deploy CRD without required cost labels → OPA blocks it
- **Reqs:** REQ-036

#### F10.3 — OpenCost Integration: Per-Node Cost Tracking
- OpenCost deployed via Helm (toggleable)
- BFF queries OpenCost API: `GET /model/allocation?...` with node label selectors
- Admin Console → Cost tab shows real $ per workflow node
- **User test:** Run pipeline → Cost tab shows real $/day breakdown per node (not demo data)
- **Reqs:** REQ-028

#### F10.4 — KTTM Connector Hub API
- Public REST API: `GET https://hub.kttm.io/connectors?q=kafka` 
- `kttm search kafka` → searches Hub
- `kttm publish connector ./my-connector/` → validates + builds + pushes to Hub
- Connector discovery chain: built-in → in-cluster CRDs → Hub
- **User test:** Publish a custom connector → search for it → `kttm install connector my-connector@v1.0` works
- **Reqs:** REQ-018, REQ-019

#### F10.5 — Artifact Hub Listing
- `artifacthub.io` metadata in `artifacthub.io/metadata.yaml`
- KTTM Helm chart listed on Artifact Hub with install button
- `kttm` CLI published to Homebrew tap
- **User test:** Search "kttm" on Artifact Hub → chart appears, install instructions work
- **Reqs:** Marketplace goals

#### F10.6 — Kubernetes Conformance & CRD Versioning
- CRD version migration webhook: `v1alpha1` → `v1alpha2` → `v1beta1`
- Conversion webhook preserves backward compat
- Test: apply old `v1alpha1` CRD → webhook auto-converts to `v1beta1`
- **User test:** Deploy old app YAML → cluster accepts it, warns about deprecation, runs correctly
- **Reqs:** NFR-004

#### F10.7 — Governance & Community Files
- `GOVERNANCE.md`, `MAINTAINERS.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`
- CNCF Project Charter document
- `ADOPTERS.md` tracking production users
- GitHub: Issue templates (bug, feature, security), PR template with checklist
- **User test:** New contributor can find all governance docs from the README
- **Reqs:** CNCF Sandbox application criteria

---

**M6 Requirements met (cumulative):** + REQ-018, REQ-019, REQ-028, REQ-036 (SPIFFE), REQ-039 (OIDC+SVID), NFR-004 (CRD versioning)

---

## Phase 11 — NIS2 / CRA Compliance & Enterprise Hardening
**Theme:** KTTM passes European enterprise compliance audits out of the box.  
**Milestone:** M6 · CNCF Sandbox (security arm)

### Features

#### F11.1 — SBOM Generation (Trivy + CycloneDX)
- Every image build → Trivy generates SBOM in CycloneDX JSON format
- Stored in Zot OCI registry as image annotation
- Downloadable from Admin Console: `GET /api/apps/{name}/sbom`
- **User test:** Build image → download SBOM → open in `sbom-tool` → all dependencies listed
- **Reqs:** REQ-047 (CRA compliance)

#### F11.2 — NIS2 Incident Response Workflow
- Pre-built `KttmApp` template: NIS2 Incident Response pipeline
- Trigger: KTTM itself emitting a CRITICAL advisor alert
- Pipeline: log incident → notify admin email → post to Slack → create Jira ticket
- **User test:** Trigger a CRITICAL memory alert → Jira ticket created, Slack message sent
- **Reqs:** REQ-047

#### F11.3 — Data Residency Controls
- `KttmApp.spec.dataResidency.regions: ["eu-west-1"]` 
- Operator adds pod affinity rules to restrict execution to labelled nodes
- MinIO bucket policy: `aws:RequestedRegion` condition
- **User test:** Set `dataResidency: eu-only` → pipeline pods only schedule on EU-labelled nodes
- **Reqs:** REQ-047 (GDPR data residency)

#### F11.4 — Secrets Management: HashiCorp Vault Integration
- BFF integrates with Vault Agent Injector
- `secretRef: vault://secret/kttm/db-creds` → sidecar injects secret as env var
- No Kubernetes Secret created (Vault is the source of truth)
- **User test:** Set Vault path in CRD → run pipeline → connector uses Vault-injected credentials
- **Reqs:** NFR-002 (extended)

---

**M6 Requirements met (cumulative):** + REQ-047 (full coverage), NFR-002 (Vault extension)

---

## Phase 12 — Marketplace & One-Click Install
**Theme:** Any developer on AWS/GCP/Azure can deploy KTTM in one click.  
**Milestone:** M7 · Marketplace & GA

### Features

#### F12.1 — AWS Marketplace (EKS)
- Helm chart packaged for AWS Marketplace
- `eksctl` addon: `eksctl enable addon --name kttm`
- CloudFormation template for infrastructure pre-requisites
- **User test:** Log into AWS Marketplace → search KTTM → one-click deploy to EKS → working in 5 min
- **Reqs:** Marketplace goals

#### F12.2 — Google Cloud Marketplace (GKE)
- GKE App Deployer image
- `gcloud marketplace solutions deploy kttm`
- **User test:** GCP Marketplace → one-click deploy to GKE → working
- **Reqs:** Marketplace goals

#### F12.3 — Azure Marketplace (AKS)
- Azure Application offering
- `az aks deploy kttm`
- **User test:** Azure Marketplace → one-click deploy to AKS → working
- **Reqs:** Marketplace goals

#### F12.4 — Open-Core Licensing Split
- Community Edition: all features in this plan (Apache 2.0)
- Enterprise Edition feature flags: `sso_oidc`, `vault_integration`, `audit_export`, `multi_tenant`
- License check in operator: CE = these flags off by default
- **User test:** Deploy CE → try to enable SSO → operator log: "Enterprise feature, contact sales"
- **Reqs:** Open-core monetization goals

#### F12.5 — Discord / Slack Community
- Public Discord server with channels: #help, #announcements, #showcase, #contributing
- Bot: GitHub issue notifications, release announcements
- **User test:** New user joins Discord → bot welcome message → question answered in < 24h

---

## Phase 13 — Edge & IoT Profile
**Theme:** KTTM runs on a Raspberry Pi with GPIO sensors.  
**Milestone:** M7 · Marketplace & GA (edge vertical)

### Features

#### F13.1 — Edge Single-Daemon Mode
- `kttm install --profile edge` → all components in 1 pod, ~60MB RAM total
- Embedded NATS (in-process), BuntDB, no Argo (use direct goroutine scheduler)
- ARM64 binary: `GOARCH=arm64 go build`
- **User test:** Deploy to Raspberry Pi 4 → KTTM runs, processes GPIO sensor data
- **Reqs:** REQ-044 (edge topology), NFR-001

#### F13.2 — GPIO Hardware Connector
- `trigger/gpio` connector: reads from Raspberry Pi GPIO pins
- Uses `go-rpio` library
- `params.pin: 18, params.edge: rising` → triggers workflow on signal
- **User test:** Press a button wired to GPIO 18 → pipeline fires → data logged to Postgres
- **Reqs:** KTTM-CON-006

#### F13.3 — MQTT IoT Connector
- Full MQTT v3.1.1/v5 support via `paho.mqtt.golang`
- Subscribe to topic → trigger workflow per message
- Works offline (local MQTT broker), air-gapped
- **User test:** MQTT sensor publishes temperature reading → pipeline triggers → alert sent if > 90°C
- **Reqs:** REQ-023

---

## Requirements Coverage Matrix

### By Milestone

| Milestone | Functional Reqs Met | NFRs Met | Notes |
|---|---|---|---|
| M0 · Dev Foundation | 2 | NFR-001,003,004,005 | Infrastructure only |
| M1 · ETL MVP | REQ-007,023,025,031,032,034,035,043 | All 5 NFRs | First real pipeline |
| M2 · Visual Flow Benchmark | + REQ-009,010,011,018,026,027 | All 5 | Canvas + Advisor |
| M3 · No-Code App Builder | + REQ-001,004,015,016,017,020,021,033,042 | All 5 | End-user forms |
| M4 · Enterprise Security | + REQ-001–006,036,039,046,047 | All 5 + NFR-002 | RBAC, CVE, Kyverno |
| M5 · Platform Scale | + REQ-008,029,030,044, Layer 10 | All 5 | HA, KEDA, algorithms |
| M6 · CNCF Sandbox | + REQ-018,019,028,036(SPIFFE),047(full) | All 5 + CRD versioning | CNCF governance |
| M7 · GA | All remaining | All 5 | Marketplace, edge |

### Uncovered / Deferred (re-evaluate in Sprint Planning)

| Requirement | Why deferred | Target Milestone |
|---|---|---|
| REQ-013 Composite connector composability | Complex UX → awaiting design feedback | M3 |
| REQ-022 Execution environment sandboxing | gVisor / kata research needed | M4 |
| REQ-037 Embedded storage (full) | BuntDB covers dev; Postgres embed complex | M5 |
| REQ-040 HA Control Plane Module | Covered by operator leader election | M5 |
| REQ-041 Node builder (full BuildKit wiring) | Scaffold complete, full production wiring | M3 |
| REQ-045 Web app always-active mode | Design tension with scale-to-zero | M5 |

---

## Agile Conventions

### Sprint Cadence
- **Sprint length:** 2 weeks
- **Sprint planning:** Pick vertical features from upcoming Phase by priority + dependency order
- **Definition of Done:** Feature works with a real user (no mocking), CI passes, tests added, docs updated
- **Scope changes:** New requirements added to appropriate Phase, milestone dates re-evaluated

### Feature Priority Labels
- `P0 — Blocker:` Nothing else works without this (e.g., F0.2 CI, F1.3 Argo compilation)
- `P1 — Core:` Main path for the milestone goal
- `P2 — Enhancement:` Adds value but milestone deliverable works without it
- `P3 — Stretch:` Nice to have this sprint, carry to next if needed

### Current Sprint Priorities (Phase 0 → Phase 1 bootstrap)
1. `P0` F0.1 — k3d cluster bootstrap
2. `P0` F0.2 — GitHub Actions CI
3. `P0` F1.1 — KttmApp CRD + operator boot
4. `P1` F1.2 — GraphLinter
5. `P1` F1.3 — Argo compilation
6. `P1` F1.4 + F1.5 — S3 + Postgres connectors
7. `P2` F0.4 — Go module structure
8. `P2` F1.8 — Status tracking
9. `P3` F0.6 — Documentation

---

## Tech Debt & Risk Register

| Risk | Impact | Mitigation |
|---|---|---|
| BuildKit rootless in K3d | High | Test on both OrbStack and Rancher Desktop; fallback to Kaniko |
| InPlacePodVerticalScaling K8s 1.27 feature gate | Medium | Detect at runtime; graceful degradation to pod restart |
| NATS WebSocket behind nginx in production | Medium | Configure `upstream_keepalive`, test with `nginx` controller |
| Trivy offline DB size (>100MB) | Medium | Optional `--skip-trivy` flag for edge profile |
| Argo Workflows CRD version drift | Low | Pin Argo version in go.mod + Helm chart, tested in CI |
| Module Federation browser compat | Low | Target Chrome 100+, Firefox 100+, Safari 16+ |

---

*This plan is a living document. Update feature status here before each sprint. When requirements change, add new features to the relevant phase and re-assess milestone dates.*
