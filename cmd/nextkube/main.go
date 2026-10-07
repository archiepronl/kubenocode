// Package main implements the nextkube CLI tool (FR-5.4).
//
// The CLI enables disconnected workstation execution — running workflows locally
// using Docker or Podman without requiring a running Kubernetes cluster.
// It also provides bundle export/import for air-gapped deployments (FR-5.1, FR-5.2).
//
// Commands:
//
//	nextkube run <workflow.yaml>           - Execute a workflow locally via Docker/Podman
//	nextkube export <name> -o bundle.tar.gz - Bundle workflow + images for air-gap export
//	nextkube import bundle.tar.gz          - Load bundle into current cluster
//	nextkube validate <workflow.yaml>      - Run the pre-deploy linter without executing
//	nextkube gitops sync                   - Commit current state to GitOps repo
//	nextkube version                       - Print version info
package main

import (
	"fmt"
	"os"

	"github.com/kubeworkflow/flowengine/core/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
