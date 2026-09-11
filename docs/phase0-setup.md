# Phase 0 — Development Foundation
# KubeNLCode Platform · Bootstrap Guide

> [!IMPORTANT]
> **MANUAL STEPS REQUIRED** are marked with 🔴 and must be done by you before the automated setup runs.  
> Everything else is automated. After the manual steps, run `make bootstrap` and you're ready.

---

## 🔴 MANUAL STEP 1 — Create the GitHub Repository

**Do this once before anything else.**

1. Go to **https://github.com/new**
2. Set:
   - **Repository name:** `kubenocode`
   - **Description:** `KubeNLCode — Kubernetes-native, air-gapped, polyglot workflow & no-code platform`
   - **Visibility:** `Public` (required for CNCF Sandbox goal)
   - **Initialize with:** ✅ Add README, ✅ Add .gitignore (Go), ✅ Choose Apache 2.0 license
3. Click **Create repository**
4. Copy the SSH clone URL (e.g. `git@github.com:YourUsername/kubenocode.git`)
5. Add it as remote: `git remote add origin git@github.com:YourUsername/kubenocode.git`

---

## 🔴 MANUAL STEP 2 — Set GitHub Repository Settings

In your new GitHub repo → **Settings**:

### Branch Protection (Settings → Branches → Add rule)
- Branch name pattern: `main`
- ✅ Require a pull request before merging
- ✅ Require status checks to pass: `lint`, `test-unit`, `test-integration`, `build`
- ✅ Require branches to be up to date before merging
- ✅ Restrict who can push to matching branches (add yourself)

### Topics (top of repo page → gear icon)
Add topics: `kubernetes`, `workflow`, `no-code`, `etl`, `cncf`, `go`, `argo-workflows`, `nats`

### GitHub Pages (Settings → Pages)
- Source: `Deploy from a branch` → `gh-pages` branch (will be auto-created by CI)

---

## 🔴 MANUAL STEP 3 — Add GitHub Actions Secrets

Go to **Settings → Secrets and variables → Actions → New repository secret**:

| Secret Name | Value | Purpose |
|---|---|---|
| `GHCR_TOKEN` | Your GitHub Personal Access Token (PAT) with `write:packages` | Push Docker images to ghcr.io |
| `GEMINI_API_KEY` | Your Google Gemini API key | AI Advisor in CI integration tests |
| `CODECOV_TOKEN` | Create free account at codecov.io, copy token | Coverage reports in PRs |

**How to create a PAT:**
1. GitHub → Profile → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. New token → Note: `kubenocode-ghcr` → Expiration: 1 year
3. Scopes: ✅ `write:packages`, ✅ `read:packages`, ✅ `delete:packages`
4. Generate and copy → paste as `GHCR_TOKEN` secret

---

## 🔴 MANUAL STEP 4 — Install Prerequisites (Mac)

Open Terminal and run these **one time**:

```bash
# 1. Homebrew (if not already installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 2. Core tools
brew install go node docker kubectl helm k3d golangci-lint lefthook

# 3. Verify
go version          # should show go1.22+
node --version      # should show v20+
docker --version    # Docker Desktop must be running
kubectl version     # client version only (no cluster yet)
k3d version         # should show v5.x
```

> ⚠️ **Docker Desktop must be running** before any `make cluster` commands.

---

## 🔴 MANUAL STEP 5 — Install Prerequisites (Windows / WSL)

Open **WSL (Ubuntu 22.04 or 24.04)** terminal:

```bash
# 1. Install Go 1.22
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 2. Install Node.js 20
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# 3. Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl && sudo mv kubectl /usr/local/bin/

# 4. Install helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# 5. Install k3d
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# 6. Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.59.1

# 7. Install lefthook
go install github.com/evilmartians/lefthook@latest

# 8. Docker Desktop (Windows host) — enable WSL integration
# In Docker Desktop → Settings → Resources → WSL Integration
# Enable for your Ubuntu distro
```

> ⚠️ **WSL Note:** Docker runs via Docker Desktop on Windows. The Docker socket is shared into WSL automatically when "WSL Integration" is enabled in Docker Desktop settings.

---

## Automated Setup (After All Manual Steps Above)

Once prerequisites are installed and GitHub repo is created, everything below is automated via `make`:

```bash
# Clone your repo and enter it
git clone git@github.com:YourUsername/kubenocode.git
cd kubenocode

# Bootstrap everything in one command:
make bootstrap
```

`make bootstrap` will:
1. Install pre-commit hooks (`lefthook install`)
2. Download Go dependencies (`go mod download`)
3. Install frontend dependencies (`npm install`)
4. Create k3d cluster
5. Install Argo Workflows, NATS, KEDA, nginx-ingress
6. Run smoke test to verify everything works
7. Print the access URLs

---

## What You Get After Phase 0

| URL | What's there |
|---|---|
| `http://localhost:5173` | KubeNLCode SPA (dev server) |
| `http://localhost:8080` | KubeNLCode via nginx ingress |
| `http://localhost:8080/argo` | Argo Workflows UI |
| `http://localhost:9090` | Prometheus metrics |
| `http://localhost:4317` | OTel collector (gRPC) |

---

## Troubleshooting

### Mac: `k3d cluster create` fails
```bash
# Ensure Docker Desktop is running
open -a Docker
# Wait 30 seconds then retry
make cluster
```

### WSL: Cannot connect to Docker daemon
```bash
# In Docker Desktop → Settings → General → ensure "Use WSL 2 based engine" is checked
# In Docker Desktop → Settings → Resources → WSL Integration → enable your distro
```

### Port 8080 already in use
```bash
lsof -i :8080   # find the process
kill <PID>      # kill it
make cluster    # retry
```

### golangci-lint: command not found (WSL)
```bash
export PATH=$PATH:$(go env GOPATH)/bin
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
```
