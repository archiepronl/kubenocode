// Package v1alpha1 defines the FullStackApplication CRD for the FlowEngine platform.
// It models the unified declarative schema that drives all three user tiers.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=fsa,categories=flowengine
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Nodes",type=integer,JSONPath=`.status.nodeCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// FullStackApplication is the core CRD that models a complete workflow application.
// It encodes the visual DAG, the no-code UI schema, GitOps configuration, and
// connector secrets references — all in a single declarative resource.
type FullStackApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FullStackApplicationSpec   `json:"spec,omitempty"`
	Status FullStackApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FullStackApplicationList contains a list of FullStackApplication resources.
type FullStackApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FullStackApplication `json:"items"`
}

// FullStackApplicationSpec defines the desired state of the FullStackApplication.
type FullStackApplicationSpec struct {
	// WorkflowDAG describes the directed acyclic graph of execution nodes.
	// Each node represents a step in the pipeline (connector, script, transform, etc.).
	// +kubebuilder:validation:MinItems=1
	WorkflowDAG []WorkflowNode `json:"workflowDag"`

	// UILayoutSchema is a raw JSON string holding the declarative form/UI configuration.
	// The schema-driven SPA reads this to dynamically render the end-user interface
	// without any compilation step. Empty string disables the UI layer.
	// +optional
	UILayoutSchema string `json:"uiLayoutSchema,omitempty"`

	// GitOps configures automatic commit-and-sync of compiled workflow manifests
	// to a GitOps repository (Argo CD / Flux).
	// +optional
	GitOps *GitOpsConfig `json:"gitOps,omitempty"`

	// Execution controls the low-level execution engine configuration.
	// +optional
	Execution *ExecutionConfig `json:"execution,omitempty"`
}

// WorkflowNode represents a single execution unit in the workflow DAG.
// It can be a data source connector, a transformation script,
// a media processing step, or a sink.
type WorkflowNode struct {
	// ID is the unique identifier for this node within the DAG.
	// Used to reference this node as an input/output in other nodes.
	// +kubebuilder:validation:Pattern=`^[a-z0-9][a-z0-9\-]*[a-z0-9]$`
	ID string `json:"id"`

	// Type identifies which connector or processor handles this node.
	// Format: "<category>/<adapter>", e.g. "connector/s3", "script/python", "transform/filter".
	// +kubebuilder:validation:Pattern=`^[a-z]+/[a-z0-9\-]+$`
	Type string `json:"type"`

	// Label is a human-readable display name shown on the visual canvas.
	// +optional
	Label string `json:"label,omitempty"`

	// Params holds adapter-specific configuration parameters.
	// These are non-sensitive values (bucket names, table names, filters, etc.).
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Params map[string]string `json:"params,omitempty"`

	// SecretRef names a Kubernetes Secret in the same namespace that holds
	// sensitive credentials (API keys, DB passwords, certificates).
	// NEVER serialized into exported bundles — always resolved at runtime.
	// +optional
	SecretRef string `json:"secretRef,omitempty"`

	// Image specifies a custom Docker image URI for script/custom nodes.
	// The image is used as the container image for the Argo Workflow step.
	// +optional
	Image string `json:"image,omitempty"`

	// Script holds an inline script body for script/* node types.
	// Supported languages are determined by the Image specified.
	// +optional
	Script string `json:"script,omitempty"`

	// Resources defines CPU and memory limits/requests for the execution container.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Outputs lists the IDs of downstream nodes that receive this node's output envelope.
	// Defines the directed edges of the DAG.
	// +optional
	Outputs []string `json:"outputs,omitempty"`

	// MimeTypeHint provides an optional hint about expected output mime type.
	// Used by the Materializer to pre-select the correct adapter factory.
	// +optional
	MimeTypeHint string `json:"mimeTypeHint,omitempty"`

	// NodeAffinity and Tolerations allow Expert-tier users to pin specific
	// heavy-compute nodes (e.g., GPU FFmpeg nodes) to specific cluster nodes.
	// +optional
	NodeAffinity *corev1.NodeAffinity `json:"nodeAffinity,omitempty"`

	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// GitOpsConfig configures GitOps-driven replication of compiled manifests.
type GitOpsConfig struct {
	// Repo is the SSH or HTTPS URL of the target Git repository.
	// +kubebuilder:validation:Pattern=`^(https?|git|ssh)://`
	Repo string `json:"repo"`

	// Branch is the target branch name for commits.
	// +kubebuilder:default="main"
	Branch string `json:"branch,omitempty"`

	// Path is the directory path within the repository where manifests are committed.
	// +kubebuilder:default="workflows/"
	Path string `json:"path,omitempty"`

	// SSHKeySecretRef names a Kubernetes Secret containing the SSH private key
	// used to authenticate with the Git repository.
	// +optional
	SSHKeySecretRef string `json:"sshKeySecretRef,omitempty"`
}

// ExecutionConfig provides low-level overrides for the execution backend.
type ExecutionConfig struct {
	// Backend selects the execution engine. "argo" (default) or "tekton".
	// +kubebuilder:default="argo"
	// +kubebuilder:validation:Enum=argo;tekton
	Backend string `json:"backend,omitempty"`

	// Namespace is the target namespace where Argo Workflow CRDs are created.
	// Defaults to the same namespace as the FullStackApplication.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// ServiceAccountName is the K8s service account used by execution pods.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// ImagePullSecrets are forwarded to all generated workflow step pods.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// TTLSecondsAfterFinished controls how long completed Argo Workflow resources
	// are retained before automatic cleanup.
	// +optional
	// +kubebuilder:default=3600
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
}

// FullStackApplicationStatus reflects the observed state of the FullStackApplication.
type FullStackApplicationStatus struct {
	// Phase represents the current lifecycle phase of the application.
	// +kubebuilder:validation:Enum=Pending;Compiling;Deploying;Running;Succeeded;Failed;Unknown
	Phase string `json:"phase,omitempty"`

	// NodeCount is the number of valid nodes detected in the WorkflowDAG.
	NodeCount int `json:"nodeCount,omitempty"`

	// CompiledWorkflowRef references the Argo Workflow CRD generated from this FSA.
	// +optional
	CompiledWorkflowRef string `json:"compiledWorkflowRef,omitempty"`

	// LastCompiledAt records when the DAG was last successfully compiled.
	// +optional
	LastCompiledAt *metav1.Time `json:"lastCompiledAt,omitempty"`

	// LintErrors holds any pre-deploy validation errors discovered by the static linter.
	// If non-empty, the reconciler will NOT deploy the workflow.
	// +optional
	LintErrors []string `json:"lintErrors,omitempty"`

	// Conditions provides detailed state machine conditions for fine-grained status reporting.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the generation of the spec that produced this status.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}
