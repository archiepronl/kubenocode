# FlowEngine

FlowEngine is a deployable MVP for schema-driven workflow intake. The current slice includes:

- A dependency-light Go API with workflow graph validation and JSON schema delivery.
- A React/Vite renderer that turns the schema into a reactive business form.
- Multi-stage container images for the API and web renderer.
- A Helm chart with probes, resource limits, non-root security contexts, optional ingress, and NATS JetStream.

## Local development

The frontend can be built without Go or Kubernetes:

```sh
npm install --prefix frontend/web-renderer
npm run build --prefix frontend/web-renderer
```

The API requires Go 1.22 or newer:

```sh
go test ./...
go run ./cmd/server
```

## Build images

Start Docker Desktop, then run:

```sh
docker build -t flowengine-api:0.1.0 .
docker build -t flowengine-web:0.1.0 frontend/web-renderer
```

The web image proxies `/api/*` to the Helm-generated Kubernetes service named `flowengine-flowengine-api` for the default `flowengine` release.

The complete repeatable build, import, deploy, and smoke-test workflow is documented in [specs/Deployment.md](specs/Deployment.md).

## Kubernetes deployment

Push both images to a registry and override the example repositories during install:

```sh
helm dependency update deploy/helm/flowengine
helm upgrade --install flowengine deploy/helm/flowengine \
  --set api.image.repository=registry.example.com/flowengine-api \
  --set web.image.repository=registry.example.com/flowengine-web \
  --set ingress.enabled=true \
  --set ingress.host=flowengine.example.com
```

NATS is enabled by default for the event-driven runtime contract. The current MVP accepts workflow submissions over HTTP; NATS-backed execution workers are the next runtime increment.