#!/usr/bin/env bash
# scripts/setup-cluster.sh
set -euo pipefail

CLUSTER_NAME="kttm-engine"

echo "=== Checking Core Prerequisites ==="
if ! command -v k3d &> /dev/null; then
    echo "ERROR: k3d is not installed. Run 'brew install k3d'."
    exit 1
fi
if ! command -v helm &> /dev/null; then
    echo "ERROR: helm is not installed. Run 'brew install helm'."
    exit 1
fi

echo "=== Purging Existing Cluster Contexts If Present ==="
k3d cluster delete "$CLUSTER_NAME" || true

echo "=== Bootstrapping KTTM Multi-Node K3s Mesh ==="
# Configuring 1 master server and 2 parallel execution agents
k3d cluster create "$CLUSTER_NAME" \
  --servers 1 \
  --agents 2 \
  --port "8080:80@loadbalancer" \
  --k3s-arg "--disable=traefik@server:0" \
  --k3s-arg "--disable=servicelb@server:0" \
  --k3s-arg "--kube-apiserver-arg=feature-gates=InPlacePodVerticalScaling=true@server:0" \
  --timeout 5m

echo "=== Verifying Cluster Communication ==="
kubectl cluster-info

echo "=== Creating Core Namespaces ==="
kubectl create namespace kttm-system --dry-run=client -o yaml | kubectl apply -f -

echo "=== Installing KEDA Event-Driven Horizontal Scaler ==="
kubectl create namespace keda --dry-run=client -o yaml | kubectl apply -f -
kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.15.1/keda-2.15.1.yaml

echo "=== Installing Argo Workflows (Core Execution Engine) ==="
kubectl create namespace argo --dry-run=client -o yaml | kubectl apply -f -
kubectl apply --server-side -n argo -f https://github.com/argoproj/argo-workflows/releases/latest/download/install.yaml

echo "--> Configuring Argo for Local Testing (Server Auth Mode bypass)..."
kubectl patch deployment argo-server -n argo --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": ["server", "--auth-mode=server"]}]'

echo "=== Installing NATS JetStream (Event Bus) ==="
helm repo add nats https://nats-io.github.io/k8s/helm/charts/ 2>/dev/null || true
helm repo update nats
helm upgrade --install kttm-nats nats/nats --namespace default \
  --set config.jetstream.enabled=true \
  --set config.jetstream.memStorage.enabled=true \
  --set config.jetstream.memStorage.size=256Mi \
  --wait --timeout=120s

echo "=== Installing Kubernetes Dashboard (K3d UI) ==="
kubectl apply --server-side -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.7.0/aio/deploy/recommended.yaml

echo "=== Installing NGINX Ingress Controller ==="
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx 2>/dev/null || true
helm repo update ingress-nginx
helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx --create-namespace \
  --set controller.service.type=LoadBalancer \
  --wait --timeout=120s

echo "=== Verifying Cluster Core Readiness ==="
kubectl wait --namespace keda --for=condition=ready pod --selector=app=keda-operator --timeout=90s
kubectl wait --namespace argo --for=condition=ready pod --selector=app=workflow-controller --timeout=90s

echo "================================================================="
echo " SUCCESS: KTTM Development Cluster Ready on Apple Silicon M2!"
echo " Included: KEDA, Argo Workflows, NATS JetStream, NGINX Ingress."
echo " In-Place Pod Resource Resizing (Vertical) Activated."
echo " Run 'make dev' to start reactive live-code compilation."
echo "================================================================="
