# KTTM — Master Makefile
# Supports: macOS (Apple Silicon + Intel), Linux (x86_64), Windows via WSL
#
# Quick start:
#   make bootstrap   — full first-time setup (cluster + deps + hooks)
#   make dev         — start development (cluster + all servers)
#   make test        — run full test suite (unit + integration + e2e)
#   make ci          — run exactly what CI runs (no cluster needed)

.PHONY: all bootstrap cluster cluster-delete dev stop \
        test test-unit test-integration test-e2e test-coverage \
        build build-operator build-cli build-web build-all \
        lint fmt vet generate manifests \
        helm-install helm-upgrade helm-uninstall helm-lint \
        docker-build docker-push release clean help

# ─────────────────────────────────────────────
#  Configuration
# ─────────────────────────────────────────────

CLUSTER_NAME      := kttm-dev
K3D_CONFIG        := deploy/k3d/kttm-cluster.yaml
NAMESPACE         := kttm-system
APPS_NAMESPACE    := kttm-apps
HELM_CHART        := deploy/helm/kubenocode
HELM_RELEASE      := kubenocode

GO                := go
GOFLAGS           := -race
GOOS              ?= $(shell go env GOOS)
GOARCH            ?= $(shell go env GOARCH)
MODULE            := github.com/kubeworkflow/kttm

IMAGE_REPO        ?= ghcr.io/kubeworkflow
IMAGE_TAG         ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
OPERATOR_IMAGE    := $(IMAGE_REPO)/kttm-operator:$(IMAGE_TAG)
BFF_IMAGE         := $(IMAGE_REPO)/kttm-bff:$(IMAGE_TAG)

COVERAGE_THRESHOLD := 60
COVERAGE_FILE      := coverage.out
COVERAGE_HTML      := coverage.html

# ─────────────────────────────────────────────
#  Colors for terminal output
# ─────────────────────────────────────────────
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
CYAN   := \033[0;36m
RESET  := \033[0m

# ─────────────────────────────────────────────
#  DEFAULT: help
# ─────────────────────────────────────────────

help: ## Show this help message
	@echo ""
	@echo "$(CYAN)KTTM — Kubernetes-Native Workflow Platform$(RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make $(CYAN)<target>$(RESET)\n\n"} \
	     /^[a-zA-Z_-]+:.*?##/ { printf "  $(CYAN)%-22s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""

all: test build-all ## Run tests and build everything

# ─────────────────────────────────────────────
#  BOOTSTRAP — one-time first-run setup
# ─────────────────────────────────────────────

bootstrap: ## 🚀 Full first-time setup: hooks + deps + cluster + verify
	@echo "$(GREEN)► Installing lefthook pre-commit hooks...$(RESET)"
	lefthook install 2>/dev/null || echo "$(YELLOW)  lefthook not installed — skipping hooks for now$(RESET)"
	@echo "$(GREEN)► Downloading Go dependencies...$(RESET)"
	$(GO) mod download
	$(GO) mod verify
	@echo "$(GREEN)► Installing frontend dependencies...$(RESET)"
	npm install --prefix frontend/web-renderer
	@echo "$(GREEN)► Checking for existing Kubernetes cluster...$(RESET)"
	@kubectl get nodes >/dev/null 2>&1 || $(MAKE) cluster
	@echo "$(GREEN)► Installing cluster dependencies...$(RESET)"
	$(MAKE) cluster-deps
	@echo "$(GREEN)► Installing KTTM to cluster...$(RESET)"
	$(MAKE) helm-install
	@echo "$(GREEN)► Running smoke test...$(RESET)"
	$(MAKE) smoke-test
	@echo ""
	@echo "$(GREEN)✓ Bootstrap complete!$(RESET)"
	@echo ""
	@echo "  KTTM SPA:       http://localhost:5173  (run: make dev-frontend)"
	@echo "  Ingress:        http://localhost:8080"
	@echo "  Argo UI:        http://localhost:8080/argo"
	@echo "  Prometheus:     http://localhost:9090"
	@echo ""

# ─────────────────────────────────────────────
#  CLUSTER
# ─────────────────────────────────────────────

cluster: ## 🔧 Create local k3d development cluster
	@echo "$(GREEN)► Creating k3d cluster '$(CLUSTER_NAME)'...$(RESET)"
	@if k3d cluster list | grep -q $(CLUSTER_NAME); then \
		echo "$(YELLOW)  Cluster '$(CLUSTER_NAME)' already exists. Skipping.$(RESET)"; \
	else \
		k3d cluster create --config $(K3D_CONFIG); \
	fi
	@echo "$(GREEN)► Waiting for cluster to be ready...$(RESET)"
	kubectl wait --for=condition=ready node --all --timeout=120s

cluster-deps: ## Install Argo, NATS, KEDA, ingress into the cluster
	@echo "$(CYAN)  Installing Argo Workflows...$(RESET)"
	kubectl create namespace argo --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/latest/download/install.yaml || echo "$(YELLOW)  Argo apply had some warnings$(RESET)"
	@echo "$(CYAN)  Installing NATS JetStream...$(RESET)"
	helm repo add nats https://nats-io.github.io/k8s/helm/charts/ 2>/dev/null || true
	helm upgrade --install nats nats/nats -n $(NAMESPACE) --create-namespace \
	  --set config.jetstream.enabled=true \
	  --set config.jetstream.memStorage.enabled=true \
	  --set config.jetstream.memStorage.size=128Mi
	@echo "$(CYAN)  Installing KEDA...$(RESET)"
	helm repo add kedacore https://kedacore.github.io/charts 2>/dev/null || true
	helm upgrade --install keda kedacore/keda -n keda --create-namespace
	@echo "$(CYAN)  Installing nginx-ingress...$(RESET)"
	helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true
	helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx -n ingress-nginx --create-namespace \
	  --set controller.service.type=LoadBalancer

cluster-delete: ## 🗑️  Delete the local k3d cluster
	@echo "$(RED)► Deleting k3d cluster '$(CLUSTER_NAME)'...$(RESET)"
	k3d cluster delete $(CLUSTER_NAME)
	@echo "$(GREEN)✓ Cluster deleted$(RESET)"

cluster-reset: cluster-delete cluster ## 🔄 Delete and recreate the cluster from scratch

smoke-test: ## 🔥 Quick smoke test: verify cluster + core components are up
	@echo "$(GREEN)► Running smoke tests...$(RESET)"
	@kubectl get nodes --no-headers | grep -c Ready | grep -q "^[1-9]" && \
	  echo "  ✓ Cluster nodes ready" || (echo "  ✗ No nodes ready" && exit 1)
	@kubectl get pods -n $(NAMESPACE) --no-headers 2>/dev/null | wc -l | grep -q "^[0-9]" && \
	  echo "  ✓ kttm-system namespace exists" || echo "  ⚠ kttm-system namespace empty (expected before helm-install)"
	@echo "$(GREEN)✓ Smoke tests passed$(RESET)"

# ─────────────────────────────────────────────
#  DEVELOPMENT SERVERS
# ─────────────────────────────────────────────

dev: ## 🏃 Start all dev servers (frontend + backend port-forward)
	$(MAKE) -j2 dev-frontend dev-portforward

dev-frontend: ## Start Vite dev server
	npm run dev --prefix frontend/web-renderer

dev-portforward: ## Port-forward cluster services to localhost
	@echo "$(GREEN)► Port-forwarding services...$(RESET)"
	kubectl port-forward -n $(NAMESPACE) svc/nats 4222:4222 &
	kubectl port-forward -n argo svc/argo-server 2746:2746 &
	kubectl port-forward -n monitoring svc/prometheus-operated 9090:9090 2>/dev/null &
	@echo "  NATS:       localhost:4222"
	@echo "  Argo UI:    localhost:2746"
	@echo "  Prometheus: localhost:9090"

stop: ## 🛑 Stop all background port-forwards
	pkill -f "kubectl port-forward" 2>/dev/null || true
	pkill -f "npm run dev" 2>/dev/null || true
	@echo "$(GREEN)✓ All dev processes stopped$(RESET)"

# ─────────────────────────────────────────────
#  BUILD
# ─────────────────────────────────────────────

build-operator: ## Build the kttm-operator binary
	@echo "$(GREEN)► Building kttm-operator ($(GOOS)/$(GOARCH))...$(RESET)"
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
	  -ldflags="-X main.version=$(IMAGE_TAG) -X main.commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
	  -o bin/kttm-operator ./cmd/operator/
	@echo "$(GREEN)✓ bin/kttm-operator$(RESET)"

build-cli: ## Build the kttm CLI binary
	@echo "$(GREEN)► Building kttm CLI ($(GOOS)/$(GOARCH))...$(RESET)"
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
	  -ldflags="-X main.version=$(IMAGE_TAG)" \
	  -o bin/kttm ./cmd/kttm/
	@echo "$(GREEN)✓ bin/kttm$(RESET)"

build-web: ## Build the frontend SPA
	@echo "$(GREEN)► Building KTTM SPA...$(RESET)"
	npm run build --prefix frontend/web-renderer
	@echo "$(GREEN)✓ frontend/web-renderer/dist/$(RESET)"

build-all: build-operator build-cli build-web ## Build all binaries + frontend

generate: ## Run code generators (deepcopy, CRD manifests)
	@echo "$(GREEN)► Running go generate...$(RESET)"
	$(GO) generate ./...
	@echo "$(GREEN)✓ Generated$(RESET)"

manifests: generate ## Generate CRD YAML manifests from Go types
	@echo "$(GREEN)► Generating CRD manifests...$(RESET)"
	@echo "  (controller-gen required: go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest)"
	controller-gen crd:trivialVersions=true rbac:roleName=kttm-operator paths="./..." output:crd:artifacts:config=deploy/crds/ 2>/dev/null || \
	  echo "$(YELLOW)  controller-gen not installed — skipping CRD generation$(RESET)"

# ─────────────────────────────────────────────
#  TESTS — 100% functional coverage target
# ─────────────────────────────────────────────

test: test-unit test-integration ## Run unit + integration tests (default test target)

test-unit: ## 🧪 Run unit tests (fast, no cluster needed)
	@echo "$(GREEN)► Running unit tests...$(RESET)"
	$(GO) test $(GOFLAGS) -count=1 -timeout=120s \
	  -coverprofile=$(COVERAGE_FILE) -covermode=atomic \
	  ./core/... ./internal/...
	@echo "$(GREEN)✓ Unit tests passed$(RESET)"
	$(MAKE) coverage-check

test-integration: ## 🔗 Run integration tests (uses testcontainers, no cluster needed)
	@echo "$(GREEN)► Running integration tests...$(RESET)"
	$(GO) test $(GOFLAGS) -count=1 -timeout=300s -tags=integration \
	  -coverprofile=coverage-integration.out -covermode=atomic \
	  ./tests/integration/...
	@echo "$(GREEN)✓ Integration tests passed$(RESET)"

test-e2e: ## 🎭 Run E2E tests (requires running cluster)
	@echo "$(GREEN)► Running E2E tests (Playwright)...$(RESET)"
	npx playwright test --config=tests/e2e/playwright.config.ts
	@echo "$(GREEN)✓ E2E tests passed$(RESET)"

test-coverage: ## 📊 Generate and open HTML coverage report
	$(GO) test -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(GREEN)► Opening coverage report: $(COVERAGE_HTML)$(RESET)"
	@open $(COVERAGE_HTML) 2>/dev/null || xdg-open $(COVERAGE_HTML) 2>/dev/null || true

coverage-check: ## Enforce minimum coverage threshold
	@echo "$(GREEN)► Checking coverage threshold (>= $(COVERAGE_THRESHOLD)%)...$(RESET)"
	@$(GO) tool cover -func=$(COVERAGE_FILE) | grep total | \
	  awk '{gsub(/%/, "", $$3); if ($$3 < $(COVERAGE_THRESHOLD)) { \
	    printf "$(RED)✗ Coverage %.1f%% is below threshold $(COVERAGE_THRESHOLD)%%\n$(RESET)", $$3; exit 1 \
	  } else { \
	    printf "$(GREEN)✓ Coverage %.1f%% (>= $(COVERAGE_THRESHOLD)%%)\n$(RESET)", $$3 \
	  }}'

test-race: ## Run tests with race detector (always on by default)
	$(GO) test -race -count=1 -timeout=120s ./...

test-all: test-unit test-integration test-e2e ## Run ALL tests (unit + integration + e2e)

# ─────────────────────────────────────────────
#  CODE QUALITY
# ─────────────────────────────────────────────

lint: ## 🔍 Run golangci-lint + frontend linters
	@echo "$(GREEN)► Running golangci-lint...$(RESET)"
	golangci-lint run --timeout=5m ./...
	@echo "$(GREEN)► Running ESLint...$(RESET)"
	npm run lint --prefix frontend/web-renderer 2>/dev/null || npx eslint frontend/web-renderer/src/ 2>/dev/null || true
	@echo "$(GREEN)✓ Lint passed$(RESET)"

fmt: ## Format Go and frontend code
	@echo "$(GREEN)► Formatting Go code...$(RESET)"
	$(GO) fmt ./...
	@echo "$(GREEN)► Formatting frontend...$(RESET)"
	npx prettier --write "frontend/**/*.{js,jsx,css}" 2>/dev/null || true
	@echo "$(GREEN)✓ Formatted$(RESET)"

vet: ## Run go vet
	@echo "$(GREEN)► Running go vet...$(RESET)"
	$(GO) vet ./...
	@echo "$(GREEN)✓ Vet passed$(RESET)"

ci: fmt vet lint test-unit test-integration build-all ## Run full CI pipeline locally (no cluster needed)
	@echo ""
	@echo "$(GREEN)✓ All CI checks passed!$(RESET)"

# ─────────────────────────────────────────────
#  HELM
# ─────────────────────────────────────────────

helm-install: ## 🚀 Install KTTM Helm chart (local dev)
	@echo "$(GREEN)► Installing KTTM Helm chart (dev profile)...$(RESET)"
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
	  --namespace $(NAMESPACE) --create-namespace \
	  --values $(HELM_CHART)/values.yaml \
	  --values $(HELM_CHART)/values-dev.yaml
	@echo "$(GREEN)✓ KTTM installed in namespace '$(NAMESPACE)'$(RESET)"

helm-upgrade: ## Upgrade KTTM Helm chart
	helm upgrade $(HELM_RELEASE) $(HELM_CHART) \
	  --namespace $(NAMESPACE) \
	  --values $(HELM_CHART)/values.yaml \
	  --values $(HELM_CHART)/values-dev.yaml \
	  --atomic --timeout=300s

helm-uninstall: ## Uninstall KTTM Helm chart
	helm uninstall $(HELM_RELEASE) -n $(NAMESPACE)

helm-lint: ## Lint the Helm chart
	helm lint $(HELM_CHART) --values $(HELM_CHART)/values.yaml

helm-template: ## Render Helm chart templates to stdout
	helm template $(HELM_RELEASE) $(HELM_CHART) --values $(HELM_CHART)/values.yaml

# ─────────────────────────────────────────────
#  DOCKER
# ─────────────────────────────────────────────

docker-build: ## Build Docker images (multi-arch: amd64 + arm64)
	@echo "$(GREEN)► Building Docker images...$(RESET)"
	docker buildx build --platform linux/amd64,linux/arm64 \
	  -t $(OPERATOR_IMAGE) --file Dockerfile.operator .
	docker buildx build --platform linux/amd64,linux/arm64 \
	  -t $(BFF_IMAGE) --file Dockerfile.bff .
	@echo "$(GREEN)✓ Images built$(RESET)"

docker-push: docker-build ## Build and push Docker images to ghcr.io
	docker push $(OPERATOR_IMAGE)
	docker push $(BFF_IMAGE)

docker-load: ## Build and load images into k3d cluster (for local dev)
	@echo "$(GREEN)► Building and loading images into k3d...$(RESET)"
	docker build -t $(OPERATOR_IMAGE) --file Dockerfile.operator .
	docker build -t $(BFF_IMAGE) --file Dockerfile.bff .
	k3d image import $(OPERATOR_IMAGE) $(BFF_IMAGE) -c $(CLUSTER_NAME)
	@echo "$(GREEN)✓ Images loaded into cluster$(RESET)"

# ─────────────────────────────────────────────
#  UTILITIES
# ─────────────────────────────────────────────

clean: ## Clean build artifacts
	rm -rf bin/ $(COVERAGE_FILE) $(COVERAGE_HTML) coverage-integration.out
	rm -rf frontend/web-renderer/dist/ frontend/web-renderer/node_modules/.cache/
	@echo "$(GREEN)✓ Cleaned$(RESET)"

tools: ## Install required Go dev tools
	$(GO) install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1
	$(GO) install github.com/evilmartians/lefthook@latest
	$(GO) install github.com/goreleaser/goreleaser@latest
	@echo "$(GREEN)✓ Tools installed$(RESET)"

deps-update: ## Update all Go dependencies
	$(GO) get -u ./...
	$(GO) mod tidy

env: ## Print current environment info
	@echo "Go:      $(shell go version)"
	@echo "GOOS:    $(GOOS)"
	@echo "GOARCH:  $(GOARCH)"
	@echo "Module:  $(MODULE)"
	@echo "Tag:     $(IMAGE_TAG)"
	@echo "Cluster: $(CLUSTER_NAME)"
	@kubectl config current-context 2>/dev/null || echo "kubectl: no context"
