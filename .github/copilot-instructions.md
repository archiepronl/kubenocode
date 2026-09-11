# Repository Instructions

For every code change in this repository, read and follow [specs/Deployment.md](../specs/Deployment.md).

After implementing a code change:

1. Run the relevant tests and builds.
2. Build ARM64-compatible API and web images with a new version tag.
3. Import changed images into the Rancher Desktop Kubernetes runtime.
4. Run `helm upgrade --install` for the `flowengine` release.
5. Wait for API, web, and NATS workloads to become ready.
6. Port-forward the web service and smoke-test the UI and `/api/schema` endpoint.
7. Inspect pod logs and events and repair deployment failures before reporting completion.

Do not report a deployment as complete based only on a successful image build or Helm command. The cluster rollout and HTTP smoke test must pass. Preserve the chart's probes, resource limits, non-root security contexts, and read-only root filesystem settings.
