# Repository Instructions

For every code change in this repository, read and follow [specs/Deployment.md](../specs/Deployment.md).

## Default Code Generation Gate

Apply the KTTM QA Architect rules to every generated or modified code change in this repository, whether or not the custom agent is selected:

1. Work test-first: add or update tests for the requested behavior before implementing it.
2. Do not modify or weaken existing functional test assertions to make a change pass.
3. Require complete automated unit and integration coverage of every new or changed feature path. Do not treat the repository-wide coverage threshold as a substitute for feature coverage. If complete coverage cannot be demonstrated, stop and report the uncovered paths and blocker rather than claiming completion.
4. Preserve backward compatibility. If a breaking architectural change is unavoidable, stop and explain the exact conflict and a product-code structural fix before proceeding.
5. Run the relevant Go unit and integration checks, then follow the deployment and smoke-test workflow below for code changes. Preserve unrelated user changes; only undo changes made for the current task when a regression requires it.

Use [.github/agents/kttm-qa-architect.agent.md](agents/kttm-qa-architect.agent.md) for the detailed workflow and reporting checklist.

After implementing a code change:

1. Run the relevant tests and builds.
2. Build ARM64-compatible API and web images with a new version tag.
3. Import changed images into the Rancher Desktop Kubernetes runtime.
4. Run `helm upgrade --install` for the `flowengine` release.
5. Wait for API, web, and NATS workloads to become ready.
6. Port-forward the web service and smoke-test the UI and `/api/schema` endpoint.
7. Inspect pod logs and events and repair deployment failures before reporting completion.

Do not report a deployment as complete based only on a successful image build or Helm command. The cluster rollout and HTTP smoke test must pass. Preserve the chart's probes, resource limits, non-root security contexts, and read-only root filesystem settings.
