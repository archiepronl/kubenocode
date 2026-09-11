# FlowEngine Deployment Runbook

This is the required deployment workflow for testing every code change in the local Kubernetes cluster.

## Prerequisites

- Rancher Desktop running with Kubernetes enabled.
- `kubectl`, `helm`, and Docker available on `PATH`.
- Kubernetes context set to the intended test cluster, normally `rancher-desktop`.
- Docker may require `DOCKER_API_VERSION=1.41` when using the current Rancher Desktop daemon.

Check the cluster before changing anything:

```sh
kubectl config current-context
kubectl get nodes
```

The node must report `Ready`.

## Build And Test

Run the application checks before deployment:

```sh
go test ./...
npm run build --prefix frontend/web-renderer
helm dependency update deploy/helm/flowengine
helm lint deploy/helm/flowengine
```

Build images for the ARM64 local cluster:

```sh
DOCKER_API_VERSION=1.41 docker --context desktop-linux build \
  --platform linux/arm64 -t flowengine-api:<version> .
DOCKER_API_VERSION=1.41 docker --context desktop-linux build \
  --platform linux/arm64 -t flowengine-web:<version> frontend/web-renderer
```

Use a new version tag for a changed image. Do not reuse a stale tag during local testing.

## Import Images Into Rancher Desktop

The local build context and Rancher Desktop Kubernetes runtime may use different image stores. Transfer changed images explicitly:

```sh
DOCKER_API_VERSION=1.41 docker --context desktop-linux save \
  flowengine-api:<version> flowengine-web:<version> |
  DOCKER_API_VERSION=1.41 docker --context rancher-desktop load
```

Confirm the images are present:

```sh
DOCKER_API_VERSION=1.41 docker --context rancher-desktop images \
  | grep -E '^flowengine-(api|web):<version> '
```

## Deploy

Install or upgrade the release with local image names:

```sh
helm upgrade --install flowengine deploy/helm/flowengine \
  --set api.image.repository=flowengine-api \
  --set api.image.tag=<version> \
  --set web.image.repository=flowengine-web \
  --set web.image.tag=<version> \
  --set nats.enabled=true
```

The release is installed in the `default` namespace unless a namespace is supplied explicitly.

## Verify

Wait for every application workload:

```sh
kubectl rollout status deployment/flowengine-flowengine-api --timeout=120s
kubectl rollout status deployment/flowengine-flowengine-web --timeout=120s
kubectl rollout status statefulset/flowengine-nats --timeout=120s
kubectl get pods
```

All FlowEngine pods must be `Running` and ready. If a rollout fails, inspect the owning pod before changing manifests:

```sh
kubectl describe pod <pod-name>
kubectl logs <pod-name> --all-containers
kubectl get events --sort-by=.lastTimestamp | tail -40
```

Expose the UI for a local smoke test:

```sh
kubectl port-forward svc/flowengine-flowengine-web 18080:80
```

In another terminal, verify the UI and schema endpoint:

```sh
curl -fsS -o /tmp/flowengine-ui.html \
  -w 'UI %{http_code}\n' http://127.0.0.1:18080/
curl -fsS http://127.0.0.1:18080/api/schema
```

The UI and schema endpoint must return HTTP 200.

## Remote Clusters

For a remote cluster, push both images to an accessible registry and replace the local image repository values with the registry paths. Never put credentials, tokens, or private registry secrets in this file or in workflow bundles.

## Important Runtime Details

- The web service is named `flowengine-flowengine-web` by the default Helm release.
- The API service is named `flowengine-flowengine-api` by the default Helm release.
- The web container proxies `/api/*` to the API service.
- NATS JetStream is enabled by default.
- Keep Kubernetes probes, resource limits, non-root security settings, and read-only root filesystems intact unless a change explicitly requires otherwise.
