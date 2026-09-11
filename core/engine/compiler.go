// Package engine implements the DAG compiler — the core translation layer that converts
// a FullStackApplication's WorkflowDAG into a native Argo Workflows WorkflowTemplate CRD.
//
// This implements FR-1.1 (Cloud-Native Execution): the application engine compiles graph
// configurations directly into Argo Workflows CRDs and delegates scheduling, auto-scaling,
// and lifecycle management to Kubernetes native loops.
//
// The Compiler is designed as a Strategy pattern: it implements the Backend interface
// so Tekton can be plugged in as an alternative without changing the reconcile loop.
package engine

import (
	"encoding/json"
	"fmt"

	v1alpha1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ─────────────────────────────────────────────
//  ArgoWorkflow — minimal CRD struct
// ─────────────────────────────────────────────
// We define a minimal representation of the Argo Workflow CRD here to avoid
// importing the full argo-workflows dependency tree during scaffolding.
// In production, replace with: github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1

// ArgoWorkflow is a minimal representation of an Argo Workflows Workflow CRD.
type ArgoWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ArgoWorkflowSpec `json:"spec"`
}

// ArgoWorkflowSpec defines the structure of an Argo workflow.
type ArgoWorkflowSpec struct {
	// Entrypoint is the name of the initial template to execute.
	Entrypoint string `json:"entrypoint"`

	// ServiceAccountName is forwarded to all step pods.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// TTLStrategy controls automatic cleanup of finished workflows.
	TTLStrategy *ArgoTTLStrategy `json:"ttlStrategy,omitempty"`

	// Templates contains all step and DAG template definitions.
	Templates []ArgoTemplate `json:"templates"`

	// Volumes defines shared volumes accessible to all step containers.
	Volumes []corev1.Volume `json:"volumes,omitempty"`
}

// ArgoTTLStrategy configures when finished workflow resources are garbage collected.
type ArgoTTLStrategy struct {
	SecondsAfterSuccess    *int32 `json:"secondsAfterSuccess,omitempty"`
	SecondsAfterFailure    *int32 `json:"secondsAfterFailure,omitempty"`
	SecondsAfterCompletion *int32 `json:"secondsAfterCompletion,omitempty"`
}

// ArgoTemplate can be either a DAG template (orchestration) or a container template (execution).
type ArgoTemplate struct {
	Name string `json:"name"`

	// DAG is set for orchestration templates that define task graphs.
	DAG *ArgoDAGTemplate `json:"dag,omitempty"`

	// Container is set for leaf execution templates.
	Container *corev1.Container `json:"container,omitempty"`

	// Volumes available to this template's container.
	Volumes []corev1.Volume `json:"volumes,omitempty"`
}

// ArgoDAGTemplate is the DAG task graph definition inside a template.
type ArgoDAGTemplate struct {
	Tasks []ArgoDAGTask `json:"tasks"`
}

// ArgoDAGTask represents a single task in the DAG template.
type ArgoDAGTask struct {
	Name         string   `json:"name"`
	Template     string   `json:"template"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// ─────────────────────────────────────────────
//  Backend interface — Strategy pattern
// ─────────────────────────────────────────────

// CompiledManifest holds the output of a successful compilation.
type CompiledManifest struct {
	// Raw is the JSON-encoded CRD manifest ready to be applied to the cluster.
	Raw []byte

	// ResourceName is the Kubernetes resource name of the compiled CRD.
	ResourceName string

	// APIVersion is the API group/version of the compiled CRD.
	APIVersion string

	// Kind is the Kubernetes Kind of the compiled CRD.
	Kind string
}

// Backend is the strategy interface that abstracts the execution engine.
// Implement this interface to support alternative backends (e.g., Tekton).
type Backend interface {
	// Compile converts a FullStackApplication spec into a deployable CRD manifest.
	Compile(fsa *v1alpha1.FullStackApplication) (*CompiledManifest, error)

	// Name returns the backend identifier (e.g., "argo", "tekton").
	Name() string
}

// ─────────────────────────────────────────────
//  ArgoCompiler — default backend
// ─────────────────────────────────────────────

// ArgoCompiler compiles FullStackApplication DAGs into Argo Workflow CRDs.
// It is the default Backend implementation.
type ArgoCompiler struct {
	// DefaultImage is used for nodes that don't specify a custom image.
	DefaultImage string

	// DefaultServiceAccount is forwarded to all generated workflow pods.
	DefaultServiceAccount string
}

// NewArgoCompiler returns a pre-configured ArgoCompiler.
func NewArgoCompiler() *ArgoCompiler {
	return &ArgoCompiler{
		DefaultImage:          "busybox:latest",
		DefaultServiceAccount: "flowengine-executor",
	}
}

// Name implements Backend.
func (c *ArgoCompiler) Name() string { return "argo" }

// Compile converts the FullStackApplication WorkflowDAG into an Argo Workflow CRD manifest.
// The reconciler calls this method after the pre-deploy linter passes.
func (c *ArgoCompiler) Compile(fsa *v1alpha1.FullStackApplication) (*CompiledManifest, error) {
	dag := fsa.Spec.WorkflowDAG
	if len(dag) == 0 {
		return nil, fmt.Errorf("workflowDag must contain at least one node")
	}

	// 1. Build dependency map: nodeID → list of node IDs that feed INTO it
	deps := buildDependencyMap(dag)

	// 2. Build the shared emptyDir volume for inter-step payload passing (FR-4.4)
	payloadVolume := corev1.Volume{
		Name: "flowengine-payload",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory, // uses tmpfs for speed
			},
		},
	}

	// 3. Build one ArgoTemplate per workflow node (leaf container templates)
	var templates []ArgoTemplate
	for _, node := range dag {
		t, err := c.compileNodeTemplate(node, payloadVolume)
		if err != nil {
			return nil, fmt.Errorf("compiling node %q: %w", node.ID, err)
		}
		templates = append(templates, t)
	}

	// 4. Build the DAG orchestration template
	dagTasks := make([]ArgoDAGTask, 0, len(dag))
	for _, node := range dag {
		task := ArgoDAGTask{
			Name:         sanitizeName(node.ID),
			Template:     sanitizeName(node.ID),
			Dependencies: resolveDependencies(node.ID, deps),
		}
		dagTasks = append(dagTasks, task)
	}

	dagTemplate := ArgoTemplate{
		Name: "main",
		DAG:  &ArgoDAGTemplate{Tasks: dagTasks},
	}
	templates = append([]ArgoTemplate{dagTemplate}, templates...)

	// 5. Configure TTL
	ttl := int32(3600)
	if fsa.Spec.Execution != nil && fsa.Spec.Execution.TTLSecondsAfterFinished != nil {
		ttl = *fsa.Spec.Execution.TTLSecondsAfterFinished
	}

	// 6. Configure service account
	sa := c.DefaultServiceAccount
	if fsa.Spec.Execution != nil && fsa.Spec.Execution.ServiceAccountName != "" {
		sa = fsa.Spec.Execution.ServiceAccountName
	}

	// 7. Assemble the final Argo Workflow CRD
	workflowName := fmt.Sprintf("fsa-%s", fsa.Name)
	wf := ArgoWorkflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflowName,
			Namespace: fsa.Namespace,
			Labels: map[string]string{
				"flowengine.io/managed-by":    "flowengine-operator",
				"flowengine.io/app-name":      fsa.Name,
				"flowengine.io/app-namespace": fsa.Namespace,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: fsa.APIVersion,
					Kind:       fsa.Kind,
					Name:       fsa.Name,
					UID:        fsa.UID,
				},
			},
		},
		Spec: ArgoWorkflowSpec{
			Entrypoint:         "main",
			ServiceAccountName: sa,
			TTLStrategy: &ArgoTTLStrategy{
				SecondsAfterCompletion: &ttl,
			},
			Templates: templates,
			Volumes:   []corev1.Volume{payloadVolume},
		},
	}

	raw, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling Argo Workflow CRD: %w", err)
	}

	return &CompiledManifest{
		Raw:          raw,
		ResourceName: workflowName,
		APIVersion:   "argoproj.io/v1alpha1",
		Kind:         "Workflow",
	}, nil
}

// compileNodeTemplate converts a single WorkflowNode into an Argo container template.
func (c *ArgoCompiler) compileNodeTemplate(node v1alpha1.WorkflowNode, payloadVol corev1.Volume) (ArgoTemplate, error) {
	image := node.Image
	if image == "" {
		image = c.DefaultImage
	}

	// Build environment variables from node params (non-sensitive only)
	var envVars []corev1.EnvVar
	for k, v := range node.Params {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "FLOWENGINE_PARAM_" + k,
			Value: v,
		})
	}

	// Add envelope routing env vars
	envVars = append(envVars,
		corev1.EnvVar{Name: "FLOWENGINE_NODE_ID", Value: node.ID},
		corev1.EnvVar{Name: "FLOWENGINE_NODE_TYPE", Value: node.Type},
		corev1.EnvVar{Name: "FLOWENGINE_PAYLOAD_PATH", Value: "/data/flowengine"},
	)

	// Mount the shared payload volume
	volumeMount := corev1.VolumeMount{
		Name:      payloadVol.Name,
		MountPath: "/data/flowengine",
	}

	// Build the container spec
	container := corev1.Container{
		Name:         sanitizeName(node.ID),
		Image:        image,
		Env:          envVars,
		VolumeMounts: []corev1.VolumeMount{volumeMount},
	}

	// Apply resource limits if specified
	if node.Resources != nil {
		container.Resources = *node.Resources
	} else {
		// Apply conservative defaults to prevent runaway resource usage
		container.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		}
	}

	// Inject SecretRef as envFrom (never expose the secret values in the spec itself)
	if node.SecretRef != "" {
		container.EnvFrom = []corev1.EnvFromSource{
			{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: node.SecretRef},
				},
			},
		}
	}

	// For script/* nodes with inline scripts, mount the script as a ConfigMap command
	if node.Script != "" {
		container.Command = []string{"sh", "-c", node.Script}
	}

	return ArgoTemplate{
		Name:      sanitizeName(node.ID),
		Container: &container,
	}, nil
}

// ─────────────────────────────────────────────
//  Helpers
// ─────────────────────────────────────────────

// buildDependencyMap inverts the output edges to build "who feeds into me" maps.
// dag output edges: A → [B, C]  means "A feeds B and C"
// dependency map:   B → [A], C → [A]  means "B depends on A"
func buildDependencyMap(dag []v1alpha1.WorkflowNode) map[string][]string {
	deps := make(map[string][]string)
	for _, node := range dag {
		for _, output := range node.Outputs {
			deps[output] = append(deps[output], node.ID)
		}
	}
	return deps
}

// resolveDependencies returns the sanitized dependency names for an Argo DAG task.
func resolveDependencies(nodeID string, deps map[string][]string) []string {
	rawDeps := deps[nodeID]
	if len(rawDeps) == 0 {
		return nil
	}
	sanitized := make([]string, len(rawDeps))
	for i, d := range rawDeps {
		sanitized[i] = sanitizeName(d)
	}
	return sanitized
}

// sanitizeName ensures a node ID is a valid Argo/Kubernetes resource name.
// Argo task names must be lowercase alphanumeric + dashes.
func sanitizeName(id string) string {
	result := make([]byte, len(id))
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result[i] = c
		} else if c >= 'A' && c <= 'Z' {
			result[i] = c + 32 // to lowercase
		} else {
			result[i] = '-'
		}
	}
	return string(result)
}
