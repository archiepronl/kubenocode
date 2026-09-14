#!/usr/bin/env bash
set -e

# Colors
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
RESET='\033[0m'

echo -e "${CYAN}==> Starting KTTM Visual Interfaces${RESET}"
echo -e "${YELLOW}Keep this terminal tab open. Press Ctrl+C to stop all UIs.${RESET}\n"

# 1. K3d UI / Kubernetes Dashboard
echo -e "--> 1. Kubernetes Dashboard (K3d UI)"
echo -e "       Available at: ${GREEN}https://localhost:9090${RESET}"
kubectl -n kubernetes-dashboard port-forward svc/kubernetes-dashboard 9090:443 > /dev/null 2>&1 &
K8S_PID=$!

# 2. Argo Workflows UI
echo -e "--> 2. Argo Workflows UI (Execution Engine)"
echo -e "       Available at: ${GREEN}https://localhost:2746${RESET}"
kubectl -n argo port-forward service/argo-server 2746:2746 > /dev/null 2>&1 &
ARGO_PID=$!

# 3. KubeNoCode App (Frontend)
echo -e "--> 3. KubeNoCode App (BFF/Frontend)"
echo -e "       Available at: ${GREEN}http://localhost:8080${RESET}"
echo -e "       (Routed natively via NGINX Ingress controller - no port-forward needed)"

# 4. Generate Auth Token
echo -e "\n${CYAN}==> Generating Unified Admin Token...${RESET}"
kubectl apply -f deploy/rbac/admin-user.yaml > /dev/null 2>&1 || true
TOKEN=$(kubectl create token kttm-admin -n kttm-system --duration=24h 2>/dev/null || echo "Token not available yet. Deploy first.")
echo -e "${YELLOW}Use this token to log into the Kubernetes Dashboard and Argo UI:${RESET}"
echo -e "${GREEN}$TOKEN${RESET}"

# Trap Ctrl+C to kill the background port-forwards cleanly
trap "echo -e '\n\n${CYAN}==> Shutting down UI connections...${RESET}'; kill $K8S_PID $ARGO_PID; exit 0" SIGINT SIGTERM

echo -e "\n${CYAN}All connections established. UIs are live!${RESET}"
wait
