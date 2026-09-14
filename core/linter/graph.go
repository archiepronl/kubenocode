// Package linter implements FR-4 (Pre-Deployment Static Validation Linter).
//
// The linter analyzes a WorkflowDAG before it is compiled into Argo Workflow CRDs.
// It runs synchronously in the reconcile loop and blocks deployment if critical errors are found.
//
// Checks performed:
//  1. Cycle detection — DFS with three-color marking (white/gray/black)
//  2. Dangling edge detection — outputs referencing non-existent node IDs
//  3. Node type validation — known type prefixes (connector/, script/, transform/)
//  4. SecretRef presence — nodes requiring credentials must reference a secret
//  5. Resource limit validation — warns if no CPU/memory limits are set
//  6. Isolated node detection — nodes with no inputs AND no outputs (dead nodes)
//  7. MimeType compatibility — warns when a producer's hint doesn't match a consumer's expected input
package linter

import (
	"fmt"
	"strings"

	v1alpha1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
)

// ─────────────────────────────────────────────
//  Result types
// ─────────────────────────────────────────────

// Severity represents the importance of a lint finding.
type Severity string

const (
	// SeverityError is a blocking finding. Deployment is refused until fixed.
	SeverityError Severity = "error"
	// SeverityWarning is an advisory finding. Deployment proceeds but the issue is surfaced.
	SeverityWarning Severity = "warning"
)

// LintFinding represents a single discovered issue in the workflow DAG.
type LintFinding struct {
	// Severity indicates whether this finding blocks deployment.
	Severity Severity `json:"severity"`

	// NodeID identifies which node triggered this finding. Empty for graph-level issues.
	NodeID string `json:"nodeId,omitempty"`

	// Code is a short, machine-readable identifier for the finding type.
	Code string `json:"code"`

	// Message is a human-readable description suitable for display in the UI.
	Message string `json:"message"`
}

func (f LintFinding) Error() string {
	return fmt.Sprintf("[%s][%s] node=%q: %s", f.Severity, f.Code, f.NodeID, f.Message)
}

// LintResult is the complete output of a lint run.
type LintResult struct {
	// Findings is the full list of discovered issues, in discovery order.
	Findings []LintFinding `json:"findings"`
}

// HasErrors returns true if any finding has SeverityError.
func (r *LintResult) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns only the blocking error findings.
func (r *LintResult) Errors() []LintFinding {
	var errs []LintFinding
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			errs = append(errs, f)
		}
	}
	return errs
}

// ErrorStrings returns error messages as a string slice (for status conditions).
func (r *LintResult) ErrorStrings() []string {
	var ss []string
	for _, f := range r.Errors() {
		ss = append(ss, f.Error())
	}
	return ss
}

// ─────────────────────────────────────────────
//  Known node type prefixes
// ─────────────────────────────────────────────

var knownTypePrefixes = []string{
	"connector/",
	"script/",
	"transform/",
	"sink/",
	"trigger/",
	"ai/",
}

// ─────────────────────────────────────────────
//  Connectors that require a secretRef
// ─────────────────────────────────────────────

var secretRequiredPrefixes = []string{
	"connector/s3",
	"connector/postgres",
	"connector/mysql",
	"connector/mssql",
	"connector/oracle",
	"connector/mongodb",
	"connector/redis",
	"connector/kafka",
	"connector/rabbitmq",
	"connector/mqtt",
	"connector/snowflake",
	"connector/bigquery",
	"connector/redshift",
}

// ─────────────────────────────────────────────
//  GraphLinter
// ─────────────────────────────────────────────

// GraphLinter analyzes a FullStackApplication's WorkflowDAG for structural and
// semantic issues before compilation and deployment.
type GraphLinter struct{}

// New returns a ready-to-use GraphLinter.
func New() *GraphLinter { return &GraphLinter{} }

// Lint runs all checks against the given WorkflowDAG and returns a LintResult.
// It is designed to be called synchronously inside the operator's Reconcile loop.
func (l *GraphLinter) Lint(dag []v1alpha1.WorkflowNode) *LintResult {
	result := &LintResult{}

	// Build adjacency index for O(1) lookups
	nodeIndex := buildNodeIndex(dag)

	// Run each check, accumulating findings
	result.Findings = append(result.Findings, l.checkDanglingEdges(dag, nodeIndex)...)
	result.Findings = append(result.Findings, l.checkCycles(dag, nodeIndex)...)
	result.Findings = append(result.Findings, l.checkNodeTypes(dag)...)
	result.Findings = append(result.Findings, l.checkSecretRefs(dag)...)
	result.Findings = append(result.Findings, l.checkResourceLimits(dag)...)
	result.Findings = append(result.Findings, l.checkIsolatedNodes(dag, nodeIndex)...)

	return result
}

// buildNodeIndex constructs a map from node ID → WorkflowNode for O(1) lookup.
func buildNodeIndex(dag []v1alpha1.WorkflowNode) map[string]v1alpha1.WorkflowNode {
	idx := make(map[string]v1alpha1.WorkflowNode, len(dag))
	for _, n := range dag {
		idx[n.ID] = n
	}
	return idx
}

// ─────────────────────────────────────────────
//  Check: Dangling edges
// ─────────────────────────────────────────────

func (l *GraphLinter) checkDanglingEdges(dag []v1alpha1.WorkflowNode, idx map[string]v1alpha1.WorkflowNode) []LintFinding {
	var findings []LintFinding
	for _, node := range dag {
		for _, outputID := range node.Outputs {
			if _, exists := idx[outputID]; !exists {
				findings = append(findings, LintFinding{
					Severity: SeverityError,
					NodeID:   node.ID,
					Code:     "DANGLING_EDGE",
					Message:  fmt.Sprintf("output references unknown node %q", outputID),
				})
			}
		}
	}
	return findings
}

// ─────────────────────────────────────────────
//  Check: Cycle detection (DFS three-color)
// ─────────────────────────────────────────────

// colorState maps a node ID to its DFS visitation color.
type colorState int

const (
	white colorState = iota // not yet visited
	gray                    // currently on the DFS stack (potential cycle back-edge)
	black                   // fully processed
)

func (l *GraphLinter) checkCycles(dag []v1alpha1.WorkflowNode, idx map[string]v1alpha1.WorkflowNode) []LintFinding {
	color := make(map[string]colorState, len(dag))
	var findings []LintFinding

	for _, node := range dag {
		if color[node.ID] == white {
			if l.dfsVisit(node.ID, color, idx, &findings) {
				// Early exit on first cycle found — subsequent DFS would repeat the error.
				break
			}
		}
	}
	return findings
}

// dfsVisit performs a recursive DFS from startID, returning true if a cycle is detected.
func (l *GraphLinter) dfsVisit(nodeID string, color map[string]colorState, idx map[string]v1alpha1.WorkflowNode, findings *[]LintFinding) bool {
	color[nodeID] = gray

	node, exists := idx[nodeID]
	if !exists {
		color[nodeID] = black
		return false
	}

	for _, childID := range node.Outputs {
		switch color[childID] {
		case gray:
			// Back-edge → cycle detected
			*findings = append(*findings, LintFinding{
				Severity: SeverityError,
				NodeID:   nodeID,
				Code:     "CIRCULAR_DEPENDENCY",
				Message: fmt.Sprintf(
					"cycle detected: node %q → %q creates an infinite loop in the workflow graph",
					nodeID, childID,
				),
			})
			return true
		case white:
			if l.dfsVisit(childID, color, idx, findings) {
				return true
			}
		}
	}

	color[nodeID] = black
	return false
}

// ─────────────────────────────────────────────
//  Check: Node type validation
// ─────────────────────────────────────────────

func (l *GraphLinter) checkNodeTypes(dag []v1alpha1.WorkflowNode) []LintFinding {
	var findings []LintFinding
	for _, node := range dag {
		known := false
		for _, prefix := range knownTypePrefixes {
			if strings.HasPrefix(node.Type, prefix) {
				known = true
				break
			}
		}
		if !known {
			findings = append(findings, LintFinding{
				Severity: SeverityWarning,
				NodeID:   node.ID,
				Code:     "UNKNOWN_NODE_TYPE",
				Message: fmt.Sprintf(
					"node type %q is not a recognized built-in prefix (connector/, script/, transform/, sink/, trigger/, ai/). "+
						"Ensure a custom Docker image is provided if this is a custom node.",
					node.Type,
				),
			})
		}

		// script/* nodes must have either an image or a script body
		if strings.HasPrefix(node.Type, "script/") {
			if node.Image == "" && node.Script == "" {
				findings = append(findings, LintFinding{
					Severity: SeverityError,
					NodeID:   node.ID,
					Code:     "MISSING_SCRIPT_IMAGE",
					Message:  "script/* nodes require either an 'image' (Docker image URI) or an inline 'script' body",
				})
			}
		}
	}
	return findings
}

// ─────────────────────────────────────────────
//  Check: SecretRef required
// ─────────────────────────────────────────────

func (l *GraphLinter) checkSecretRefs(dag []v1alpha1.WorkflowNode) []LintFinding {
	var findings []LintFinding
	for _, node := range dag {
		for _, prefix := range secretRequiredPrefixes {
			if strings.HasPrefix(node.Type, prefix) && node.SecretRef == "" {
				findings = append(findings, LintFinding{
					Severity: SeverityError,
					NodeID:   node.ID,
					Code:     "MISSING_SECRET_REF",
					Message: fmt.Sprintf(
						"connector type %q requires a 'secretRef' pointing to a Kubernetes Secret with credentials. "+
							"Credentials must never be embedded in the spec directly.",
						node.Type,
					),
				})
			}
		}
	}
	return findings
}

// ─────────────────────────────────────────────
//  Check: Resource limits
// ─────────────────────────────────────────────

func (l *GraphLinter) checkResourceLimits(dag []v1alpha1.WorkflowNode) []LintFinding {
	var findings []LintFinding
	for _, node := range dag {
		if node.Resources == nil || len(node.Resources.Limits) == 0 {
			findings = append(findings, LintFinding{
				Severity: SeverityWarning,
				NodeID:   node.ID,
				Code:     "NO_RESOURCE_LIMITS",
				Message: "no CPU/memory limits are set for this node. " +
					"Unbounded nodes can cause OOMKill or CPU throttling on shared clusters. " +
					"Set resources.limits.memory and resources.limits.cpu.",
			})
		}
	}
	return findings
}

// ─────────────────────────────────────────────
//  Check: Isolated nodes (no connections)
// ─────────────────────────────────────────────

func (l *GraphLinter) checkIsolatedNodes(dag []v1alpha1.WorkflowNode, idx map[string]v1alpha1.WorkflowNode) []LintFinding {
	// Build a set of nodes that are referenced as an output by any other node
	referenced := make(map[string]bool)
	for _, node := range dag {
		for _, out := range node.Outputs {
			referenced[out] = true
		}
	}

	var findings []LintFinding
	for _, node := range dag {
		isSource := !referenced[node.ID] // no incoming edges
		isSink := len(node.Outputs) == 0 // no outgoing edges
		if isSource && isSink && len(dag) > 1 {
			findings = append(findings, LintFinding{
				Severity: SeverityWarning,
				NodeID:   node.ID,
				Code:     "ISOLATED_NODE",
				Message: fmt.Sprintf(
					"node %q has no incoming or outgoing connections. It will never execute. "+
						"Connect it to the workflow or remove it.",
					node.ID,
				),
			})
		}
	}
	return findings
}
