module github.com/kubeworkflow/flowengine

go 1.22.0

require (

	// NATS.io JetStream — FR-1.3 Low-Latency Messaging
	github.com/nats-io/nats.go v1.35.0

	// CLI
	github.com/spf13/cobra v1.8.1
	// Kubernetes client and controller-runtime
	k8s.io/api v0.30.1
	k8s.io/apimachinery v0.30.1
	k8s.io/client-go v0.30.1
	sigs.k8s.io/controller-runtime v0.18.4
)
