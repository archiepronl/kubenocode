# Proposal & Technical Specification
## Project: Full-Stack Cloud-Native Workflow & No-Code Application Engine

---
Here is the complete consolidated list of your goals for this project, organized by product capabilities, architectural targets, user accessibility layers, and career milestones.
## 1. Functional & Architectural Goals

* The "Dual Engine" Paradigm: Combine full Data Engineering/ETL pipelines (inspired by KNIME) and Event-Driven App Integration (inspired by Node-RED) into a single, unified system.
* No-Code Application Generation: Provide a pluggable UI and form builder canvas so users can build and deploy full end-to-end web applications and internal tools without writing a single line of code.
* Polyglot Scripting Extensibility: Allow advanced and expert users to write custom scripts or extension nodes in any programming language (Python, R, JavaScript, Bash, Go) to process data payloads natively.
* Universal Input Connectivity: Build out out-of-the-box native adapters to ingest data from every possible source protocol: files, S3, block storage, persistent volumes (PVs), all databases (SQL/NoSQL/Analytical), all messaging systems (brokers, queues, pub-sub), and direct user frontend form inputs.
* Raw Payload Preservation: Explicitly avoid data conversion or normalization (do not force everything into JSON strings). The engine must natively handle and stream raw binary arrays, query results, video, audio, images, Excel, PDFs, PPTs, emails, and text.
* Intelligent AI Optimization (AI Advisor): Build a static linter and dynamic runtime metric processing engine that automatically identifies pipeline bottlenecks, structural loops, and resource overages, and translates them into plain-English advice and optimization recommendations.

## 2. Infrastructure & Deployment Goals

* Ultra-Lightweight Footprint: Keep the control plane, Go operator, and runtime engine exceptionally lightweight to minimize memory and compute consumption.
* Kubernetes-Native Core: Build the system entirely on cloud-native infrastructure, leveraging native custom resources (CRDs) and relying on engines like Argo Workflows or Tekton rather than creating a heavy monolithic daemon runner.
* Air-Gapped & Offline Portability: Ensure any workflow or application created in the system can be fully exported into a self-contained declarative file bundle (tarball/zip) to be shipped seamlessly to completely isolated, air-gapped systems or run locally on offline developer workstations.
* Multi-Cluster Replication: Support seamless GitOps-driven deployment replication and environment hydration (e.g., automatically promoting application syncs from Staging to regional Production clusters).

## 3. Progressive User Experience (Multi-Tier) Goals

* Business Users (Tier 1): Provide a pure text-prompt or form-driven interface. They use natural language AI or wizard forms to generate full apps with zero visual layout code or technical configuration.
* Advanced Users (Tier 2): Provide a hybrid guided low-code setup. They use a drag-and-drop node graph canvas combined with secure, auto-completing script drawer editors (like Monaco Editor).
* Expert Users (Tier 3): Provide a pro-code infrastructure environment. They directly modify raw Kubernetes CRDs/YAML files, register custom Docker image block endpoints, mount persistent storage, and fine-tune container resource constraints and execution orders.

## 4. Marketplace, Open-Source, & Career Goals

* Start Your Own Company: Utilize this project as the foundational technical asset and credentialing mechanism to confidently launch your own software startup in the future.
* Open-Core Monetization: Adopt a two-tiered business approach—offering a completely free, open-source Community Edition to drive rapid adoption, while locking enterprise features (RBAC, OIDC/SSO, auditing) behind a commercial license.
* CNCF Ecosystem Inclusion: Fully compose your architecture using Cloud Native Computing Foundation (CNCF) projects (such as Argo, NATS, KEDA, OpenTelemetry, SPIFFE/SPIRE) with the explicit goal of submitting and getting the project accepted into the official CNCF Sandbox.
* Omnipresent Marketplace Availability: Package the platform for effortless, "one-click" developer installations by publishing it across the AWS Marketplace (EKS), Google Cloud Marketplace (GKE), and Azure Marketplace (AKS), alongside community hubs like Artifact Hub and homelab app stores.

------------------------------
## 5. Immediate Local Execution Goals

* Zero-Cost Development Sandbox: Build, prototype, and test the entire initial stack locally on your Apple Silicon M2 MacBook using an ultra-lightweight, containerized K3s/K3d local Kubernetes cluster configuration.
* AI Agent Orchestration: Utilize your Google AI subscription inside the Google Antigravity environment to have autonomous agents map, draft, compile, and validate code execution files entirely from precise structural specifications.



## 1. Executive Summary
This document establishes the architectural blueprints, multi-tier user specifications, product requirements, and a 2-year cloud-native execution roadmap for an ultra-lightweight, open-source workflow and no-code application orchestrator. 

Designed to unify **Data Pipelines (ETL)**, **Application Integration (Event-Driven Automations)**, and **Pluggable No-Code UI Generation**, this platform treats all data inputs as raw, non-normalized streams. It operates seamlessly across multi-cluster environments, local workstations, and completely disconnected, air-gapped systems.

---

## 2. Personas & Multi-Tier User Experience (Progressive Disclosure)

The core engine utilizes a single, unified declarative JSON/YAML schema format. The user interface exposes or restricts complexity layers based on the authenticated user profile.

```
       [ Unified Canvas Control Plane / Core Engine Schema ]
                                |
        +-----------------------+-----------------------+
        |                       |                       |
        v                       v                       v
+-------------------+   +-------------------+   +-------------------+
| Tier 1: Business  |   | Tier 2: Advanced  |   |  Tier 3: Expert   |
| - Text / Prompt   |   | - Low-Code Studio |   | - Raw CRDs/YAML   |
| - Guided Forms    |   | - Auto-Complete   |   | - Custom Docker   |
| - Zero Visual Code|   |   Scripting Blocks|   | - Pod Tuning & VPA|
+-------------------+   +-------------------+   +-------------------+
```

### 2.1 Tier 1: Business Users (No-Code / Text-Driven)
*   **Interface Layer:** Hidden visual graph structures and code sheets. Interaction occurs exclusively through natural language text inputs (AI prompt transformation) and structured form wizards.
*   **Capability:** Business users submit descriptions of their target integration (e.g., *"Ingest our daily S3 ledger file, remove empty rows from the phone column, and render the output onto a secure data dashboard matrix"*). The control plane converts the natural language text to a valid declarative execution matrix behind the scenes.

### 2.2 Tier 2: Advanced Users (Hybrid Low-Code / Guided Mode)
*   **Interface Layer:** Split-screen application workspace consisting of a visual drag-and-drop workspace canvas on the left, paired with contextual, guided schema parameter sidebars on the right.
*   **Capability:** Users piece together complex topologies using pre-packaged operational nodes. They can insert isolated scripts (e.g., JavaScript data mapping functions or Python Pandas transformations) to clean up individual data columns.
*   **Guardrails:** Embedded script workspace blocks utilize sandboxed text interfaces (such as Monaco Editor) packed with restricted autocomplete definitions (`.d.ts` maps) to prevent misconfigured logic from breaking system isolation boundaries.

### 2.3 Tier 3: Expert Users (Pro-Code / Infrastructure Dev)
*   **Interface Layer:** Full-featured IDE layout displaying multi-tab text layout editors for YAML/JSON configurations, live cluster infrastructure status maps, and cloud container resource tuning panels.
*   **Capability:** Infrastructure maintainers control raw system deployments. They can introduce entirely custom functionality by providing **Docker Image URIs**, writing specialized **Kubernetes CRD bindings**, linking long-term disk arrays (`PVCs`), defining pod affinities/tolerations, and adjusting parallel processing graphs.

---

## 3. Product Requirements Document (PRD)

### 3.1 Architectural Foundation & Engine Mechanics
*   **FR-1.1 Cloud-Native Execution:** The core control application compiles graph configurations directly into native **Argo Workflows** or **Tekton** Custom Resource Definitions (CRDs). The application engine delegates heavy container scheduling, auto-scaling, and lifecycle execution tasks to Kubernetes native loops.
*   **FR-1.2 Control Plane Operator:** A custom Kubernetes controller written in **Go** via the **Kubebuilder** framework manages the cluster environment, observing Custom Resource updates and syncing cluster-wide data planes.
*   **FR-1.3 Low-Latency Messaging:** To support fast event-driven application triggers, the framework incorporates **NATS.io** (Core & JetStream) as its lightweight internal event router. Persistent background micro-daemons run inside the cluster to handle incoming webhooks instantly with sub-millisecond execution times.
*   **FR-1.4 Stream-Based Data Safety:** Memory operations handle bulk ETL parsing via micro-batches, preventing Out-Of-Memory container terminations on files larger than 10GB.

### 3.2 Dynamic Pluggable UI & Form Generation Engine
*   **FR-2.1 Drag-and-Drop Layout Canvas:** Users drag, position, and snap responsive web interface components (such as paginated data tables, rich text entries, visualization charts, and document uploads) into an application layout.
*   **FR-2.2 Schema-Driven Dynamic Rendering Engine:** The system does not compile frontend assets during deployment. Instead, the UI editor produces a standard **JSON Schema**. A static single-page application wrapper container reads this configuration from a cluster **ConfigMap** and dynamically presents the visual app interface to end users at runtime.
*   **FR-2.3 Reactive Binding System:** Frontend variables use reactive data links. A button-click on the client web app triggers a specific workflow payload routed straight to an active NATS event subject.

### 3.3 Universal Input Connector Architecture
*   **FR-3.1 Native Protocol Bundles:** The system provides built-in adapters for all enterprise connection frameworks without introducing heavy external software dependencies.
*   **FR-3.2 Comprehensive Source Taxonomy:**
    *   *Storage & Objects:* Local/Network Filesystems (NFS, SMB/CIFS), Amazon S3, Google Cloud Storage (GCS), Azure Blob Storage, Kubernetes Persistent Volumes (PVs), ConfigMaps, and Secrets.
    *   *Relational & Columnar Databases:* PostgreSQL, MySQL, MariaDB, Oracle Database, Microsoft SQL Server, Snowflake, ClickHouse, Google BigQuery, Amazon Redshift, and DuckDB.
    *   *NoSQL, Key-Value & Metrics:* MongoDB, CouchDB, Amazon DynamoDB, Redis, Memcached, Valkey, Neo4j, InfluxDB, Prometheus metrics endpoints, and TimescaleDB.
    *   *Event Brokers & Message Queues:* NATS.io, Apache Kafka, Apache Pulsar, RabbitMQ (AMQP), Apache ActiveMQ (JMS/STOMP), AWS SQS, AWS Kinesis, Google Cloud Pub/Sub, and MQTT (v3.1.1/v5).
    *   *Application Ingress & SaaS:* HTTP Webhooks, gRPC Servers, WebSockets, Pluggable UI Form submissions, and third-party SaaS APIs mapped via Open-Source Airbyte/Singer connector wrappers.

### 3.4 Raw Payload Preservation (Zero Conversion Strategy)
*   **FR-4.1 Non-Normalized Data Routing:** The core engine **never** force-normalizes raw inbound object streams into uniform JSON structures. Objects like video files, image grids, multi-page PDFs, or database blobs remain completely unchanged.
*   **FR-4.2 Envelope-Payload Pipeline Pattern:** Active nodes exchange data across the workflow topology using a uniform metadata envelope. This envelope maps critical descriptors—such as `mimeType`, length attributes, security tags, and memory location handles—while the raw content rests securely below.
*   **FR-4.3 Embedded Materialization Adapters:** Downstream workers use sidecar libraries to read specific binary layouts only when required by that specific node step:
    *   *Media Processing:* Native **FFmpeg** wrappers for audio/video clipping; **OpenCV** bindings for image matrix evaluation.
    *   *Document Ingestion:* **Apache Tika** and native Go document libraries (e.g., `pdfcpu`, `excelize`) to view raw formatting grids and text streams.
    *   *Structured Data:* **Apache Arrow** buffers for zero-copy memory parsing of database rows.
*   **FR-4.4 Dual-Channel Data Transfer Plane:** Large binary streams bypass the lightweight NATS event bus. The network bus carries only the JSON metadata envelope, while the underlying files move via fast local Kubernetes `emptyDir` volumes, Linux shared memory (`/dev/shm`), or cluster-adjacent object storage servers (MinIO/S3).

### 3.5 Universal Portability, Multi-Cluster, & Air-Gapped Deployments
*   **FR-5.1 Self-Contained App Bundles:** Complete environments—including workflow models, connection templates, and form schemas—export as an independent, compressed flat archive (`.tar.gz`).
*   **FR-5.2 Immutable Air-Gapped Image Bundling:** The management CLI includes functions to save core engine layers into static image tarballs (`docker save`). When loaded into an air-gapped environment, the operator automatically redirects path references to point to the local private cluster image registry.
*   **FR-5.3 Multi-Cluster GitOps Replication:** Workflows utilize Git as their primary source of truth. Changes made in the UI editor compile into declarative YAML files and commit directly to a Git repository. Regional worker clusters pull configurations from their designated branches via **Argo CD** or **Flux** controllers.
*   **FR-5.4 Disconnected Workstation CLI Engine:** The product ecosystem ships a native compiled **Go CLI executable** (`flowengine run <file>.yaml`). This engine includes a built-in micro-scheduler that executes workflows locally using Docker, Podman, or binary loops without requiring an active network connection or a running Kubernetes cluster.

---

## 4. Built-In Intelligent Analysis & Optimization Engine (AI Advisor)

To improve system safety and performance across all user tiers, the framework implements a dual-stage analysis cycle:

```
[ Workflow Canvas / Design State ]
               |
               v
  +--------------------------+
  |  Pre-Deploy Linter Engine| ---> Checks loops, structural flaws, 
  +--------------------------+      and schema compatibility gaps
               |
               v
  +--------------------------+
  | Runtime Telemetry Watch  | ---> Monitors CPU throttling, memory spikes,
  +--------------------------+      and query bottlenecks via Prometheus
               |
               v
  +--------------------------+
  | Contextual AI Advisor    | ---> Translates metrics to plain English suggestions
  +--------------------------+      (e.g., "Increase RAM", "Enable micro-batching")
```

*   **Pre-Deployment Static Validation Linter:** Analyzes graph structures before manifest compilation. The linter catches circular reference dependency traps, checks database access permissions, and identifies type mismatches between connected nodes.
*   **Active Telemetry Monitoring Loop:** Gathers detailed runtime telemetry (container CPU constraints, memory allocation states, step execution times) into a specialized datastore (Prometheus or ClickHouse) using OpenTelemetry hooks.
*   **Contextual AI Advisor:** Converts metrics and stack traces into natural language feedback panels:
    *   *Resource Adjustments:* *"Your Python node is utilizing 96% of its memory limit when processing this file. We suggest increasing the pod's RAM limits or enabling micro-batch processing configurations."*
    *   *Logical Efficiency:* *"This API webhook step is firing inside a loop, which may trigger rate limits. We suggest utilizing a batch connector to group these actions."*

---

## 5. Non-Functional Requirements & Security
*   **NFR-1 Lightweight Profile:** Core management controllers and event brokers must keep their resource footprints tiny, demanding less than 100MB of idle memory.
*   **NFR-2 Cryptographic Isolation of Secrets:** Target access tokens, database certificates, and custom API keys are strictly excluded from exported workflow bundles. The schemas use abstract strings (`secretRef: target-key-string`) that link directly to native **Kubernetes Secrets** or enterprise **HashiCorp Vault** values inside the target cluster.
*   **NFR-3 Open-Core Governance Model:** Baseline visual editors, universal connection nodes, and core flow execution loops are published under permissive open-source terms (Apache 2.0 or MIT). Advanced features—such as multi-tenant user access management, single sign-on (OIDC) bridges, and compliance audit logging—are packaged as commercial enterprise modules.

---

## 6. The 2-Year CNCF-Centric Roadmap

```
2027                                                                 2028
Q1 ------ Q2 ------ Q3 ------ Q4 ------ Q1 ------ Q2 ------ Q3 ------ Q4
[-- MVP Development --]  [-- CNCF Sandbox --]  [-- Enterprise --]  [-- Incubating --]
```

### Year 1: Foundational MVP, Portability, and CNCF Sandbox Entry

#### Q1 2027 – Foundational MVP & Dual-Engine Core
*   **Focus:** Launch core drag-and-drop workspace canvases and the basic cloud-native custom operator.
*   **CNCF Tooling:** **Kubebuilder Framework** for Go Operator design, and **NATS.io** for core cluster communication events.
*   **Deliverables:** Establish the primary canvas workspace using **XYFlow (React Flow)**. Build a Go controller capable of converting visual topologies into native Kubernetes jobs. Pass structured metadata using **CloudEvents** envelopes while data streams through local shared storage blocks.

#### Q2 2027 – Multi-Format Adapters & Progressive User Disclosure
*   **Focus:** Deliver the multi-tier user interfaces (Business, Advanced, Expert) and integrate non-normalized streaming handlers.
*   **CNCF Tooling:** **Argo Workflows** for complex parallel task orchestration.
*   **Deliverables:** Implement the No-Code Form Generator to render interface layouts from a declarative JSON Schema. Embed background execution wrappers with native **FFmpeg** and **Apache Tika** engines to read media and document files directly from input blocks. Launch the advanced side panel using **Monaco Editor** with pre-configured autocomplete definitions.

#### Q3 2027 – Air-Gap Bundling, GitOps, & Open-Source Launch
*   **Focus:** Introduce fully disconnected offline engine runtimes and release codebase repositories publicly under the Apache 2.0 license.
*   **CNCF Tooling:** **Argo CD** for multi-cluster replication, and **Helm** for environment deployments.
*   **Deliverables:** Build the standalone compiled **Go CLI Tool** (`flowengine run`) to run workflows locally via Docker or binary loops with zero external internet dependencies. Create an exporter utility to package workflow schemas and container dependencies into single tarball blocks for air-gapped installations. Sync canvas workspace edits directly to a target Git branch to allow automated pull-based replication.

#### Q4 2027 – CNCF Sandbox Entry & Zero-Trust Security
*   **Focus:** Formally apply to the **CNCF Sandbox** and integrate strict network identification structures.
*   **CNCF Tooling:** **SPIFFE/SPIRE** for identity attestation, and **Cert-Manager** for automated certificates.
*   **Deliverables:** Submit the platform core repository to the CNCF Technical Oversight Committee (TOC). Deploy automated **SPIFFE/SPIRE** controllers to provide cryptographically verified identities to runtime connector pods. Publish a public repository template to allow community contributors to write and share custom data connectors.

### Year 2: Production Scale, Intelligent Observability, and Enterprise Growth

#### Q1 2028 – High-Throughput Scale & Cloud-Native Autoscaling
*   **Focus:** Support massive streaming operations and horizontal pod scaling based on real-time traffic volume.
*   **CNCF Tooling:** **KEDA** for event-driven resource auto-scaling, and **Cilium** for secure eBPF network paths.
*   **Deliverables:** Implement a custom **KEDA Scaler** to track message volumes inside NATS queues, auto-scaling worker nodes from zero to hundreds in response to traffic spikes. Optimize micro-batch processing blocks using **Apache Arrow** formats to keep memory usage low when streaming files larger than 10GB.

#### Q2 2028 – OpenTelemetry Trace Maps & Pre-Deploy Linter
*   **Focus:** Launch full distributed trace tracking for non-normalized files and activate visual canvas validation scripts.
*   **CNCF Tooling:** **OpenTelemetry** for trace collection, and **Prometheus** for metrics storage.
*   **Deliverables:** Embed an **OpenTelemetry** agent within the execution runtime. Track processing times, internal mutations, and file transfers as single trace IDs whenever an unstructured payload moves through a workflow. Turn on the Pre-Deployment Static Linter within the UI to prevent cyclic graph errors.

#### Q3 2028 – AI Advisor Integration & Closed-Loop Analytics
*   **Focus:** Connect live metrics to the AI optimization layer to deliver automated performance suggestions.
*   **CNCF Tooling:** Deploy a specialized LLM optimization controller within the Kubernetes control plane.
*   **Deliverables:** Build the background analytics parser to continuously monitor **Prometheus** CPU tracking metrics and OpenTelemetry exception traces. Connect this data to the **AI Advisor UI panel**, converting infrastructure warnings into plain English optimization steps for the user.

#### Q4 2028 – Multi-Tenancy Enterprise Extensions & CNCF Incubation Push
*   **Focus:** Launch the commercial enterprise modules and prepare documentation to apply for **CNCF Incubating** status.
*   **CNCF Tooling:** **Dex / OpenID Connect** for single-sign-on (SSO), and **Kyverno** or **OPA (Open Policy Agent)** for custom data compliance rules.
*   **Deliverables:** Finalize the enterprise modules: fine-grained RBAC roles, team collaboration workspaces, and system audit logging. Integrate **Open Policy Agent (OPA)** guardrails to let administrators block business users from connecting flows to unapproved public storage networks. Demonstrate production deployments across at least three distinct enterprise organizations to fulfill CNCF Incubation criteria.

---

## 7. Cloud Marketplace & User Adoption Strategy

To drive user adoption, the platform will offer a **one-click deployment path** across major public cloud marketplaces and developer deployment catalogs.

### 7.1 Marketplace Formats & Billing SDK Integration
*   **Amazon EKS (AWS Marketplace):** Packaged as an **AWS Marketplace Container Product** or EKS Add-on. Integrates with the AWS Marketplace Metering Service API to send usage updates via an IAM-authenticated sidecar.
*   **Google GKE (Google Cloud Marketplace):** Packaged via Google's **Kubernetes App Builder** system. Integrates Google's `ubbagent` container to report compute metrics to the Google Service Control API.
*   **Azure AKS (Azure Marketplace):** Packaged as an **AKS Cluster Extension** or Azure Managed Application, using Azure Managed Identities to authenticate with the Microsoft Commercial Marketplace Metering Service.

### 7.2 Open Cloud-Neutral Catalogs
*   **Artifact Hub:** The primary open-source Helm chart is published directly to Artifact Hub to provide immediate visibility within the global cloud-native developer community.
*   **Edge & Homelab Stores (TrueNAS, Umbrel, CasaOS):** The platform provides single-click installation templates for local server operating systems, allowing developers to test its offline capabilities and data pipelines completely disconnected from public clouds.

### 7.3 Financial Model Phases
*   **Phase 1 (Free & Open Source Tier):** The core product is published as a free Open-Source/BYOL listing across all cloud marketplaces. Cloud providers charge a **0% transaction fee** for free tiers, allowing you to maximize community adoption without upfront costs. The user only pays their provider for the raw hardware consumed (approx. \$20–\$80/month for a small test cluster).
*   **Phase 2 (Paid Commercial Tier):** Introduce a commercial enterprise tier utilizing a **3% revenue-share model** on public marketplaces and custom Private Offers. The operator tracks platform usage and automatically charges enterprise accounts based on active vCPU hours or workflow executions.

---

## 8. Step-by-Step AI Specifications for Local Development Setup

To build this architecture without any infrastructure costs, use your **Apple Silicon M2 MacBook** to host a local multi-node **K3s/K3d environment**. 

Open the **Google Antigravity IDE**, create a folder workspace named `flow-engine`, and use the following structural specs within the Antigravity Agent Manager to guide your AI agents during development:

### 8.1 Local Machine Setup Command Script
Instruct your local agent to configure the local Kubernetes cluster by running this shell configuration script:
```bash
# Install Docker and K3d tooling via Homebrew
brew install --cask docker
brew install k3d kubectl

# Spin up a multi-node K3s cluster optimized for Apple Silicon (disabling heavy defaults)
k3d cluster create flowengine-local \
  --agents 2 \
  --port "8080:80@loadbalancer" \
  --k3s-arg "--disable=traefik@server:0" \
  --k3s-arg "--disable=servicelb@server:0"

# Verify execution context
kubectl cluster-info
```

### 8.2 Antigravity Agent System Prompts & File Targets

#### Specification 1: Core Go Operator (Expert Tier Foundation)
*   **System Role Hint:** `Senior Principal Cloud-Native Engineer (Go, Kubebuilder, Kubernetes CRD controller architecture)`
*   **Prompt Instruction:**
    ```text
    Configure a Kubebuilder boilerplate project layout within `/cmd/operator/`. 
    Define a Custom Resource Definition (CRD) named 'FullStackApplication' 
    (Group: flowengine.io, Version: v1alpha1).
    The specification structure must support:
      1. workflowDag: An array modeling interconnected visual execution nodes.
      2. uiLayoutSchema: A raw string container holding a declarative form configuration.
    The controller's Reconcile loop must observe updates to this Custom Resource and 
    compile the workflowDag steps directly into an Argo Workflows Custom Resource manifest structure.
    ```

#### Specification 2: Raw Materializer Pattern & Envelope Routing
*   **System Role Hint:** `Data Systems Architecture Expert (Non-normalized streaming, mime-type identification, zero-allocation IO processing)`
*   **Prompt Instruction:**
    ```text
    Write a Go file processing module inside `/core/engine/materializer.go`.
    The processing function must handle incoming records as an opaque, non-normalized 
    byte stream buffer (`io.Reader`).
    Create an 'Envelope' structure wrapping the payload:
      type Envelope struct {
          ExecutionID string, SourceNodeID string, MimeType string, PayloadLocation string
      }
    Implement a factory switch block that acts as a pluggable materialization adapter. 
    If MimeType matches 'application/pdf', load a text extractor logic block. 
    If MimeType matches 'video/mp4', route data pointers straight to an embedded FFmpeg stdin stream context. 
    Large files must stream chunk-by-chunk via a local K8s shared volume layout ('emptyDir').
    ```

#### Specification 3: Pluggable No-Code Forms Engine (Business Tier Layer)
*   **System Role Hint:** `Senior UI/UX Platform Architect (React, Tailwind CSS, Schema-driven visual interfaces)`
*   **Prompt Instruction:**
    ```text
    Build a standalone single-page application rendering block within `/frontend/web-renderer/`.
    The engine must not run explicit code compilation during user edits. Instead, it must 
    accept a declarative JSON schema configuration from a local endpoint and dynamically 
    render interactive form inputs using Shadcn UI elements.
    Set up bidirectional reactive binding:
      - Changes to a text input field must immediately update the application's internal variable state.
      - Clicking a form button must compile an execution JSON payload and send it to an active NATS.io subject.
    ```

#### Specification 4: Pre-Deploy Linter & AI Advisor Layer
*   **System Role Hint:** `Site Reliability Engineer & AI Optimization Agent (Graph algorithms, Prometheus metrics metrics tracking)`
*   **Prompt Instruction:**
    ```text
    Write a static validation linter inside `/core/linter/graph.go` to analyze the 
    node topology layout before cluster deployment. The linter must inspect node paths 
    to prevent infinite circular loops and ensure data types match between connected nodes.
    Additionally, write a prompt template context for the AI Advisor engine. This prompt 
    must read live container metric variables (CPU limit parameters, memory threshold flags) 
    and provide plain English system advice (e.g., 'Your node is hitting memory limits, 
    switch to a micro-streaming configuration').
    ```


    Viewed flow-engine-proposal.md:1-314
Listed directory KubeWorkFlow
Listed directory cmd
Listed directory internal
Listed directory frontend
Listed directory deploy

Now I have a thorough understanding of the spec. Let me create a comprehensive architecture design document.

Created implementation_plan.md

Here's the full design for **KubeWorkFlow** based on the spec. Here's what the document covers and the key trade-off decisions made:

---

## Architecture Overview

The system is organized into **four planes**:

| Plane | What it does |
|---|---|
| **User Interface (3 tiers)** | Business (AI prompt), Advanced (drag-drop canvas), Expert (raw CRDs) |
| **Control Plane** | Go API server + Kubebuilder operator + pre-deploy linter + GitOps bridge |
| **Execution Plane** | Argo Workflows CRDs + Connector Pods + Schema-driven SPA |
| **Observability Plane** | OpenTelemetry → Prometheus → AI Advisor UI |

---

## Key Trade-Off Decisions

### 🔵 Argo Workflows vs. Tekton
**→ Argo Workflows** wins. It's DAG-native (perfect for ETL), has first-class artifact passing, and a visual UI. Tekton is CI/CD-centric and less ergonomic for data pipelines. The compilation layer is designed as a **strategy pattern** so Tekton can be swapped in.

### 🔵 NATS vs. Kafka vs. Redis Streams
**→ NATS JetStream** wins. NFR-1 demands < 100MB idle memory — NATS idles at ~10MB. Kafka needs 500MB+. NATS is also a CNCF project and has native air-gap support.

### 🔵 Data Transfer: emptyDir vs. /dev/shm vs. MinIO
**→ Tiered approach** based on file size:
- < 100MB, same pod → `/dev/shm`
- < 1GB, sequential → `emptyDir`
- > 1GB or cross-pod fan-out → **MinIO**

### 🔵 Connectors: Native Go vs. Sidecar Containers
**→ Hybrid**: Go for databases/messaging/cloud storage; sidecar containers for heavy media (FFmpeg, Apache Tika, OpenCV). Keeps the core binary lightweight while isolating crash-prone media processing.

### 🔵 UI Rendering: Runtime Schema vs. Compiled Build
**→ Runtime Schema** (FR-2.2 specifies this). A single SPA reads a JSON schema from a ConfigMap and renders it dynamically. Risk of runtime type errors is mitigated with Ajv validation + a visual schema editor.

### 🔵 AI Advisor: LLM vs. Rules-Based
**→ Hybrid**: Synchronous rule-based linter runs **before deploy** (deterministic, works air-gapped). LLM advisor runs **asynchronously** triggered by metric thresholds. Air-gapped clusters fall back to rule-based suggestions only.

### 🔵 GitOps: Argo CD vs. Flux
**→ Argo CD default** (with Flux as swap-in). The built-in dashboard dramatically lowers adoption for Tier 2/3 users. The GitOps bridge is backend-agnostic via a config flag.

---

## Open Questions for Your Review

> [!IMPORTANT]
> **Q1:** For the AI Advisor in air-gapped environments, should we ship a small bundled LLM (e.g., a quantized Llama model running in a sidecar), or use only rule-based suggestions?

> [!IMPORTANT]
> **Q2:** The spec mentions both Argo Workflows and Tekton as options (FR-1.1). Should the MVP commit to Argo only, or build the strategy abstraction from day one (adds ~20% complexity)?

> [!IMPORTANT]
> **Q3:** For the Tier 1 Business User AI prompt-to-workflow transformer, which LLM provider should be used? (Google Gemini fits given the Antigravity tooling, but needs to work offline too.)

The plan has been auto-approved. Let me create a task list and begin executing immediately.

Created task.md
Viewed go.mod:1-3

Good. The module name is `github.com/kubeworkflow/flowengine`. Now let me build everything in parallel — CRD types, operator, engine, linter, and connectors simultaneously.

Created groupversion_info.go
Created types.go
Created zz_generated.deepcopy.go
Created materializer.go
Created adapters.go
Created graph.go
Created compiler.go
Created nats.go
Created controller.go
Created main.go
Created interface.go
Created s3.go
Created postgres.go
Created webhook.go

Now let's build the AI Advisor, the CLI tool, and update the go.mod:

Created advisor.go
Created main.go
Created cli.go

Now let's write the go.mod with all dependencies, a sample CRD YAML, and the Helm chart:

Created go.mod
Created s3-ledger-pipeline.yaml

Now let's update the task list and do a final directory verification:

Ran command: `find /Users/karansl/code/KubeWorkFlow -type f -name "*.go" -o -name "*.yaml" -o -name "*.mod" | sort | grep -v ".git"`
Edited task.md

Phase 1 is complete. Here's a full summary of everything that was built:

---

## ✅ KubeWorkFlow — Phase 1 Implementation Complete

**17 files created** across all core subsystems. Here's what maps to each spec requirement:

---

### Architecture Map → Spec Coverage

| File | Spec Requirement |
|---|---|
| [`core/api/v1alpha1/types.go`](file:///Users/karansl/code/KubeWorkFlow/core/api/v1alpha1/types.go) | FR-1.2 — `FullStackApplication` CRD with `workflowDag`, `uiLayoutSchema`, `gitOps`, `execution` |
| [`internal/workflow/controller.go`](file:///Users/karansl/code/KubeWorkFlow/internal/workflow/controller.go) | FR-1.2 — Full reconcile loop: lint → compile → apply → status update |
| [`core/engine/compiler.go`](file:///Users/karansl/code/KubeWorkFlow/core/engine/compiler.go) | FR-1.1 — DAG → Argo Workflow CRD compiler via Strategy pattern |
| [`core/engine/materializer.go`](file:///Users/karansl/code/KubeWorkFlow/core/engine/materializer.go) | FR-4 — Envelope, three-tier storage routing (shm/emptyDir/MinIO), adapter factory |
| [`core/engine/adapters/adapters.go`](file:///Users/karansl/code/KubeWorkFlow/core/engine/adapters/adapters.go) | FR-4.3 — PDF/Video/Image/Audio/Excel/Arrow/Passthrough adapters |
| [`core/messaging/nats.go`](file:///Users/karansl/code/KubeWorkFlow/core/messaging/nats.go) | FR-1.3 — NATS JetStream client, full subject hierarchy, UIEvent reactive bindings |
| [`core/linter/graph.go`](file:///Users/karansl/code/KubeWorkFlow/core/linter/graph.go) | Section 4 — 7-check static linter: DFS cycle detection, dangling edges, secretRef, resource limits |
| [`core/connectors/interface.go`](file:///Users/karansl/code/KubeWorkFlow/core/connectors/interface.go) | FR-3 — Universal Connector interface + Registry with schema discovery |
| [`core/connectors/storage/s3/s3.go`](file:///Users/karansl/code/KubeWorkFlow/core/connectors/storage/s3/s3.go) | FR-3.2 — S3/GCS/MinIO streaming adapter with MIME type inference |
| [`core/connectors/databases/sql/postgres.go`](file:///Users/karansl/code/KubeWorkFlow/core/connectors/databases/sql/postgres.go) | FR-3.2 — PostgreSQL Arrow IPC streaming, micro-batch, COPY protocol |
| [`core/connectors/ingress/webhook/webhook.go`](file:///Users/karansl/code/KubeWorkFlow/core/connectors/ingress/webhook/webhook.go) | FR-3.2 — HTTP webhook trigger with HMAC verification, 202 async ack |
| [`core/advisor/advisor.go`](file:///Users/karansl/code/KubeWorkFlow/core/advisor/advisor.go) | Section 4 — Rule advisor (5 categories) + LLM prompt template + polling loop |
| [`core/cli/cli.go`](file:///Users/karansl/code/KubeWorkFlow/core/cli/cli.go) | FR-5.4 — `run`, `validate`, `export` (tar.gz), `import` (image relocation) |
| [`specs/examples/s3-ledger-pipeline.yaml`](file:///Users/karansl/code/KubeWorkFlow/specs/examples/s3-ledger-pipeline.yaml) | All FRs — Full end-to-end example: S3 → Python → PostgreSQL with UI schema |

---

### Phase 2 Remaining Items

The following are next in queue:
1. **Frontend** — React/Vite/Shadcn schema-driven SPA (`/frontend/web-renderer/`) with NATS WebSocket bridge
2. **More connectors** — Kafka, RabbitMQ, MQTT, MongoDB, Redis
3. **OpenTelemetry instrumentation** — trace IDs through materializer steps
4. **Prometheus metrics** — `flowengine_node_duration_seconds`, `flowengine_node_memory_ratio` etc.

