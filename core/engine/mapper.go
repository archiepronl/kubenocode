package engine

import (
	"time"

	v1alpha1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
)

// MapCRDToExecutionGraph translates the Kubernetes CRD definition of a workflow
// into the internal Engine domain model for compilation and execution.
func MapCRDToExecutionGraph(app *v1alpha1.FullStackApplication) *ExecutionGraph {
	if app == nil || len(app.Spec.WorkflowDAG) == 0 {
		return nil
	}

	graph := &ExecutionGraph{
		ID:        app.Name,
		Nodes:     make(map[string]*ExecutionNode),
		CreatedAt: time.Now(),
	}

	// First pass: create all nodes
	for _, crdNode := range app.Spec.WorkflowDAG {
		node := &ExecutionNode{
			ID:            crdNode.ID,
			Type:          crdNode.Type,
			Label:         crdNode.Label,
			GroupID:       crdNode.GroupID,
			PackagingMode: crdNode.PackagingMode,
			Language:      crdNode.Language,
			Params:        make(map[string]string),
			SecretRef:     crdNode.SecretRef,
			Image:         crdNode.Image,
			Script:        crdNode.Script,
			Outputs:       append([]string{}, crdNode.Outputs...),
			MimeTypeHint:  crdNode.MimeTypeHint,
		}

		for k, v := range crdNode.Params {
			node.Params[k] = v
		}

		graph.Nodes[node.ID] = node

		// Determine EntryNode (heuristic: first node without any incoming edges)
		// This is a naive implementation; GraphLinter handles proper DAG validation.
		if graph.EntryNode == "" {
			graph.EntryNode = node.ID
		}
	}

	return graph
}
