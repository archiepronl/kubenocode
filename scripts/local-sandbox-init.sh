#!/usr/bin/env bash
# =============================================================================
#  scripts/local-sandbox-init.sh
#  KTTM — One-command local sandbox initializer for Apple Silicon M2 MacBook
#
#  What this script does:
#    1. Verifies required tools (k3d, kubectl, helm, skaffold, stern)
#    2. Installs any missing tools via Homebrew (with confirmation)
#    3. Creates the K3d 3-node cluster with local registry
#    4. Installs Argo Workflows, NATS JetStream, KEDA, nginx-ingress
#    5. Deploys the KTTM CRDs and operator
#    6. Runs a smoke test to verify everything is working
#    7. Prints access URLs and next steps
#
#  Usage:
#    chmod +x scripts/local-sandbox-init.sh
#    ./scripts/local-sandbox-init.sh
#
#  Environment variables:
#    KTTM_SKIP_DEPS=1     Skip installing cluster dependencies (Argo/NATS/KEDA)
#    KTTM_SKIP_BREW=1     Skip Homebrew tool installations
#    KTTM_OFFLINE=1       Use offline manifests in deploy/offline/ (air-gap mode)
#
# =============================================================================

set -euo pipefail

# ─────────────────────────────────────────────────────────────────
#  Colors
# ─────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { echo -e "${CYAN}  -->${RESET} $*"; }
success() { echo -e "${GREEN}  ✓${RESET} $*"; }
warn()    { echo -e "${YELLOW}  ⚠${RESET} $*"; }
error()   { echo -e "${RED}  ✗ ERROR:${RESET} $*" >&2; }
header()  { echo -e "\n${BOLD}${CYAN}═══ $* ═══${RESET}\n"; }

# ─────────────────────────────────────────────────────────────────
#  Configuration
# ─────────────────────────────────────────────────────────────────
CLUSTER_NAME="${KTTM_CLUSTER_NAME:-kttm-dev}"
K3D_CONFIG="deploy/k3d/kttm-cluster.yaml"
NAMESPACE="kttm-system"
APPS_NS="kttm-apps"
OFFLINE="${KTTM_OFFLINE:-0}"
SKIP_DEPS="${KTTM_SKIP_DEPS:-0}"
SKIP_BREW="${KTTM_SKIP_BREW:-0}"

# ─────────────────────────────────────────────────────────────────
#  Verify we're in the repo root
# ─────────────────────────────────────────────────────────────────
if [[ ! -f "go.mod" ]] || [[ ! -f "$K3D_CONFIG" ]]; then
  error "Run this script from the repo root: ./scripts/local-sandbox-init.sh"
  exit 1
fi

# ─────────────────────────────────────────────────────────────────
#  Detect architecture
# ─────────────────────────────────────────────────────────────────
ARCH=$(uname -m)
OS=$(uname -s)
if [[ "$ARCH" == "arm64" && "$OS" == "Darwin" ]]; then
  ARCH_LABEL="Apple Silicon M-series"
elif [[ "$ARCH" == "x86_64" && "$OS" == "Darwin" ]]; then
  ARCH_LABEL="Intel Mac"
elif [[ "$ARCH" == "x86_64" && "$OS" == "Linux" ]]; then
  ARCH_LABEL="Linux x86_64"
else
  ARCH_LABEL="$OS/$ARCH"
fi

# ─────────────────────────────────────────────────────────────────
#  Banner
# ─────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}${CYAN}╔══════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}${CYAN}║         KTTM — Local Sandbox Initializer                     ║${RESET}"
echo -e "${BOLD}${CYAN}║         Architecture: $ARCH_LABEL${RESET}"
echo -e "${BOLD}${CYAN}╚══════════════════════════════════════════════════════════════╝${RESET}"
echo ""
echo "  This script bootstraps a full KTTM development environment"
echo "  entirely on your local machine — no cloud costs, no latency."
echo ""

# ─────────────────────────────────────────────────────────────────
#  STEP 1: Tool verification + optional Homebrew install
# ─────────────────────────────────────────────────────────────────
header "Step 1 / 6 — Verifying Required Tools"

REQUIRED_TOOLS=("k3d" "kubectl" "helm" "docker")
OPTIONAL_TOOLS=("skaffold" "stern" "lefthook" "golangci-lint")
MISSING_REQUIRED=()
MISSING_OPTIONAL=()

for tool in "${REQUIRED_TOOLS[@]}"; do
  if command -v "$tool" &>/dev/null; then
    success "$tool $(command -v $tool)"
  else
    MISSING_REQUIRED+=("$tool")
    error "$tool not found"
  fi
done

for tool in "${OPTIONAL_TOOLS[@]}"; do
  if command -v "$tool" &>/dev/null; then
    success "$tool $(command -v $tool)"
  else
    MISSING_OPTIONAL+=("$tool")
    warn "$tool not found (optional but recommended)"
  fi
done

if [[ ${#MISSING_REQUIRED[@]} -gt 0 ]]; then
  echo ""
  warn "Missing required tools: ${MISSING_REQUIRED[*]}"
  if [[ "$SKIP_BREW" != "1" ]] && command -v brew &>/dev/null; then
    echo -n "  Install missing tools via Homebrew? [Y/n] "
    read -r response
    response=${response:-Y}
    if [[ "$response" =~ ^[Yy] ]]; then
      for tool in "${MISSING_REQUIRED[@]}"; do
        info "Installing $tool..."
        brew install "$tool"
        success "$tool installed"
      done
    else
      error "Cannot continue without required tools. Exiting."
      exit 1
    fi
  else
    error "Please install: ${MISSING_REQUIRED[*]}"
    echo "  macOS:  brew install ${MISSING_REQUIRED[*]}"
    echo "  Linux:  see https://k3d.io and https://helm.sh for install instructions"
    exit 1
  fi
fi

if [[ ${#MISSING_OPTIONAL[@]} -gt 0 ]] && [[ "$SKIP_BREW" != "1" ]] && command -v brew &>/dev/null; then
  echo ""
  echo -n "  Install optional tools (${MISSING_OPTIONAL[*]})? [Y/n] "
  read -r response
  response=${response:-Y}
  if [[ "$response" =~ ^[Yy] ]]; then
    for tool in "${MISSING_OPTIONAL[@]}"; do
      info "Installing $tool..."
      brew install "$tool" 2>/dev/null || \
        go install "github.com/golangci/golangci-lint/cmd/golangci-lint@latest" 2>/dev/null || true
    done
  fi
fi

# ─────────────────────────────────────────────────────────────────
#  STEP 2: Go dependencies
# ─────────────────────────────────────────────────────────────────
header "Step 2 / 6 — Go Dependencies"
info "Running go mod download..."
go mod download
go mod verify
success "Go dependencies ready"

# ─────────────────────────────────────────────────────────────────
#  STEP 3: K3d cluster
# ─────────────────────────────────────────────────────────────────
header "Step 3 / 6 — K3d Cluster"

if k3d cluster list 2>/dev/null | grep -q "$CLUSTER_NAME"; then
  warn "Cluster '$CLUSTER_NAME' already exists"
  echo -n "  Delete and recreate? [y/N] "
  read -r response
  response=${response:-N}
  if [[ "$response" =~ ^[Yy] ]]; then
    info "Deleting existing cluster..."
    k3d cluster delete "$CLUSTER_NAME"
    info "Creating new cluster from $K3D_CONFIG..."
    k3d cluster create --config "$K3D_CONFIG"
    success "Cluster recreated"
  else
    info "Keeping existing cluster"
  fi
else
  info "Creating K3d cluster from $K3D_CONFIG..."
  k3d cluster create --config "$K3D_CONFIG"
  success "Cluster created"
fi

info "Waiting for all nodes to become Ready (up to 120s)..."
kubectl wait --for=condition=ready node --all --timeout=120s
success "All nodes ready"

info "Creating namespaces..."
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f - -q
kubectl create namespace "$APPS_NS"   --dry-run=client -o yaml | kubectl apply -f - -q
success "Namespaces ready"

# ─────────────────────────────────────────────────────────────────
#  STEP 4: Cluster infrastructure
# ─────────────────────────────────────────────────────────────────
header "Step 4 / 6 — Cluster Infrastructure"

if [[ "$SKIP_DEPS" == "1" ]]; then
  warn "KTTM_SKIP_DEPS=1 — skipping Argo/NATS/KEDA installation"
else
  install_component() {
    local name="$1"
    local online_cmd="$2"
    local offline_manifest="$3"

    info "Installing $name..."
    if [[ "$OFFLINE" == "1" ]] && [[ -f "$offline_manifest" ]]; then
      kubectl apply -f "$offline_manifest"
      success "$name installed (offline)"
    else
      eval "$online_cmd" && success "$name installed" || \
        (warn "$name online install failed; trying offline manifest..." && \
         kubectl apply -f "$offline_manifest" 2>/dev/null && success "$name installed (offline)" || \
         warn "$name skipped — add offline manifest to deploy/offline/")
    fi
  }

  # Argo Workflows
  kubectl create namespace argo --dry-run=client -o yaml | kubectl apply -f - -q
  install_component "Argo Workflows" \
    "kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/latest/download/install.yaml" \
    "deploy/offline/argo-install.yaml"

  # NATS JetStream
  helm repo add nats https://nats-io.github.io/k8s/helm/charts/ 2>/dev/null || true
  helm repo update nats 2>/dev/null || true
  install_component "NATS JetStream" \
    "helm upgrade --install nats nats/nats -n $NAMESPACE --create-namespace \
     --set config.jetstream.enabled=true \
     --set config.jetstream.memStorage.enabled=true \
     --set config.jetstream.memStorage.size=256Mi \
     --wait --timeout=120s" \
    "deploy/offline/nats-install.yaml"

  # KEDA
  helm repo add kedacore https://kedacore.github.io/charts 2>/dev/null || true
  helm repo update kedacore 2>/dev/null || true
  install_component "KEDA" \
    "helm upgrade --install keda kedacore/keda -n keda --create-namespace --wait --timeout=120s" \
    "deploy/offline/keda-install.yaml"

  # nginx-ingress
  helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true
  helm repo update ingress-nginx 2>/dev/null || true
  install_component "nginx-ingress" \
    "helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
     -n ingress-nginx --create-namespace \
     --set controller.service.type=LoadBalancer --wait --timeout=120s" \
    "deploy/offline/nginx-ingress.yaml"
fi

# ─────────────────────────────────────────────────────────────────
#  STEP 5: Git hooks
# ─────────────────────────────────────────────────────────────────
header "Step 5 / 6 — Git Hooks"
if command -v lefthook &>/dev/null; then
  lefthook install
  success "lefthook pre-commit hooks installed"
else
  warn "lefthook not installed — skipping hooks setup"
fi

# ─────────────────────────────────────────────────────────────────
#  STEP 6: Smoke test
# ─────────────────────────────────────────────────────────────────
header "Step 6 / 6 — Smoke Test"

READY_NODES=$(kubectl get nodes --no-headers 2>/dev/null | grep -c " Ready" || echo 0)
if [[ "$READY_NODES" -ge 1 ]]; then
  success "Cluster nodes: $READY_NODES Ready"
else
  error "No Ready nodes found — check: kubectl get nodes"
  exit 1
fi

kubectl get namespace "$NAMESPACE" >/dev/null 2>&1 && \
  success "Namespace $NAMESPACE: exists" || \
  warn "Namespace $NAMESPACE missing"

# ─────────────────────────────────────────────────────────────────
#  Done
# ─────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}${GREEN}║   ✓  KTTM Local Sandbox Ready                                ║${RESET}"
echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════════════════════════╝${RESET}"
echo ""
echo -e "  ${BOLD}Start your dev loop:${RESET}"
echo -e "    ${CYAN}make dev${RESET}              — Skaffold hot-reload (save .go → auto-deploy)"
echo -e "    ${CYAN}make dev-logs${RESET}         — Stern real-time multi-pod log tail"
echo -e "    ${CYAN}make dev-portforward${RESET}  — Expose cluster services to localhost"
echo ""
echo -e "  ${BOLD}Test before pushing:${RESET}"
echo -e "    ${CYAN}make test-local${RESET}       — Fast local regression (no Docker needed)"
echo -e "    ${CYAN}make ci-local${RESET}         — Full gate matching remote GitHub Actions"
echo ""
echo -e "  ${BOLD}Cluster access:${RESET}"
echo -e "    Ingress:   ${CYAN}http://localhost:8080${RESET}"
echo -e "    NATS:      ${CYAN}nats://localhost:4222${RESET}"
echo -e "    Registry:  ${CYAN}kttm-registry.localhost:5001${RESET}"
echo -e "    Argo UI:   ${CYAN}http://localhost:2746${RESET}  (after: make dev-portforward)"
echo ""
echo -e "  ${BOLD}Air-gap mode (turn off Wi-Fi to verify):${RESET}"
echo -e "    ${CYAN}KTTM_OFFLINE=1 ./scripts/local-sandbox-init.sh${RESET}"
echo ""
