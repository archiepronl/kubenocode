# Phase 1 & 2 — ETL MVP (Core Operator & Scripting Engine)
# KubeNoCode Platform · Implementation Guide

To move beyond the baseline infrastructure into real execution, we must build the core control plane. This phase orchestrates the deployment of a real data pipeline from S3, pushing bytes through an transformation engine, and sinking them into Postgres — all running on the K3d cluster.

We will accomplish this by building three primary architecture pillars:

1. **The Custom Controller & Linter**: A Kubernetes Operator that watches for `KttmApp` CRDs, statically validates the Directed Acyclic Graph (DAG) for cycles, and reports status via K8s events.
2. **The Workflow Compiler**: Translates our abstract `KttmApp` DAG nodes into native Argo Workflows, mapping our inputs/outputs to Argo artifacts and NATS streams.
3. **The Scripting Execution Engine**: Spawns secure, isolated containers (Python, Bash, JS) inside the cluster that process streaming data securely from our storage systems.

---

## Step 1: Generate the Core API Definitions (CRDs)

We must fully define the `KttmApp` structure which will serve as the single source of truth for the No-Code platform.

### File 1: API Types (`api/v1alpha1/kttmapp_types.go`)
This defines the Go structs for the DAG schema.

```go
// api/v1alpha1/kttmapp_types.go
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkflowNode struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // e.g. "trigger/webhook", "script/python", "sink/postgres"
	Name        string            `json:"name,omitempty"`
	Dependencies []string         `json:"dependencies,omitempty"` // IDs of parent nodes
	Params      map[string]string `json:"params,omitempty"`
	Script      string            `json:"script,omitempty"` // For script nodes
	SecretRef   string            `json:"secretRef,omitempty"`
}

type KttmAppSpec struct {
	AppID       string         `json:"appId"`
	Version     string         `json:"version"`
	Nodes       []WorkflowNode `json:"nodes"`
}

type KttmAppStatus struct {
	Phase            string `json:"phase,omitempty"` // Pending, Linting, Running, Failed, Succeeded
	ArgoWorkflowName string `json:"argoWorkflowName,omitempty"`
	Conditions       []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

type KttmApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KttmAppSpec   `json:"spec,omitempty"`
	Status KttmAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KttmAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KttmApp `json:"items"`
}
```

---

## Step 2: The Core Graph Linter

Before the operator attempts to compile a workflow to Argo, it must mathematically ensure the graph is executable (no cyclic dependencies, no orphan nodes).

### File 2: Graph Linter (`core/linter/graph.go`)

```go
// core/linter/graph.go
package linter

import (
	"errors"
	"github.com/kubeworkflow/kttm/api/v1alpha1"
)

// ValidateDAG runs a DFS to detect cycles in the provided nodes
func ValidateDAG(nodes []v1alpha1.WorkflowNode) error {
	if len(nodes) == 0 {
		return errors.New("workflow contains no nodes")
	}

	adjList := make(map[string][]string)
	for _, n := range nodes {
		// Child nodes depend on parent nodes
		for _, dep := range n.Dependencies {
			adjList[dep] = append(adjList[dep], n.ID)
		}
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range adjList[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true // Cycle detected
			}
		}
		recStack[node] = false
		return false
	}

	for _, n := range nodes {
		if !visited[n.ID] {
			if dfs(n.ID) {
				return errors.New("invalid DAG: circular dependency detected")
			}
		}
	}

	return nil
}
```

---

## Step 3: Argo Workflow Compiler

The core logic that bridges KTTM to Argo. It translates `KttmApp` DAGs into native `Workflow` specs.

### File 3: The Argo Compiler (`core/engine/compiler.go`)

```go
// core/engine/compiler.go
package engine

import (
	"github.com/kubeworkflow/kttm/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// CompileToArgo takes a KttmApp DAG and returns a raw Argo Workflow unstructured object
func CompileToArgo(app *v1alpha1.KttmApp) (*unstructured.Unstructured, error) {
	tasks := []interface{}{}

	// Map KTTM nodes to Argo DAG Tasks
	for _, node := range app.Spec.Nodes {
		task := map[string]interface{}{
			"name":         node.ID,
			"template":     "kttm-node-executor",
			"dependencies": node.Dependencies,
			"arguments": map[string]interface{}{
				"parameters": []map[string]interface{}{
					{"name": "nodeType", "value": node.Type},
					{"name": "scriptContent", "value": node.Script},
				},
			},
		}
		tasks = append(tasks, task)
	}

	// Build the Argo Workflow Custom Resource
	workflow := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Workflow",
			"metadata": map[string]interface{}{
				"generateName": app.Name + "-",
				"namespace":    app.Namespace,
			},
			"spec": map[string]interface{}{
				"entrypoint": "main-dag",
				"templates": []interface{}{
					map[string]interface{}{
						"name": "main-dag",
						"dag": map[string]interface{}{
							"tasks": tasks,
						},
					},
					// Executor template representing a secure runner for Python/Bash/JS
					map[string]interface{}{
						"name": "kttm-node-executor",
						"inputs": map[string]interface{}{
							"parameters": []map[string]interface{}{
								{"name": "nodeType"},
								{"name": "scriptContent"},
							},
						},
						"container": map[string]interface{}{
							"image":   "kttm-registry.localhost:5001/kttm-server", // Unified runner for MVP
							"command": []string{"/app/runner"},
							"args":    []string{"{{inputs.parameters.nodeType}}", "{{inputs.parameters.scriptContent}}"},
						},
					},
				},
			},
		},
	}

	return workflow, nil
}
```

---

## Step 4: The Operator Reconcile Loop

The control plane that wires it all together.

### File 4: Reconciler (`controllers/kttmapp_controller.go`)

```go
// controllers/kttmapp_controller.go
package controllers

import (
	"context"
	"github.com/kubeworkflow/kttm/api/v1alpha1"
	"github.com/kubeworkflow/kttm/core/engine"
	"github.com/kubeworkflow/kttm/core/linter"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KttmAppReconciler struct {
	client.Client
}

func (r *KttmAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var app v1alpha1.KttmApp
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 1. Lint the DAG
	if err := linter.ValidateDAG(app.Spec.Nodes); err != nil {
		app.Status.Phase = "LintFailed"
		_ = r.Status().Update(ctx, &app)
		return ctrl.Result{}, err
	}

	// 2. Compile to Argo
	workflow, err := engine.CompileToArgo(&app)
	if err != nil {
		app.Status.Phase = "CompilationFailed"
		_ = r.Status().Update(ctx, &app)
		return ctrl.Result{}, err
	}

	// 3. Set Owner Reference (Cascade Delete)
	ctrl.SetControllerReference(&app, workflow, r.Scheme())

	// 4. Apply to Cluster
	if err := r.Create(ctx, workflow); err != nil {
		if !errors.IsAlreadyExists(err) {
			return ctrl.Result{}, err
		}
	}

	// 5. Update Status
	app.Status.Phase = "Deployed"
	app.Status.ArgoWorkflowName = workflow.GetName()
	if err := r.Status().Update(ctx, &app); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
```

---

## Validation & Testing

To test this local MVP:
1. Ensure your `make setup` local cluster is running.
2. Apply the new CRD structs via `make manifests && kubectl apply -f deploy/crds/kttmapp_crd.yaml`.
3. Submit a test `KttmApp` DAG containing a cyclic dependency and verify the linter correctly halts execution and updates the CRD status to `LintFailed`.
4. Fix the cycle, re-apply, and observe the custom controller successfully deploying the compiled Argo Workflow into the `argo` namespace.
