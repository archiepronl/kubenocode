# KTTM QA Validation & Testing Guide (Phase 0)

This document outlines the rigorous, multi-layered validation sequence required to prove that the Dev Foundation (Phase 0) is entirely stable and strictly follows our zero-regression mandate. 

Run these commands in your M2 MacBook terminal to validate the entire Phase 0 local sandbox architecture:

## 1. Validate Infrastructure Bootstrapping

First, tear down any stale states and bring up the pristine local K3d mesh. This proves the setup script correctly provisions the cluster and all necessary CNCF components (KEDA, Argo, NATS, NGINX).

```bash
# 1. Provision the cluster and dependencies
make setup

# 2. Verify the K3d nodes are running locally (expect 1 server, 2 agents)
kubectl get nodes

# 3. Verify KEDA (Auto-scaler) is active
kubectl get pods -n keda

# 4. Verify Argo Workflows (Execution Engine) is active
kubectl get pods -n argo
```

## 2. Validate Code Quality & CI Regression Gate

As defined in our `.antigravityrc` policies, we must ensure zero regressions and passing test execution before any code is pushed. Run the universal task master to prove formatting, linting, builds, and unit tests all pass.

```bash
# Execute the full local CI gate (Format -> Build -> Test -> Lint)
make ci-local
```

*(Expected Result: You should see `==> CI Local Gate PASSED — safe to push to Git` at the end of the output.)*

## 3. Validate Live-Sync Compilation (Skaffold)

Finally, verify that the reactive developer loop successfully packages the code, builds the ARM64 Alpine Docker container locally, and injects it into the cluster without crashing.

```bash
# Start the reactive sync engine
make dev
```

*(Expected Result: Skaffold will build the `kttm-operator` image, apply the CRDs and Deployment from `deployments/`, and stream the pod logs directly to your terminal. Once it says the deployment stabilized, press `Ctrl+C` to terminate the sync.)*
