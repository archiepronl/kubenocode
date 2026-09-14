// Package main implements the flowengine CLI tool (FR-5.4).
//
// The CLI enables disconnected workstation execution — running workflows locally
// using Docker or Podman without requiring a running Kubernetes cluster.
// It also provides bundle export/import for air-gapped deployments (FR-5.1, FR-5.2).
//
// Commands:
//
//	flowengine run <workflow.yaml>           - Execute a workflow locally via Docker/Podman
//	flowengine export <name> -o bundle.tar.gz - Bundle workflow + images for air-gap export
//	flowengine import bundle.tar.gz          - Load bundle into current cluster
//	flowengine validate <workflow.yaml>      - Run the pre-deploy linter without executing
//	flowengine gitops sync                   - Commit current state to GitOps repo
//	flowengine version                       - Print version info
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
