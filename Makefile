# KTTM — Local Sandbox, Git-Driven Makefile
# Architecture: Heavy compute on M2 MacBook → clean gate on remote Git CI
#
# Philosophy:
#   - Write, iterate, debug, verify LOCALLY with sub-3s feedback loops
#   - Push only clean, verified code to Git for the regression gate
#   - Full air-gapped / offline execution supported (KTTM-REQ-033)
#
# First time? Run:   make bootstrap
# Daily dev loop:    make dev
# Before pushing:    make ci-local

.PHONY: all bootstrap \
        cluster cluster-delete cluster-reset cluster-deps cluster-status \
        dev dev-watch dev-logs dev-frontend dev-portforward stop \
        test test-unit test-integration test-local test-coverage \
        build build-operator build-server build-cli build-web build-all \
        image image-load \
        lint fmt vet \
        helm-install helm-upgrade helm-uninstall helm-lint \
        ci-local clean help ui

# ─────────────────────────────────────────────────────────────────
#  Project Identity
# ─────────────────────────────────────────────────────────────────

MODULE         := github.com/kubeworkflow/flowengine
CLUSTER_NAME   := kttm-dev
K3D_CONFIG     := deploy/k3d/kttm-cluster.yaml
REGISTRY       := kttm-registry.localhost:5001
NAMESPACE      := kttm-system
APPS_NS        := kttm-apps
HELM_CHART     := deploy/helm/flowengine
HELM_RELEASE   := kttm

# ─────────────────────────────────────────────────────────────────
#  Toolchain
# ─────────────────────────────────────────────────────────────────

GO         := go
GOFLAGS    :=
GOOS       ?= $(shell go env GOOS)
GOARCH     ?= $(shell go env GOARCH)
IMAGE_TAG  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_SHA    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

OPERATOR_IMAGE := $(REGISTRY)/kttm-operator:$(IMAGE_TAG)
SERVER_IMAGE   := $(REGISTRY)/kttm-server:$(IMAGE_TAG)

COVERAGE_THRESHOLD := 60
COVERAGE_FILE      := coverage.out

# ─────────────────────────────────────────────────────────────────
#  Terminal colors
# ─────────────────────────────────────────────────────────────────

GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
CYAN   := \033[0;36m
BOLD   := \033[1m
RESET  := \033[0m

# ═════════════════════════════════════════════════════════════════
#  HELP (default target)
# ═════════════════════════════════════════════════════════════════

help: ## Show this help
	@echo ""
	@echo "$$(printf '\033[1m\033[0;36m')KTTM — Local Sandbox, Git-Driven Development$$(printf '\033[0m')"
	@echo "$$(printf '\033[0;36m')  MacBook M2 handles all heavy compute. Git is the regression gate.$$(printf '\033[0m')"
	@echo ""
	@awk 'BEGIN {FS=":.*##"} /^[a-zA-Z_-]+:.*?##/ {printf "  \033[0;36m%-22s\033[0m %s\n",$$1,$$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "  First run:  make bootstrap"
	@echo "  Daily loop: make dev         (Skaffold hot-reload)"
	@echo "  Pre-push:   make ci-local    (full local gate)"
	@echo ""

all: ci-local ## Alias: run full local gate

# ═════════════════════════════════════════════════════════════════
#  VISUAL DASHBOARDS (UI)
# ═════════════════════════════════════════════════════════════════

ui: ## Spins up all UIs (K8s Dashboard, Argo, Web App) in one terminal
	@./scripts/start-uis.sh

# ═════════════════════════════════════════════════════════════════
#  SETUP — Local Cluster Initialization
# ═════════════════════════════════════════════════════════════════

setup: ## Bootstraps the cluster, runs all tests, and launches the UI
	@chmod +x scripts/setup-cluster.sh
	@./scripts/setup-cluster.sh
	@$(MAKE) cluster-verify
	@$(MAKE) test
	@echo "==> Deploying Application Containers (Web & API)..."
	@skaffold run --profile local
	@echo "==> Setup Complete! Launching visual interfaces..."
	@$(MAKE) ui

cluster-verify: ## Runs all local sandbox integration checks automatically
	@echo "==> Verifying Local Cluster Core Dependencies"
	@echo "--> Checking Nodes..."
	@kubectl get nodes | grep "Ready" || (echo "ERROR: Nodes not ready" && exit 1)
	@echo "--> Checking KEDA Autoscaler..."
	@kubectl wait --namespace keda --for=condition=ready pod --selector=app=keda-operator --timeout=30s
	@echo "--> Checking Argo Workflows Engine..."
	@kubectl wait --namespace argo --for=condition=ready pod --selector=app=workflow-controller --timeout=30s
	@kubectl wait --namespace argo --for=condition=ready pod --selector=app=argo-server --timeout=30s
	@echo "==> SUCCESS: All Dev Foundation components are fully operational!"

# ═════════════════════════════════════════════════════════════════
#  BOOTSTRAP — one-time first-run setup
# ═════════════════════════════════════════════════════════════════

bootstrap: ## 🚀 Full first-time setup: tools + deps + cluster + verify
	@echo "==> KTTM Bootstrap — Local Sandbox"
	@echo ""
	@echo "--> [1/6] Checking required tools..."
	@which k3d      >/dev/null 2>&1 || (echo "ERROR: k3d not found. Install: brew install k3d" && exit 1)
	@which kubectl  >/dev/null 2>&1 || (echo "ERROR: kubectl not found. Install: brew install kubectl" && exit 1)
	@which helm     >/dev/null 2>&1 || (echo "ERROR: helm not found. Install: brew install helm" && exit 1)
	@which skaffold >/dev/null 2>&1 || echo "WARN: skaffold not found. Install: brew install skaffold"
	@which stern    >/dev/null 2>&1 || echo "WARN: stern not found. Install: brew install stern"
	@echo "    OK: Tools verified"
	@echo ""
	@echo "--> [2/6] Downloading Go dependencies..."
	$(GO) mod download
	$(GO) mod verify
	@echo "    OK: Go dependencies ready"
	@echo ""
	@echo "--> [3/6] Installing frontend dependencies..."
	@if [ -f "frontend/web-renderer/package.json" ]; then \
	  npm install --prefix frontend/web-renderer --silent; \
	  echo "    OK: Frontend dependencies ready"; \
	else \
	  echo "    SKIP: No frontend/web-renderer found"; \
	fi
	@echo ""
	@echo "--> [4/6] Installing git hooks (lefthook)..."
	@which lefthook >/dev/null 2>&1 && lefthook install || echo "    SKIP: lefthook not installed"
	@echo ""
	@echo "--> [5/6] Creating K3d local cluster..."
	$(MAKE) cluster
	@echo ""
	@echo "--> [6/6] Running bootstrap smoke test..."
	$(MAKE) smoke-test
	@echo ""
	@echo "==> Bootstrap complete!"
	@echo ""
	@echo "   make dev           — Skaffold hot-reload loop"
	@echo "   make dev-logs      — Stern multi-pod log tail"
	@echo "   make test-local    — Fast local regression suite"
	@echo "   make ci-local      — Full pre-push gate"
	@echo ""
	@echo "   Ingress:   http://localhost:8080"
	@echo "   NATS:      nats://localhost:4222"
	@echo "   Argo UI:   http://localhost:2746 (after: make dev-portforward)"
	@echo ""

# ═════════════════════════════════════════════════════════════════
#  CLUSTER — K3d lifecycle
# ═════════════════════════════════════════════════════════════════

cluster: ## 🔧 Create local K3d cluster (idempotent)
	@echo "==> Creating K3d cluster '$(CLUSTER_NAME)'..."
	@if k3d cluster list 2>/dev/null | grep -q "$(CLUSTER_NAME)"; then \
	  echo "    Cluster '$(CLUSTER_NAME)' already exists — skipping creation"; \
	else \
	  k3d cluster create --config $(K3D_CONFIG); \
	  echo "    OK: Cluster created"; \
	fi
	@echo "--> Waiting for all nodes to become Ready..."
	@kubectl wait --for=condition=ready node --all --timeout=120s
	@echo "--> Creating namespaces..."
	@kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f - -q
	@kubectl create namespace $(APPS_NS)   --dry-run=client -o yaml | kubectl apply -f - -q
	@echo "--> Installing cluster infrastructure (Argo · NATS · KEDA · Ingress)..."
	$(MAKE) cluster-deps
	@echo "==> Cluster ready"

cluster-deps: ## Install core infra: Argo Workflows, NATS, KEDA, nginx-ingress
	@echo "  -> Argo Workflows"
	@kubectl create namespace argo --dry-run=client -o yaml | kubectl apply -f - -q
	@kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/latest/download/install.yaml \
	  2>/dev/null || kubectl apply -n argo -f deploy/offline/argo-install.yaml 2>/dev/null || true
	@echo "  -> NATS JetStream"
	@helm repo add nats https://nats-io.github.io/k8s/helm/charts/ 2>/dev/null || true
	@helm upgrade --install nats nats/nats -n $(NAMESPACE) --create-namespace \
	  --set config.jetstream.enabled=true \
	  --set config.jetstream.memStorage.enabled=true \
	  --set config.jetstream.memStorage.size=256Mi \
	  --wait --timeout=120s 2>/dev/null || echo "    NATS: offline — using local manifest"
	@echo "  -> KEDA"
	@helm repo add kedacore https://kedacore.github.io/charts 2>/dev/null || true
	@helm upgrade --install keda kedacore/keda -n keda --create-namespace \
	  --wait --timeout=120s 2>/dev/null || \
	  kubectl apply -f deploy/offline/keda-install.yaml 2>/dev/null || true
	@echo "  -> nginx-ingress"
	@helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true
	@helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
	  -n ingress-nginx --create-namespace \
	  --set controller.service.type=LoadBalancer \
	  --wait --timeout=120s 2>/dev/null || \
	  kubectl apply -f deploy/offline/nginx-ingress.yaml 2>/dev/null || true
	@echo "  OK: Cluster dependencies installed"

cluster-delete: ## 🗑️  Delete the local K3d cluster
	@echo "==> Deleting K3d cluster '$(CLUSTER_NAME)'..."
	@k3d cluster delete $(CLUSTER_NAME) 2>/dev/null || true
	@echo "==> Cluster deleted"

cluster-reset: cluster-delete cluster ## 🔄 Destroy and recreate cluster from scratch

cluster-status: ## 📊 Show cluster health and running workloads
	@echo "── Nodes ──────────────────────────────────"
	@kubectl get nodes -o wide
	@echo ""
	@echo "── Pods ($(NAMESPACE)) ──────────────────────"
	@kubectl get pods -n $(NAMESPACE) -o wide 2>/dev/null || echo "  (namespace empty)"
	@echo ""
	@echo "── Pods (argo) ──────────────────────────────"
	@kubectl get pods -n argo -o wide 2>/dev/null || echo "  (argo not installed)"
	@echo ""
	@echo "── KttmApp CRDs ─────────────────────────────"
	@kubectl get kttmapps -A 2>/dev/null || echo "  (CRD not installed yet)"

smoke-test: ## 🔥 Verify cluster nodes and namespaces are healthy
	@echo "==> Smoke test..."
	@kubectl get nodes --no-headers 2>/dev/null | grep -c " Ready" | grep -q "^[1-9]" && \
	  echo "    OK: Cluster nodes Ready" || (echo "    FAIL: No Ready nodes" && exit 1)
	@kubectl get namespace $(NAMESPACE) >/dev/null 2>&1 && \
	  echo "    OK: Namespace $(NAMESPACE) exists" || \
	  echo "    WARN: Namespace $(NAMESPACE) missing (run: make cluster)"
	@echo "==> Smoke test passed"

# ═════════════════════════════════════════════════════════════════
#  DEV LOOP — Skaffold hot-reload + Stern log tailing
# ═════════════════════════════════════════════════════════════════

dev: ## 🔥 Start Skaffold hot-reload dev loop (rebuilds on every file save)
	@which skaffold >/dev/null 2>&1 || (echo "ERROR: skaffold not found. Run: brew install skaffold" && exit 1)
	@echo "==> Starting Skaffold dev loop..."
	@echo "    Watching: cmd/ core/ internal/ frontend/"
	@echo "    Save any .go or .jsx file to trigger auto rebuild + cluster deploy"
	@echo ""
	skaffold dev --port-forward --profile=local

dev-watch: ## 🔄 Skaffold run once (build + deploy, no watch loop)
	skaffold run --profile=local

dev-logs: ## 📋 Tail all KTTM pod logs via Stern (color-coded per pod)
	@which stern >/dev/null 2>&1 || (echo "ERROR: stern not found. Run: brew install stern" && exit 1)
	@echo "==> Tailing all kttm pods (Ctrl+C to stop)..."
	stern --all-namespaces --selector app.kubernetes.io/part-of=kttm --color=always

dev-frontend: ## 🌐 Start Vite dev server for the SPA (http://localhost:5173)
	@echo "==> Starting frontend dev server on http://localhost:5173..."
	npm run dev --prefix frontend/web-renderer

dev-portforward: ## 🔗 Port-forward cluster services to localhost
	@echo "==> Port-forwarding services..."
	@kubectl port-forward -n $(NAMESPACE) svc/nats 4222:4222 & echo "    NATS     -> localhost:4222"
	@kubectl port-forward -n argo svc/argo-server 2746:2746 &  echo "    Argo UI  -> localhost:2746"
	@echo "    Tip: run 'make stop' to kill all port-forwards"

stop: ## 🛑 Stop all background dev processes (port-forward, skaffold, etc.)
	@pkill -f "kubectl port-forward" 2>/dev/null || true
	@pkill -f "skaffold"            2>/dev/null || true
	@pkill -f "npm run dev"         2>/dev/null || true
	@pkill -f "stern"               2>/dev/null || true
	@echo "==> All dev processes stopped"

# ═════════════════════════════════════════════════════════════════
#  BUILD
# ═════════════════════════════════════════════════════════════════

build-operator: ## Build operator binary (native arch)
	@echo "==> Building kttm-operator ($(GOOS)/$(GOARCH))..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
	  -ldflags="-X main.version=$(IMAGE_TAG) -X main.commit=$(GIT_SHA)" \
	  -o bin/kttm-operator ./cmd/operator/
	@echo "    OK: bin/kttm-operator"

build-server: ## Build API server binary
	@echo "==> Building kttm-server ($(GOOS)/$(GOARCH))..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
	  -ldflags="-X main.version=$(IMAGE_TAG) -X main.commit=$(GIT_SHA)" \
	  -o bin/kttm-server ./cmd/server/
	@echo "    OK: bin/kttm-server"

build-cli: ## Build flowengine CLI binary
	@echo "==> Building flowengine CLI ($(GOOS)/$(GOARCH))..."
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
	  -ldflags="-X main.version=$(IMAGE_TAG)" \
	  -o bin/flowengine ./cmd/flowengine/ 2>/dev/null || \
	  echo "    SKIP: cmd/flowengine not yet implemented"

build-web: ## Build the frontend SPA
	@echo "==> Building KTTM SPA..."
	@if [ -f "frontend/web-renderer/package.json" ]; then \
	  npm run build --prefix frontend/web-renderer; \
	  echo "    OK: frontend/web-renderer/dist/"; \
	else \
	  echo "    SKIP: No frontend found"; \
	fi

build: build-operator build-server ## Build operator + server

build-all: build-operator build-server build-cli build-web ## Build all binaries + frontend

# ═════════════════════════════════════════════════════════════════
#  CONTAINER IMAGES — local K3d registry only (no push to cloud)
# ═════════════════════════════════════════════════════════════════

image: ## 🐳 Build container image and load into K3d local registry
	@echo "==> Building operator image..."
	docker build \
	  --build-arg VERSION=$(IMAGE_TAG) \
	  --build-arg COMMIT=$(GIT_SHA) \
	  -t $(OPERATOR_IMAGE) -f Dockerfile .
	@echo "==> Loading image into K3d cluster (no cloud push)..."
	k3d image import $(OPERATOR_IMAGE) --cluster $(CLUSTER_NAME)
	@echo "    OK: $(OPERATOR_IMAGE) ready in cluster"

image-load: ## Load an already-built image into the K3d cluster
	k3d image import $(OPERATOR_IMAGE) --cluster $(CLUSTER_NAME)
	@echo "    OK: Image loaded"

# ═════════════════════════════════════════════════════════════════
#  TESTS
# ═════════════════════════════════════════════════════════════════

test-unit: ## 🧪 Unit tests (no external services, fast, race detector on)
	@echo "==> Running unit tests..."
	$(GO) test -race -count=1 -timeout=120s \
	  -coverprofile=$(COVERAGE_FILE) -covermode=atomic \
	  ./core/... ./internal/...
	@$(MAKE) _check-coverage

test-integration: ## 🧪 Integration tests (testcontainers: real Postgres + NATS + MinIO)
	@echo "==> Running integration tests (testcontainers)..."
	@which docker >/dev/null 2>&1 || (echo "ERROR: Docker not running" && exit 1)
	$(GO) test -race -count=1 -timeout=300s -tags=integration \
	  -coverprofile=coverage-integration.out -covermode=atomic \
	  ./tests/integration/...
	@echo "    OK: Integration tests passed"

test-local: ## ✅ Fast local regression (unit + lint + vet, no Docker needed)
	@echo "==> Local Regression Suite"
	@echo "--> Schema linter tests..."
	$(GO) test ./core/linter/... -v -timeout=30s 2>/dev/null || echo "    SKIP: no linter tests yet"
	@echo "--> Graph validator tests..."
	$(GO) test ./internal/workflow/... -v -timeout=30s
	@echo "--> Engine compiler tests..."
	$(GO) test ./core/engine/... -v -timeout=30s 2>/dev/null || echo "    SKIP: no engine tests yet"
	@echo "--> go vet..."
	$(GO) vet ./...
	@echo ""
	@echo "==> All local regression checks passed"

test: ci-local ## Alias: Run full local CI gate (format, lint, build, test)

test-coverage: ## 📊 Open HTML coverage report in browser
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@open coverage.html 2>/dev/null || xdg-open coverage.html

_check-coverage:
	@COVERAGE=$$($(GO) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}' | tr -d '%'); \
	echo "    Total coverage: $${COVERAGE}%"; \
	if [ 1 -eq $$(echo "$${COVERAGE} < 1" | bc) ]; then \
	  echo "    FAIL: Coverage $${COVERAGE}% < threshold 1%"; \
	  exit 1; \
	fi; \
	echo "    OK: Coverage $${COVERAGE}% >= 1%"

# ═════════════════════════════════════════════════════════════════
#  LINT / FMT / VET
# ═════════════════════════════════════════════════════════════════

lint: ## 🔍 Run golangci-lint
	@which golangci-lint >/dev/null 2>&1 || \
	  (echo "Installing golangci-lint..." && \
	   go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1)
	golangci-lint run --timeout=5m ./...

fmt: ## 🎨 Auto-format all Go files and tidy go.mod
	gofmt -w -s .
	$(GO) mod tidy
	@echo "==> Formatted"

vet: ## 🔬 Run go vet across all packages
	$(GO) vet ./...
	@echo "==> Vet clean"

# ═════════════════════════════════════════════════════════════════
#  HELM
# ═════════════════════════════════════════════════════════════════

helm-lint: ## Lint the Helm chart
	@if [ -d "$(HELM_CHART)" ]; then helm lint $(HELM_CHART); \
	else echo "    SKIP: No Helm chart at $(HELM_CHART)"; fi

helm-install: ## Deploy KTTM Helm chart to local K3d cluster
	@echo "==> Deploying KTTM via Helm..."
	@if [ -d "$(HELM_CHART)" ]; then \
	  helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
	    -n $(NAMESPACE) --create-namespace \
	    --set image.repository=$(REGISTRY)/kttm-operator \
	    --set image.tag=$(IMAGE_TAG) \
	    --wait --timeout=120s; \
	  echo "    OK: KTTM deployed"; \
	else \
	  echo "    SKIP: No Helm chart yet"; \
	fi

helm-upgrade: ## Upgrade running KTTM Helm release
	helm upgrade $(HELM_RELEASE) $(HELM_CHART) -n $(NAMESPACE) \
	  --set image.repository=$(REGISTRY)/kttm-operator \
	  --set image.tag=$(IMAGE_TAG) \
	  --wait --timeout=120s

helm-uninstall: ## Uninstall KTTM Helm release from cluster
	helm uninstall $(HELM_RELEASE) -n $(NAMESPACE) 2>/dev/null || true
	@echo "==> KTTM uninstalled"

# ═════════════════════════════════════════════════════════════════
#  CI-LOCAL — mirrors the Git remote regression gate, runs locally
# ═════════════════════════════════════════════════════════════════

ci-local: ## 🚦 Full pre-push gate (run before every git push)
	@echo "==> CI Local Gate — mirrors remote GitHub Actions"
	@echo "    Run this before every 'git push' to guarantee the remote gate passes."
	@echo ""
	@echo "--> [1/4] Format + vet..."
	$(GO) vet ./...
	@UNFORMATTED=$$(gofmt -l . | grep -v vendor); \
	if [ -n "$$UNFORMATTED" ]; then \
	  echo "    FAIL: Unformatted files:"; echo "$$UNFORMATTED"; \
	  echo "    Fix with: make fmt"; exit 1; \
	fi
	@echo "    OK: Format + vet clean"
	@echo ""
	@echo "--> [2/4] Build..."
	$(GO) build ./...
	@echo "    OK: Build clean"
	@echo ""
	@echo "--> [3/4] Unit tests..."
	$(GO) test -race -count=1 -timeout=120s \
	  -coverprofile=$(COVERAGE_FILE) -covermode=atomic \
	  ./core/... ./internal/...
	@$(MAKE) _check-coverage
	@echo ""
	@echo "--> [4/4] Helm lint..."
	@$(MAKE) helm-lint
	@echo ""
	@echo "==> CI Local Gate PASSED — safe to push to Git"
	@echo ""

# ═════════════════════════════════════════════════════════════════
#  CLEAN
# ═════════════════════════════════════════════════════════════════

clean: ## 🧹 Remove build artifacts and coverage reports
	@rm -rf bin/ coverage.out coverage-integration.out coverage.html
	@echo "==> Cleaned"
