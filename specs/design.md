# KTTM Architecture Design Document
**Project:** KTTM — *Kattum* (கட்டும்) — "That which builds, structures, or binds together"  
**Classification:** CNCF Sandbox Target · Kubernetes-Native · Air-Gapped Ready  
**Coverage:** All 47 functional requirements (REQ-001–047), 5 NFRs, Layer 9 connector matrix, Layer 10 algorithm engine  

---

## 1. System Overview

KTTM is a **unified full-stack platform** that binds three disconnected paradigms into one executable Kubernetes CRD:

| Paradigm | What it replaces |
|---|---|
| **ETL / Data Pipelines** | Apache NiFi, Kestra, Airflow, Apache Camel |
| **Real-Time App Integration** | n8n, Node-RED, Zapier Enterprise |
| **No-Code UI / Forms** | Retool, Bubble, Appsmith |

All three paradigms run as a single `KttmApp` Kubernetes CRD, compiled by one Go operator, executed by Argo Workflows, and observed via a unified OpenTelemetry stack.

---

## 2. Core Architectural Planes

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│ PLANE 1: USER ACCESS LAYER (REQ-001–006)                                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────────────────────────┐ │
│  │  Developer SDK  │  │  Admin Console  │  │  End-User Portal (app:execute)   │ │
│  │  (all perms     │  │  (rbac:manage   │  │  (forms, dashboards only)        │ │
│  │   except rbac)  │  │   no-canvas)    │  │                                  │ │
│  └────────┬────────┘  └────────┬────────┘  └──────────────┬───────────────────┘ │
│           │                   │                           │                      │
│        Polymorphic Rendering Engine (permission-driven, single SPA bundle)       │
└──────────────────────────────┬───────────────────────────────────────────────────┘
                               │
┌──────────────────────────────▼───────────────────────────────────────────────────┐
│ PLANE 2: CONTROL PLANE (REQ-007–024, REQ-040–045)                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌───────────────────┐    │
│  │  Go Operator │  │  GraphLinter │  │ Node Builder │  │  GitOps Bridge    │    │
│  │  (KttmApp    │  │  (DFS+type   │  │  Factory     │  │  (Argo CD / Flux) │    │
│  │   CRD ctrl)  │  │   checker)   │  │  (REQ-041)   │  │  (REQ-008)        │    │
│  └──────────────┘  └──────────────┘  └──────────────┘  └───────────────────┘    │
└──────────────────────────────┬───────────────────────────────────────────────────┘
                               │
┌──────────────────────────────▼───────────────────────────────────────────────────┐
│ PLANE 3: EXECUTION PLANE (REQ-023, REQ-029–036, REQ-043, REQ-045)                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌──────────────────────┐   │
│  │ Argo        │  │ NATS        │  │ KEDA        │  │  Connector Runtime   │   │
│  │ Workflows   │  │ JetStream   │  │ Auto-Scaler │  │  (60+ node types)    │   │
│  └─────────────┘  └─────────────┘  └─────────────┘  └──────────────────────┘   │
└──────────────────────────────┬───────────────────────────────────────────────────┘
                               │
┌──────────────────────────────▼───────────────────────────────────────────────────┐
│ PLANE 4: TELEMETRY & ADVISOR PLANE (REQ-025–028)                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌───────────────────┐    │
│  │  GraphLinter │  │  OTel Traces │  │  AI Advisor  │  │  OpenCost         │    │
│  │  (pre-deploy)│  │  Prometheus  │  │  (LLM+rules) │  │  (cost per node)  │    │
│  └──────────────┘  └──────────────┘  └──────────────┘  └───────────────────┘    │
└──────────────────────────────┬───────────────────────────────────────────────────┘
                               │
┌──────────────────────────────▼───────────────────────────────────────────────────┐
│ PLANE 5: INFRASTRUCTURE PLANE (REQ-037–039, REQ-046–047)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌───────────────────┐    │
│  │  Embedded    │  │  Embedded    │  │  Kyverno /   │  │  Trivy Scanner    │    │
│  │  Registry    │  │  SQLite DB   │  │  OPA Policy  │  │  CIS Benchmark    │    │
│  │  (k3d/Zot)   │  │  (BuntDB)   │  │  Engine      │  │  Compliance       │    │
│  └──────────────┘  └──────────────┘  └──────────────┘  └───────────────────┘    │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. The `KttmApp` CRD Schema

Every KTTM application is stored as a single `KttmApp` Kubernetes Custom Resource (REQ-007).

```yaml
apiVersion: kttm.io/v1alpha1
kind: KttmApp
metadata:
  name: my-etl-pipeline
  namespace: kttm-apps
spec:
  # ── Identity ───────────────────────────────────────────────────────────────
  displayName: "Daily Ledger ETL"
  version: "1.3.0"
  mode: workflow           # workflow | webapp | hybrid (REQ-045)
  
  # ── RBAC ───────────────────────────────────────────────────────────────────
  rbac:
    roles:
      - name: developer
        permissions: ["app:create", "app:modify", "app:debug", "app:export"]
      - name: admin
        permissions: ["rbac:manage", "infra:install", "infra:upgrade", "bundle:import"]
      - name: enduser
        permissions: ["app:execute"]
    serviceAccountRef: kttm-executor

  # ── UI Layout Schema (Tier 1 End-Users) ────────────────────────────────────
  uiLayout:
    type: form              # form | dashboard | hybrid
    schema: |               # JSON Schema rendered at runtime (REQ-001, REQ-015)
      { "fields": [...], "actions": [...] }
    customReactBundle:      # Optional custom React bundle (REQ-016)
      configMapRef: my-custom-ui-bundle

  # ── Workflow DAG ────────────────────────────────────────────────────────────
  workflowDag:
    nodes:
      - id: source-s3
        type: connector/s3
        label: "Read Daily Ledger"
        groupId: ingress-group    # Node grouping (REQ-010)
        secretRef: aws-creds
        params:
          bucket: acme-lake
          prefix: "ledgers/{{date}}"
        resources:
          requests: { cpu: "100m", memory: "128Mi" }
          limits:   { cpu: "500m", memory: "256Mi" }
        retryPolicy:              # REQ-035
          maxRetries: 3
          backoff: exponential
        outputs: [transform-python]
      
      - id: transform-python
        type: script/python
        label: "Filter & Normalize"
        image: python:3.12-slim
        script: |
          import sys, pandas as pd
          df = pd.read_csv(sys.stdin, chunksize=5000)  # micro-batch REQ-031
          for chunk in df:
              chunk.dropna(subset=['phone']).to_csv(sys.stdout, index=False)
        resources:
          limits: { cpu: "1000m", memory: "512Mi" }
        outputs: [sink-postgres]

      - id: sink-postgres
        type: connector/postgres
        secretRef: db-creds
        params:
          query: "COPY ledger FROM STDIN (FORMAT CSV)"

    # Parallel fan-out + barrier pattern (REQ-023)
    parallelGroups:
      - id: parallel-reports
        nodes: [report-pdf, report-slack, report-db]
        barrierAfter: true   # Wait for all three before continuing
    
    edges:
      - from: source-s3
        to: transform-python
      - from: transform-python
        to: parallel-reports

  # ── Execution Configuration ────────────────────────────────────────────────
  execution:
    backend: argo            # argo | tekton (strategy pattern)
    scaling:
      type: keda             # REQ-029 zero-to-N
      natsSubject: "kttm.apps.my-etl.queue"
      minReplicas: 0
      maxReplicas: 100
    verticalResize: inplace  # REQ-030 - requires K8s 1.27+ feature gate
    
  # ── GitOps ────────────────────────────────────────────────────────────────
  gitOps:                    # REQ-008
    repo: git@github.com:acme/kttm-apps.git
    branch: main
    path: apps/etl/
    sshKeySecretRef: gitops-key

  # ── Breakpoint Debug (REQ-014) ─────────────────────────────────────────────
  debug:
    enabled: false           # Set true in staging/dev
    breakpoints: ["transform-python", "sink-postgres"]
    streamLogs: true
```

---

## 4. Component Architecture

### 4.1 Polymorphic UI Rendering Engine (REQ-001–006)

**Design decision:** Single SPA bundle with permission-driven slot masking.

```
┌─────────────────────────────────────────────────────────┐
│               Single React SPA (one bundle)              │
│                                                         │
│  usePermissions() hook ──► JWT / K8s RBAC query         │
│           │                                             │
│           ▼                                             │
│  PermissionGate component masks slots:                  │
│  ┌──────────────┬──────────────┬────────────────────┐   │
│  │ Developer    │ Admin        │ EndUser             │   │
│  │ ✓ Canvas     │ ✗ Canvas     │ ✗ Canvas            │   │
│  │ ✓ Debug      │ ✓ Audit Log  │ ✗ Debug             │   │
│  │ ✓ Node CRUD  │ ✓ Lifecycle  │ ✓ Forms/Dashboards  │   │
│  │ ✓ iFrame     │ ✓ RBAC Mgmt │ ✗ Infrastructure    │   │
│  │   preview    │ ✓ Cost View  │                     │   │
│  └──────────────┴──────────────┴────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

**Trade-off: Single SPA vs. separate app per role**  
→ **Single SPA wins.** Security boundary enforced server-side via K8s RBAC. The UI is a presentation layer only — even if client-side masking is bypassed, API calls will be rejected. Avoids maintaining three separate codebases.

**Permission model:**

| Permission | Developer | Admin | End-User |
|---|---|---|---|
| `app:create` | ✓ | ✗ | ✗ |
| `app:modify` | ✓ | ✗ | ✗ |
| `app:debug` | ✓ | ✗ | ✗ |
| `app:export` | ✓ | ✓ | ✗ |
| `app:execute` | ✓ | ✓ | ✓ |
| `rbac:manage` | ✗ | ✓ | ✗ |
| `infra:install` | ✗ | ✓ | ✗ |
| `infra:upgrade` | ✗ | ✓ | ✗ |
| `bundle:import` | ✗ | ✓ | ✗ |
| `audit:view` | ✗ | ✓ | ✗ |
| `cost:view` | ✗ | ✓ | ✗ |

Custom roles combine any subset of the above (REQ-005) via an Admin checkbox matrix UI.

---

### 4.2 Go Operator — The Control Plane Core (REQ-007, REQ-032, REQ-043)

The operator is the brain of KTTM — a standard `controller-runtime` reconciler.

```
Reconcile Loop (KttmApp):
┌───────────────────────────────────────────────────────┐
│ 1. Fetch KttmApp CRD from etcd                        │
│ 2. GraphLinter.Lint(spec.workflowDag)                 │
│    ├─ DFS cycle detection (REQ-025)                   │
│    ├─ Type compatibility check (output→input)         │
│    ├─ SecretRef resolution check                      │
│    ├─ Resource limits validation                      │
│    └─ BLOCK deployment on any error                   │
│ 3. NodeBuilder.Compile(spec) → container images       │
│    └─ For script/python: BuildKit sandbox (REQ-041)  │
│ 4. TrivyScanner.Scan(images) → CVE audit (REQ-046)   │
│    └─ Block on CRITICAL severity                      │
│ 5. OPAGate.Validate(podSpec) → policy check (REQ-036)│
│ 6. ArgoCompiler.Compile(dag) → ArgoWorkflow CRD       │
│    └─ Parallel groups → dag.tasks (parallelSteps)    │
│ 7. ServerSideApply(argoWorkflow, ownerRef)            │
│ 8. KEDAScaler.Reconcile(scaling spec) (REQ-029)       │
│ 9. GitOps.Push(yaml snapshot) (REQ-008)               │
│ 10. Status.Update(phase, conditions, nodeStatuses)    │
└───────────────────────────────────────────────────────┘
```

**Memory target:** < 100MB idle (KTTM-NFR-001).  
**Implementation:** Pure Go, no reflection-heavy frameworks. Controller compiled with `GOGC=off` + tuned `GOMEMLIMIT`. Uses `sigs.k8s.io/controller-runtime` v0.18.

---

### 4.3 DAG Compilation Strategy — Argo vs. Tekton (REQ-043)

**Trade-off: Argo Workflows vs. Tekton**

| Criterion | Argo Workflows | Tekton |
|---|---|---|
| Native DAG support | ✓ first-class | ✗ requires Pipelines chaining |
| Parallel fan-out | ✓ `parallelSteps` | ✓ via `parallel` |
| UI / Argo UI | ✓ built-in | ✗ requires Tekton Dashboard |
| Artifact passing | ✓ first-class | Partial (via workspaces) |
| Air-gapped support | ✓ fully offline | ✓ |
| Memory footprint | ~80MB controller | ~120MB controller |

**→ Default: Argo Workflows.** Tekton supported via Strategy Pattern — `Backend` interface allows swap without CRD changes (REQ-038).

```go
type Backend interface {
    Compile(dag *DAGSpec) (runtime.Object, error)
    GetStatus(ctx context.Context, name, ns string) (Phase, error)
}

// Registered backends:
// - ArgoBackend  (default)
// - TektonBackend (opt-in via KttmApp.spec.execution.backend=tekton)
```

---

### 4.4 Node Grouping & Worker Packaging Engine (REQ-010, REQ-024)

**Design:** Nodes marked with the same `groupId` compile into a single Argo step with a shared pod. Ungrouped nodes get isolated pods.

```
GroupId "ingress-group": [source-s3, validate-schema]
  → Compiled as ONE Argo step, ONE pod, ONE container
  → Nodes communicate via in-process channels (not NATS)
  → Eliminates inter-pod latency for tight chains

No groupId: transform-python, sink-postgres
  → Each compiled as separate Argo step + separate pod
  → Communicates via NATS envelope + emptyDir/shm volume
```

**Packaging modes** (configurable per node/group):
- `pod` — default isolated Kubernetes pod
- `sidecar` — sidecar container in the Argo step pod
- `init` — init container (one-shot setup)
- `binary` — embedded Go binary in the operator image (ultra-lightweight)
- `crd` — custom Kubernetes CRD (for stateful long-running services)

---

### 4.5 Dual-Channel Binary Preservation Stream (REQ-031)

This is the **zero-normalization data backbone** — the core data transport guarantee.

```
Node A ─────► NATS JetStream ─────► Node B
              (tiny Envelope)
              ~200 bytes JSON
              {
                "envelopeId": "uuid",
                "mimeType": "application/x-parquet",
                "payloadBytes": 2147483648,
                "storageRef": {
                  "type": "emptyDir",       // or "shm" or "minio"
                  "path": "/data/payload-uuid.parquet"
                }
              }

              Raw bytes ──────────────────► emptyDir mount
              (GB-scale binary,            (same pod) or
               never touches NATS)         MinIO (cross-pod)
```

**Storage routing logic:**

| Payload size | Same pod | Different pods | Decision |
|---|---|---|---|
| < 100MB | ✓ | — | `/dev/shm` (RAM, zero syscall) |
| < 1GB | ✓ | — | `emptyDir` (disk, no network) |
| Any size | — | ✓ | MinIO (internal S3) |
| Any | Fan-out (1→N) | ✓ | MinIO (shared reference) |

---

### 4.6 Polyglot Script Engine (REQ-012, REQ-024)

**Design:** Each code block node compiles into a language-specific container image via BuildKit.

```
KttmApp spec (script/python):
  image: kttm-base/python:3.12
  script: "..." (inline code)
  ▼
NodeBuilderFactory (REQ-041):
  Dockerfile = base image + COPY script + ENTRYPOINT
  ▼
BuildKit daemon (in-cluster, rootless, air-gap safe)
  ▼
Zot OCI Registry (internal, REQ-037)
  ▼
Argo Workflow step pulls from Zot
```

**Supported language runtimes:**

| Language | Base image | Package manager |
|---|---|---|
| Python 3.x | `python:3.12-slim` | pip / uv |
| JavaScript (Node) | `node:22-alpine` | npm |
| Bash | `alpine:3.20` | apk |
| R | `r-base:4.4` | CRAN |
| Go binary | `scratch` | pre-built binary |
| Java | `eclipse-temurin:21-jre-alpine` | Maven/Gradle artifacts baked in |

---

### 4.7 Interactive Debugger (REQ-014)

**Design:** Debug mode injects a `kttm-debug-sidecar` into each Argo step pod. The sidecar exposes a WebSocket stream consumed by the Developer SDK UI.

```
Developer SDK UI
    │  (WebSocket /ws/debug/{runId}/{nodeId})
    ▼
BFF API Server
    │  (gRPC stream to debug sidecar)
    ▼
kttm-debug-sidecar (per pod)
    │
    ├─ Pre-breakpoint: capture stdin bytes before forwarding
    ├─ At breakpoint: PAUSE signal → hold stdin pipe
    ├─ Stream to UI: raw payload hex dump + metadata
    └─ On "continue": release pipe to main container
```

The sidecar is only injected when `KttmApp.spec.debug.enabled: true`. Zero overhead in production.

---

### 4.8 No-Restart Pluggable UI Extensions (REQ-017)

**Design:** UI plugins are stored as `KttmUIPlugin` CRDs. The SPA's module federation runtime fetches new plugin manifests via NATS `kttm.ui.plugins.updated` events and dynamically imports the module.

```
Admin uploads plugin bundle
    ▼
kubectl apply KttmUIPlugin CRD
    ▼
Operator stores bundle in Zot OCI registry
    ▼
Operator publishes to NATS: kttm.ui.plugins.updated
    ▼
All active Developer SDK sessions receive event
    ▼
React Module Federation → dynamic import() of new plugin URL
    ▼
New panel/screen appears instantly, no page reload, no restart
```

---

### 4.9 Auto-Scaling Architecture (REQ-029, REQ-030)

**Horizontal (KEDA):**

```yaml
# KEDA ScaledObject — auto-generated by operator
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: kttm-transform-scaler
spec:
  scaleTargetRef:
    name: kttm-transform-deployment
  minReplicaCount: 0      # Scale to zero!
  maxReplicaCount: 100
  triggers:
    - type: nats-jetstream
      metadata:
        natsServerMonitoringEndpoint: nats.kttm:8222
        streamName: KTTM_APPS
        consumer: transform-consumer
        lagThreshold: "50"    # Scale up when 50+ messages pending
```

**Vertical in-place (REQ-030):**

```go
// Operator patches the Pod directly — no eviction, no restart
// Requires K8s 1.27+ with InPlacePodVerticalScaling=true
patch := &corev1.Pod{
    Spec: corev1.PodSpec{
        Containers: []corev1.Container{{
            Name: "worker",
            Resources: corev1.ResourceRequirements{
                Requests: newRequests,
                Limits:   newLimits,
            },
        }},
    },
}
// Kernel adjusts Linux cgroups v2 without SIGTERM
```

---

### 4.10 AI Performance Diagnostics (REQ-026)

**Two-phase design (same as FlowEngine advisor):**

**Phase 1 — Rule-based (air-gap safe, synchronous):**
- Runs before deployment (DFS + type check)
- Memory > 90% → CRITICAL block
- Error rate > 20% → CRITICAL alert
- CPU throttle > 80% → WARNING

**Phase 2 — LLM-assisted (async, optional):**
- Polls Prometheus every 60s
- Threshold breach → builds structured prompt
- Sends to: Google Gemini (default) → local Ollama (air-gap fallback)
- Response → NATS `kttm.{ns}.advisor.{nodeId}.advice` → toast modal in UI

**Trade-off: Cloud LLM vs. bundled LLM:**
- Cloud LLM (Gemini): Best quality, requires outbound network
- Bundled Ollama + Llama-3.2-3B-Q4: ~2GB image, works fully air-gapped, lower quality
- **→ Hybrid: Auto-detect network, fall back to Ollama if offline**

---

### 4.11 GitOps Bridge (REQ-008)

**Design:** The operator commits a canonical YAML snapshot after every successful reconcile.

```go
// Triggered post-reconcile
func (g *GitOpsbridge) Push(app *kttmv1.KttmApp) error {
    // 1. Render the KttmApp CRD as clean YAML (secrets scrubbed)
    yaml := renderScrubbed(app)      // NFR-002: strip all secrets
    // 2. git pull (rebase) → write file → git add → git commit
    commitMsg := fmt.Sprintf("kttm: auto-sync %s@%s", app.Name, app.ResourceVersion)
    // 3. git push (SSH key from SecretRef)
    // Regional clusters (Argo CD / Flux) pull and apply automatically
}
```

**GitOps backend support:**

| Backend | Protocol | Notes |
|---|---|---|
| Argo CD | GitOps push | Default — visual deployment history |
| Flux CD | GitOps push | Alternative — lighter footprint |
| Raw Git | SSH push | For clusters without GitOps operators |

---

### 4.12 Air-Gapped Export System (REQ-020, REQ-021, REQ-033, REQ-042)

**Bundle structure (`kttm export my-app --output bundle.tar.gz`):**

```
bundle.tar.gz
├── manifest.yaml          (KttmApp CRD, secrets scrubbed)
├── bundle-info.json       (version, checksum, created-at)
├── ui/
│   └── schema.json        (UI layout schema)
├── connectors/
│   └── *.json             (custom connector definitions)
├── images/                (only with --include-images flag)
│   ├── transform-python.tar   (docker save output)
│   └── sink-postgres.tar
└── algorithms/            (embedded algorithm library binaries)
    └── kttm-alg-engine.tar
```

**Import pipeline (`kttm import bundle.tar.gz --registry registry.local:5000`):**
1. Extract tarball
2. For each image: `docker load` → `docker tag` → `docker push` to private registry
3. Rewrite `manifest.yaml` image references to `registry.local:5000/*`
4. `kubectl apply -f manifest.yaml`
5. Operator loads bundle into embedded Zot registry

---

### 4.13 Embedded Infrastructure Components (REQ-037, REQ-038)

**Modular toggle design:** Every component can be swapped for an existing cluster installation.

```yaml
# Helm values.yaml — infrastructure toggles
kttm:
  embedded:
    registry:
      enabled: true          # false → point to existing Harbor/Quay
      type: zot              # zot (default, <50MB) | docker-registry
      url: ""                # set if enabled: false
    
    database:
      enabled: true          # false → point to existing PostgreSQL
      type: buntdb           # buntdb embedded (default) | sqlite | etcd
    
    nats:
      enabled: true          # false → point to existing NATS cluster
      url: ""
    
    argo:
      enabled: true          # false → use existing Argo Workflows
    
    trivy:
      enabled: true          # false → disable CVE scanning
      url: ""
    
    opencost:
      enabled: false         # true → enable cost tracking
      url: ""
    
    opa:
      enabled: true          # false → no policy enforcement
      type: kyverno          # kyverno | opa-gatekeeper
```

---

### 4.14 Connector Registry (REQ-009, REQ-018, REQ-019)

**Connector discovery chain:**

```
Developer SDK → "Find connector for Kafka"
    │
    ▼
ConnectorRegistry.Search("kafka")
    │
    ├─ 1. Built-in connectors (compiled into operator binary)
    ├─ 2. In-cluster KttmConnector CRDs (custom/imported)
    ├─ 3. Private Git registry (configured URL + auth)
    └─ 4. Public KTTM Hub (kttm.io/connectors, if online)
    
Result: ConnectorManifest {
    type:      "connector/kafka",
    schema:    JSON Schema (rendered as config form in UI),
    image:     "kttm/kafka-connector:v1.4.2",
    inputTypes:  ["application/x-kafka-records"],
    outputTypes: ["application/octet-stream", "application/json"],
}
```

**Publishing (`kttm publish connector ./my-connector/`):**
```
Validate connector schema
    ▼
Build OCI image via BuildKit
    ▼
Tag + push to target:
  - Local Zot registry (default)
  - Private corporate registry (--registry flag)
  - KTTM Hub (--public flag, requires auth token)
    ▼
Create KttmConnector CRD in cluster
    ▼
ConnectorRegistry hot-reloads (NATS event, no restart)
```

---

## 5. NATS Subject Hierarchy

```
kttm.
├── apps.
│   └── {appName}.
│       ├── trigger           → Launch workflow
│       ├── status            → Phase updates (running/succeeded/failed)
│       ├── nodes.
│       │   └── {nodeId}.
│       │       ├── envelope  → Metadata envelope (200 bytes)
│       │       ├── log       → Streaming step logs
│       │       └── debug     → Breakpoint data streams
│       └── ui.
│           └── state         → Reactive UI state updates
├── advisor.
│   └── {namespace}.{appName}.
│       ├── metrics           → Prometheus snapshot
│       └── advice            → AI recommendation payload
├── infra.
│   ├── plugins.updated       → UI plugin hot-reload trigger
│   ├── connectors.updated    → Registry refresh trigger
│   └── lifecycle.            → Install/upgrade/uninstall events
└── audit.
    └── {namespace}           → Immutable audit event log
```

---

## 6. Security Architecture (REQ-036, REQ-039, REQ-046, REQ-047, NFR-002)

### 6.1 RBAC Alignment (REQ-039)

```
User JWT/OIDC token
    ▼
KTTM BFF API Server
    ▼
TokenReview API (Kubernetes native)
    ▼
SubjectAccessReview for each permission check
    ▼
Map K8s ClusterRole → KTTM permission array
    ▼
Fallback to embedded default roles if K8s RBAC missing
```

### 6.2 Secret Isolation (NFR-002)

All secrets are injected via Kubernetes Secrets or HashiCorp Vault at runtime — **never stored in the KttmApp CRD**:

```go
// CRD stores only a reference
type ConnectorNode struct {
    SecretRef string `json:"secretRef"` // → K8s Secret name or Vault path
    // ...no actual secret values
}

// At reconcile time: inject as env vars into pod
envFrom:
  - secretRef:
      name: {{ .SecretRef }}
```

### 6.3 Compliance (REQ-047)

| Standard | Implementation |
|---|---|
| CIS Kubernetes Benchmark | Kyverno policies baked into Helm chart |
| Cyber Resilience Act (CRA) | Trivy SBOM generation per image build |
| NIS2 Directive | Audit log to NATS `kttm.audit.*` + immutable log sink |
| SOC 2 Type II | All API calls logged with principal + timestamp |

### 6.4 Image Scanning (REQ-046)

```
NodeBuilderFactory builds image
    ▼
TrivyScanner.Scan(imageRef)
    ▼
CVE Report generated (SARIF format)
    ▼
if CRITICAL CVEs found:
    BlockDeployment()
    PublishAlert(admin console)
else if HIGH CVEs:
    WarnAndProceed()
    PublishAdvisory()
```

---

## 7. Connector Matrix — Full Taxonomy

### 7.1 Trigger Connectors (Source / Ingress)

| ID | Name | Protocol | Implementation |
|---|---|---|---|
| `trigger/ui-form` | User UI Input | NATS JetStream | Built-in |
| `trigger/cron` | Scheduled / Cron | Go `cron/v3` | Built-in |
| `trigger/file-watch` | File Activity Monitor | inotify / S3 Events | Built-in |
| `trigger/websocket` | WebSocket Listener | gorilla/websocket | Built-in |
| `trigger/webhook` | HTTP Hook Server | net/http | Built-in |
| `trigger/cdc` | Database CDC | Debezium / pglogical | Sidecar |
| `trigger/email` | Email Event Trigger | IMAP/POP3 | Built-in |
| `trigger/kafka` | Kafka Consumer | confluent-kafka-go | Built-in |
| `trigger/mqtt` | MQTT IoT | paho.mqtt.golang | Built-in |
| `trigger/nats` | NATS Subscriber | nats.go | Built-in |
| `trigger/grpc` | gRPC Server | google.golang.org/grpc | Built-in |
| `trigger/sms` | SMS Gateway | Twilio/AWS SNS | Plugin |
| `trigger/whatsapp` | WhatsApp Business | Meta Cloud API | Plugin |
| `trigger/gpio` | Hardware GPIO | go-rpio (edge) | Plugin |

### 7.2 Logic / Transform Connectors

| ID | Name | Implementation |
|---|---|---|
| `logic/function` | Core Math/String Functions | Built-in Go |
| `logic/table` | Tabular Data Factory | Apache Arrow (Go) |
| `script/python` | Python Sandbox | Container (python:slim) |
| `script/javascript` | Node.js Sandbox | Container (node:alpine) |
| `script/bash` | Bash Sandbox | Container (alpine) |
| `script/r` | R Analytics | Container (r-base) |
| `script/go` | Go Binary | Compiled static binary |
| `logic/query` | Database Query Runner | pgx/go-sql-driver |
| `logic/aggregate` | Message Aggregator | In-memory batch buffer |
| `logic/formula` | Formula Engine | go-expr |
| `logic/branch` | Graph Branch Filter | CEL expressions |
| `logic/fanout` | Parallel Fan-Out | Argo `parallelSteps` |
| `logic/barrier` | Barrier Aggregator | Argo `dag` sync |
| `logic/databricks` | Databricks (Arrow) | Arrow Flight SQL |
| `logic/spark-livy` | Spark / Livy | REST API |
| `logic/knime-table` | Line-Stream CSV Parser | Built-in |
| `logic/sql-pushdown` | SQL Pushdown Optimizer | sqlparser-go |
| `logic/ml-sklearn` | Classical ML | Python container + sklearn |
| `logic/llm` | LLM AI Agent | LangChain / MCP |

### 7.3 Terminator / Sink Connectors

| ID | Name | Protocol |
|---|---|---|
| `sink/postgres` | PostgreSQL | pgx + COPY protocol |
| `sink/mysql` | MySQL | go-sql-driver |
| `sink/clickhouse` | ClickHouse | clickhouse-go |
| `sink/snowflake` | Snowflake | gosnowflake |
| `sink/elasticsearch` | Elasticsearch | elastic/go-elasticsearch |
| `sink/mongodb` | MongoDB | mongo-go-driver |
| `sink/s3` | S3 / GCS / MinIO | aws-sdk-go-v2 |
| `sink/nfs` | NFS / SMB Mount | OS mount |
| `sink/pv` | Kubernetes PV | volume mount |
| `sink/kafka` | Kafka Producer | confluent-kafka-go |
| `sink/nats` | NATS Publisher | nats.go |
| `sink/rabbitmq` | RabbitMQ | amqp091-go |
| `sink/ui-state` | UI State Renderer | NATS → WebSocket → React |
| `sink/report-pdf` | PDF Report | go-wkhtmltopdf / Tika |
| `sink/report-excel` | Excel Report | excelize |
| `sink/slack` | Slack Alert | slack-go |
| `sink/email` | Email Alert | net/smtp |
| `sink/teams` | MS Teams Alert | Adaptive Cards |
| `sink/discord` | Discord Alert | discordgo |
| `sink/salesforce` | Salesforce | REST API |
| `sink/jira` | Jira | REST API |
| `sink/monday` | Monday.com | GraphQL API |
| `sink/office365` | Office 365 | Microsoft Graph API |
| `sink/google-workspace` | Google Workspace | Google APIs |

---

## 8. Embedded Algorithm Engine (Layer 10)

The algorithm library is compiled directly into the KTTM operator binary as Go packages. No external process spawning — all algorithms execute in-memory with streaming input.

### Architecture

```
KttmApp node type: algorithm/sort-merge
    ▼
AlgorithmEngine.Dispatch("sort-merge", input io.Reader) io.Writer
    ▼
kttm/algorithms/
├── level1/  (math, sort, string, recursion)
├── level2/  (linked-list, hashmap, heap, trees)
├── level3/  (graph, BFS/DFS, shortest-path, MST)
├── level4/  (dynamic programming, bitmask DP)
├── level5/  (greedy, backtracking, divide-conquer)
├── level6/  (advanced trees, string matching)
├── level7/  (number theory, geometry, probabilistic)
├── level8/  (parallel, distributed clocks, MapReduce)
├── level9/  (DB simulation, network routing, OS schedulers)
└── level10/ (cloud LB, ML kernels, consensus - Raft/Paxos)
```

### Algorithm Catalog Summary

| Level | Category | Algorithms (KTTM-ALG-001–033) |
|---|---|---|
| 1 | Math Basics | Factorial, Prime, Fibonacci, GCD, LCM, Palindrome |
| 1 | Complexity | Big O / Omega / Theta analyzer |
| 1 | Arrays & Sort | Bubble, Merge, Quick, Binary Search, Sliding Window |
| 1 | Strings | KMP, Palindrome, Anagram, Substring |
| 2 | Data Structures | Linked Lists, Min/Max Heap, AVL Tree, Red-Black Tree |
| 2 | Expression Eval | Stack, Queue, Infix→Postfix |
| 3 | Graphs | BFS, DFS, Union-Find, Topological Sort (Kahn) |
| 3 | Shortest Path | Dijkstra, Bellman-Ford, Floyd-Warshall |
| 3 | MST | Prim, Kruskal |
| 4 | DP | Knapsack, LCS, LIS, Edit Distance, Matrix Chain |
| 4 | Advanced DP | Tree DP, Graph DP, Bitmask DP, Digit DP |
| 5–7 | Greedy | Activity Selection, Huffman Coding |
| 5–7 | Backtracking | N-Queens, Sudoku, Maze |
| 5–7 | Divide & Conquer | Strassen, Karatsuba |
| 8–9 | Advanced | Segment Trees, Fenwick, B-Tree, B+ Tree, LSM |
| 8–9 | String | KMP, Rabin-Karp, Boyer-Moore, Aho-Corasick |
| 8–9 | Compression | LZW, Huffman, RLE |
| 10 | Parallel | Parallel Merge Sort, MapReduce, Lamport/Vector Clocks |
| 10 | DB Simulation | WAL, MVCC, Nested-Loop/Hash/Merge Joins |
| 10 | Cloud LB | Consistent Hashing, Token Bucket, Erasure Coding |
| 10 | ML | Gradient Descent, Decision Trees, Random Forest, CNN/RNN |
| 10 | Consensus | Raft, Paxos, Two-Phase Commit, Saga Orchestration |
| 10 | Optimization | A*, Genetic Algorithms, FFT, Simplex |

---

## 9. Deployment Architecture (REQ-044)

### 9.1 Local Development (MacBook / k3d)

```bash
# k3d-kttm-config.yaml (from spec, annotated)
apiVersion: k3d.io/v1alpha4
kind: Simple
metadata:
  name: kttm-engine
servers: 1
agents: 3
image: rancher/k3s:v1.30.2-k3s1   # ARM64 M2 optimized
ports:
  - port: 8080:80
    nodeFilters: [loadbalancer]
options:
  k3s:
    extraArgs:
      - arg: --disable=traefik       # KTTM uses nginx-ingress
        nodeFilters: [server:0]
      - arg: --disable=servicelb
        nodeFilters: [server:0]
      - arg: --kube-apiserver-arg=feature-gates=InPlacePodVerticalScaling=true
        nodeFilters: [server:0]     # REQ-030: vertical resize without restart
```

**Memory footprint (idle, local):**

| Component | RAM |
|---|---|
| kttm-operator | ~35MB |
| kttm-bff | ~25MB |
| nats | ~10MB |
| zot registry | ~20MB |
| buntdb (embedded) | ~5MB |
| **Total** | **~95MB** ✓ (< NFR-001 100MB cap) |

### 9.2 Production HA Cluster

```
┌─────────────────────────────────────────────────────┐
│ Namespace: kttm-system                               │
│                                                     │
│  kttm-operator (3 replicas, leader election)         │
│  kttm-bff      (3 replicas, HPA)                     │
│  nats          (3-node JetStream cluster)            │
│  zot-registry  (2 replicas + shared PV)              │
│  argo-server   (2 replicas)                          │
│                                                     │
│ Namespace: kttm-apps                                 │
│  Workflow pods (zero-to-N via KEDA)                  │
└─────────────────────────────────────────────────────┘
```

### 9.3 Edge / Air-Gap Single Node

```bash
# Single-daemon mode
kttm install --profile edge --offline-bundle edge-bundle.tar.gz
# All components run in one process (operator + BFF + embedded NATS)
# Total RAM: ~60MB
```

---

## 10. Key Technology Decisions Summary

| Decision | Choice | Rejected | Reason |
|---|---|---|---|
| Workflow engine | **Argo Workflows** | Tekton | DAG-native, better UI, lower memory |
| Messaging | **NATS JetStream** | Kafka, RabbitMQ | <10MB idle, CNCF, air-gap ready |
| Embedded DB | **BuntDB** (dev) / etcd | PostgreSQL | Zero deps, in-cluster, < 5MB |
| OCI Registry | **Zot** | Harbor, Docker Registry | CNCF, <50MB, air-gap native |
| Image Build | **BuildKit** (rootless) | Kaniko, buildah | Fastest, rootless, cache-efficient |
| Policy Engine | **Kyverno** | OPA Gatekeeper | Simpler CRD-native policies |
| CVE Scanner | **Trivy** | Anchore, Snyk | <100ms scan, offline DB, CNCF |
| Cost Tracking | **OpenCost** | CloudHealth, Kubecost | OSS, Prometheus-native |
| Auto-Scaling | **KEDA** | Custom HPA | NATS-native scaler, scale-to-zero |
| UI Framework | **React + Vite + Module Federation** | Angular, Vue | Module federation for plugin hot-reload |
| State Management | **Zustand** | Redux | Minimal footprint, reactive |
| LLM (online) | **Google Gemini** | GPT-4, Claude | Native Antigravity integration |
| LLM (air-gap) | **Ollama + Llama-3.2-3B-Q4** | None | 2GB, fully offline |
| GitOps | **Argo CD** (default) / Flux | Manual push | Visual history, operator integration |

---

## 11. Non-Functional Requirements Mapping

| NFR | Constraint | Design Response |
|---|---|---|
| NFR-001 | < 100MB RAM per instance | Go binary, GOGC tuning, embedded BuntDB, Zot, NATS ~10MB |
| NFR-002 | Zero cleartext secrets in exports | `secretRef` pattern only; `renderScrubbed()` on export |
| NFR-003 | Automated test coverage for all contributions | Go `testing` + `testcontainers-go` + Playwright for UI |
| NFR-004 | CI/CD backward compat enforcement | CRD version bump policy; schema migration webhook |
| NFR-005 | Godoc on every exported symbol | `golangci-lint` with `godot` + `revive` doc rules |

---

## 12. Repository Structure

```
KubeWorkFlow/
├── cmd/
│   ├── operator/         (kttm-operator binary)
│   └── kttm/             (kttm CLI binary)
│
├── core/
│   ├── api/v1alpha1/     (KttmApp, KttmConnector, KttmUIPlugin CRDs)
│   ├── advisor/          (AI Advisor: rule engine + LLM prompt)
│   ├── algorithms/       (Layer 10: 33 algorithm packages)
│   ├── cli/              (kttm run/validate/export/import/publish)
│   ├── compiler/         (DAG → Argo/Tekton CRD strategy)
│   ├── connectors/       (60+ connector implementations)
│   ├── debug/            (Breakpoint sidecar + WebSocket stream)
│   ├── engine/           (Materializer + adapters + envelope)
│   ├── gitops/           (Git push bridge)
│   ├── linter/           (GraphLinter: DFS + type checker)
│   ├── messaging/        (NATS JetStream client)
│   ├── nodebuilder/      (BuildKit factory + registry push)
│   ├── observability/    (OTel traces + Prometheus metrics)
│   ├── plugins/          (UI plugin registry + hot-reload)
│   ├── policy/           (Kyverno/OPA bridge)
│   ├── rbac/             (K8s RBAC → KTTM permission mapping)
│   └── scanner/          (Trivy CVE integration)
│
├── frontend/
│   ├── web-renderer/     (Polymorphic SPA: Developer+Admin+EndUser)
│   │   ├── src/
│   │   │   ├── planes/
│   │   │   │   ├── DeveloperSDK.jsx   (Canvas + debugger + node CRUD)
│   │   │   │   ├── AdminConsole.jsx   (RBAC + lifecycle + cost)
│   │   │   │   └── EndUserPortal.jsx  (Forms + dashboards only)
│   │   │   ├── components/
│   │   │   │   ├── PermissionGate.jsx
│   │   │   │   ├── DAGCanvas.jsx      (XYFlow visual editor)
│   │   │   │   ├── ConnectorCatalog.jsx
│   │   │   │   ├── DebugConsole.jsx
│   │   │   │   ├── SchemaForm.jsx     (schema-driven form renderer)
│   │   │   │   ├── AdvisorPanel.jsx
│   │   │   │   └── CostDashboard.jsx
│   │   │   └── store.js               (Zustand reactive state)
│   │   └── plugins/                   (Module Federation remotes)
│   └── canvas/                        (XYFlow DAG canvas - standalone)
│
├── internal/
│   └── workflow/         (controller-runtime reconciler)
│
├── deploy/
│   ├── helm/kttm/        (Production Helm chart)
│   ├── k3d/              (Local dev cluster config)
│   └── kustomize/        (Raw manifest install)
│
└── specs/
    ├── requirement.md    (This SRD)
    ├── flow-engine-proposal.md
    └── examples/         (Sample KttmApp CRDs)
```

---

## 13. Phased Implementation Roadmap

### Phase 1 — Core Engine (Weeks 1–4) ✅ Complete
CRDs, operator reconciler, GraphLinter, Argo compiler, NATS messaging, materializer, S3/PostgreSQL/webhook connectors, CLI, AI Advisor (rule engine).

### Phase 2 — Frontend & Connectors (Weeks 5–8) ✅ In Progress
Polymorphic SPA, Zustand store, AI Advisor UI panel, Kafka/NATS connectors, OpenTelemetry.

### Phase 3 — Developer SDK (Weeks 9–14)
- XYFlow DAG canvas (visual node editor)
- Node grouping UI + execution mode selector
- Breakpoint debugger (sidecar + WebSocket stream)
- Schema-driven form builder (drag-and-drop binding REQ-015)
- Custom React bundle injection (REQ-016)
- Module Federation plugin hot-reload (REQ-017)
- Connector catalog UI (REQ-009, REQ-018, REQ-019)

### Phase 4 — Admin Plane & Security (Weeks 15–18)
- Admin Console (RBAC matrix UI, lifecycle controls, bundle import)
- Trivy CVE scanner integration (REQ-046)
- Kyverno policy binding (REQ-036)
- OpenCost cost dashboard (REQ-028)
- K8s RBAC → KTTM permission mapper (REQ-039)
- CIS Benchmark Helm policies (REQ-047)

### Phase 5 — Algorithm Engine & Advanced (Weeks 19–24)
- Embedded algorithm library (Layer 10, ALG-001–033)
- KEDA auto-scaler integration (REQ-029)
- In-place vertical resize (REQ-030, K8s 1.27+)
- Node builder BuildKit factory (REQ-041)
- Composite connector composability (REQ-013)
- Connector publish to KTTM Hub (REQ-019)

### Phase 6 — Compliance & GA (Weeks 25–28)
- NIS2 / CRA compliance profiles
- HA deployment topology (REQ-044)
- Air-gap bundle export/import full pipeline
- E2E test suite (Playwright + testcontainers-go)
- API stability + CRD versioning webhook

---

## 14. Open Questions for Review

> [!IMPORTANT]
> **Q1 — Algorithm engine packaging:** Should the 33-algorithm library (Layer 10) ship as a single compiled binary embedded in the operator, or as separate microservice containers invokable as script nodes? Embedding keeps footprint minimal but makes the algorithm catalog harder to extend. Separate containers allow independent versioning but add ~20MB per runtime. **Recommendation: embed Level 1–9 (pure Go, zero deps), containerize Level 10 (ML/Consensus, heavy deps).**

> [!IMPORTANT]
> **Q2 — KttmApp naming vs. FlowEngine:** This repo already has a `FullStackApplication` CRD from the earlier FlowEngine implementation. Should KTTM use a new `KttmApp` CRD alongside it, or should KTTM _replace_ FlowEngine and migrate the existing CRDs? **Recommendation: KTTM supersedes FlowEngine — rename existing CRD to `KttmApp`, update all references.**

> [!IMPORTANT]
> **Q3 — LLM for air-gapped AI Advisor:** The SRD requires the AI Advisor to work in fully air-gapped environments. Bundling Ollama + Llama-3.2-3B-Q4 adds ~2GB to the air-gap bundle. Is this acceptable, or should the advisor fall back to rule-only suggestions? **Recommendation: Make the LLM optional (toggle in Helm values), default to rule-only.**

> [!NOTE]
> **Q4 — KEDA ScaleToZero for web app mode:** REQ-045 defines a "hybrid always-active web application" mode, but REQ-029 wants scale-to-zero. These conflict for web apps (can't scale to zero if it must always handle web traffic). Web apps should have `minReplicas: 1`, workflow-only apps can use `minReplicas: 0`.
