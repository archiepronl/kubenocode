// Package v1alpha1 defines the KttmApp Custom Resource Definition for the KTTM platform.
//
// KttmApp (கட்டும் — "that which binds") is the single declarative unit that unifies:
//   - ETL / Data Pipelines (Argo Workflows backend)
//   - Real-Time App Integration (NATS event mesh)
//   - No-Code UI / Forms (schema-driven SPA)
//
// API Group:  kttm.io/v1alpha1
// Kind:       KttmApp
// Scope:      Namespaced
//
// KTTM-REQ-007: Kubernetes Custom Resource Mapping
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KttmApp kind constants used by the controller and webhooks.
const (
	Kind          = "KttmApp"
	ConnectorKind = "KttmConnector"
	PluginKind    = "KttmUIPlugin"
)

// ─────────────────────────────────────────────
//  KttmApp — the top-level CRD
// ─────────────────────────────────────────────

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.mode`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type KttmApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KttmAppSpec   `json:"spec,omitempty"`
	Status KttmAppStatus `json:"status,omitempty"`
}

// KttmAppSpec defines the full application specification.
type KttmAppSpec struct {
	// DisplayName is a human-readable label shown in the Developer SDK canvas.
	DisplayName string `json:"displayName,omitempty"`

	// Version is a semantic version string for this application revision.
	// Used by GitOps and rollback operations.
	Version string `json:"version,omitempty"`

	// Mode controls how the operator provisions infrastructure (KTTM-REQ-045).
	// +kubebuilder:validation:Enum=workflow;webapp;hybrid
	// +kubebuilder:default=workflow
	Mode AppMode `json:"mode,omitempty"`

	// RBAC defines role and permission bindings for this application (KTTM-REQ-005).
	RBAC KttmRBACSpec `json:"rbac,omitempty"`

	// UILayout defines the end-user form and dashboard schema (KTTM-REQ-001, KTTM-REQ-015).
	UILayout *UILayoutSpec `json:"uiLayout,omitempty"`

	// WorkflowDAG defines the directed acyclic graph of execution nodes (KTTM-REQ-023).
	WorkflowDAG WorkflowDAGSpec `json:"workflowDag,omitempty"`

	// Execution controls how and where the workflow runs (KTTM-REQ-029, KTTM-REQ-030).
	Execution ExecutionSpec `json:"execution,omitempty"`

	// GitOps configures the automated Git repository synchronization (KTTM-REQ-008).
	GitOps *GitOpsSpec `json:"gitOps,omitempty"`

	// Debug enables interactive breakpoint debugging (KTTM-REQ-014).
	Debug *DebugSpec `json:"debug,omitempty"`

	// Policy configures OPA/Kyverno policy enforcement (KTTM-REQ-036).
	Policy *PolicySpec `json:"policy,omitempty"`
}

// AppMode defines the operational mode of a KttmApp.
type AppMode string

const (
	// AppModeWorkflow — batch or streaming data pipeline (scale-to-zero eligible).
	AppModeWorkflow AppMode = "workflow"
	// AppModeWebapp — always-active web application (minReplicas >= 1).
	AppModeWebapp AppMode = "webapp"
	// AppModeHybrid — combines persistent web endpoints with background pipelines.
	AppModeHybrid AppMode = "hybrid"
)

// ─────────────────────────────────────────────
//  RBAC
// ─────────────────────────────────────────────

// KttmRBACSpec defines custom roles and the service account for this application.
// KTTM-REQ-005, KTTM-REQ-039
type KttmRBACSpec struct {
	// Roles defines custom permission sets (combinable via Admin UI checkbox matrix).
	Roles []KttmRole `json:"roles,omitempty"`

	// ServiceAccountName is the Kubernetes ServiceAccount the workflow pods run as.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// KttmRole maps a named role to a set of atomic permissions.
type KttmRole struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"` // e.g., ["app:create", "app:debug"]
}

// KttmPermission enumerates all available atomic permissions.
// Admins compose custom roles from this master list (KTTM-REQ-005).
type KttmPermission string

const (
	PermAppCreate    KttmPermission = "app:create"
	PermAppModify    KttmPermission = "app:modify"
	PermAppDelete    KttmPermission = "app:delete"
	PermAppDebug     KttmPermission = "app:debug"
	PermAppExport    KttmPermission = "app:export"
	PermAppExecute   KttmPermission = "app:execute"
	PermRBACManage   KttmPermission = "rbac:manage"
	PermInfraInstall KttmPermission = "infra:install"
	PermInfraUpgrade KttmPermission = "infra:upgrade"
	PermBundleImport KttmPermission = "bundle:import"
	PermAuditView    KttmPermission = "audit:view"
	PermCostView     KttmPermission = "cost:view"
)

// AllPermissions is the master permission list presented in the Admin checkbox matrix.
var AllPermissions = []KttmPermission{
	PermAppCreate, PermAppModify, PermAppDelete, PermAppDebug,
	PermAppExport, PermAppExecute, PermRBACManage, PermInfraInstall,
	PermInfraUpgrade, PermBundleImport, PermAuditView, PermCostView,
}

// DefaultRoles provides the three built-in roles.
var DefaultRoles = map[string][]KttmPermission{
	"developer": {PermAppCreate, PermAppModify, PermAppDelete, PermAppDebug, PermAppExport, PermAppExecute, PermAuditView},
	"admin":     {PermRBACManage, PermInfraInstall, PermInfraUpgrade, PermBundleImport, PermAppExport, PermAppExecute, PermAuditView, PermCostView},
	"enduser":   {PermAppExecute},
}

// ─────────────────────────────────────────────
//  UI Layout
// ─────────────────────────────────────────────

// UILayoutSpec defines the schema-driven end-user interface (KTTM-REQ-001, REQ-004, REQ-015).
type UILayoutSpec struct {
	// Type controls the layout template rendered for end-users.
	// +kubebuilder:validation:Enum=form;dashboard;hybrid
	Type string `json:"type,omitempty"`

	// Schema is a JSON Schema string rendered at runtime by the web-renderer SPA.
	// Stored in a ConfigMap for hot-reload without CRD changes (KTTM-REQ-017).
	Schema string `json:"schema,omitempty"`

	// SchemaConfigMapRef points to a ConfigMap containing the JSON schema.
	// Takes precedence over inline Schema.
	SchemaConfigMapRef *corev1.ConfigMapKeySelector `json:"schemaConfigMapRef,omitempty"`

	// CustomReactBundle references a ConfigMap containing a Webpack Module Federation
	// remote entry for a custom React UI bundle (KTTM-REQ-016).
	CustomReactBundle *corev1.ConfigMapKeySelector `json:"customReactBundle,omitempty"`
}

// ─────────────────────────────────────────────
//  Workflow DAG
// ─────────────────────────────────────────────

// WorkflowDAGSpec defines the full execution graph (KTTM-REQ-023).
type WorkflowDAGSpec struct {
	// Nodes is the list of execution steps.
	Nodes []WorkflowNode `json:"nodes,omitempty"`

	// ParallelGroups defines fan-out groups with optional barrier synchronization (KTTM-REQ-023).
	ParallelGroups []ParallelGroup `json:"parallelGroups,omitempty"`

	// Edges defines the directed connections between nodes.
	Edges []DAGEdge `json:"edges,omitempty"`
}

// ParallelGroup defines a fan-out execution group (KTTM-REQ-023).
type ParallelGroup struct {
	ID           string   `json:"id"`
	Nodes        []string `json:"nodes"`
	BarrierAfter bool     `json:"barrierAfter,omitempty"` // Wait for all before continuing
}

// DAGEdge connects two nodes in the workflow graph.
type DAGEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	// Label is optional — displayed on canvas edge.
	Label string `json:"label,omitempty"`
}

// ─────────────────────────────────────────────
//  Execution
// ─────────────────────────────────────────────

// ExecutionSpec configures the execution backend and scaling (KTTM-REQ-029, REQ-030, REQ-043).
type ExecutionSpec struct {
	// Backend selects the workflow orchestration engine.
	// +kubebuilder:validation:Enum=argo;tekton
	// +kubebuilder:default=argo
	Backend string `json:"backend,omitempty"`

	// Scaling configures KEDA horizontal auto-scaling (KTTM-REQ-029).
	Scaling *ScalingSpec `json:"scaling,omitempty"`

	// VerticalResize enables in-place vertical scaling without pod restart (KTTM-REQ-030).
	// Requires Kubernetes 1.27+ with InPlacePodVerticalScaling feature gate.
	// +kubebuilder:validation:Enum=inplace;restart
	// +kubebuilder:default=restart
	VerticalResize string `json:"verticalResize,omitempty"`

	// ServiceAccountName is the Kubernetes ServiceAccount the Argo workflow steps run as.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// TTLSecondsAfterFinished defines how long to keep the Argo Workflow CRD after completion.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// ImagePullSecrets are used to pull private container images.
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// ScalingSpec configures KEDA-driven auto-scaling (KTTM-REQ-029).
type ScalingSpec struct {
	// Type selects the scaler implementation.
	// +kubebuilder:validation:Enum=keda;hpa;none
	// +kubebuilder:default=keda
	Type string `json:"type,omitempty"`

	// NATSSubject is the NATS subject monitored for queue depth.
	NATSSubject string `json:"natsSubject,omitempty"`

	// LagThreshold is the number of pending messages that triggers a scale-up event.
	// +kubebuilder:default=50
	LagThreshold int32 `json:"lagThreshold,omitempty"`

	// MinReplicas is the minimum pod count (0 = scale-to-zero; webapp mode forces >= 1).
	// +kubebuilder:default=0
	MinReplicas int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the maximum pod count.
	// +kubebuilder:default=10
	MaxReplicas int32 `json:"maxReplicas,omitempty"`
}

// ─────────────────────────────────────────────
//  GitOps
// ─────────────────────────────────────────────

// GitOpsSpec configures automated Git repository synchronization (KTTM-REQ-008).
type GitOpsSpec struct {
	// Repo is the SSH Git repository URL.
	Repo string `json:"repo"`
	// Branch is the target Git branch (default: main).
	Branch string `json:"branch,omitempty"`
	// Path is the directory path inside the repo where the YAML file is committed.
	Path string `json:"path,omitempty"`
	// SSHKeySecretRef is the Kubernetes Secret name containing the SSH private key.
	SSHKeySecretRef string `json:"sshKeySecretRef,omitempty"`
	// Backend selects the GitOps pull agent running in downstream clusters.
	// +kubebuilder:validation:Enum=argocd;flux;raw
	Backend string `json:"backend,omitempty"`
}

// ─────────────────────────────────────────────
//  Debug
// ─────────────────────────────────────────────

// DebugSpec enables interactive breakpoint debugging (KTTM-REQ-014).
type DebugSpec struct {
	// Enabled injects the kttm-debug-sidecar into Argo step pods when true.
	Enabled bool `json:"enabled,omitempty"`
	// StreamLogs enables real-time log streaming to the Developer SDK.
	StreamLogs bool `json:"streamLogs,omitempty"`
}

// ─────────────────────────────────────────────
//  Policy
// ─────────────────────────────────────────────

// PolicySpec configures policy enforcement for this application (KTTM-REQ-036).
type PolicySpec struct {
	// Engine selects the policy enforcement backend.
	// +kubebuilder:validation:Enum=kyverno;opa;none
	// +kubebuilder:default=kyverno
	Engine string `json:"engine,omitempty"`
	// PolicyRefs are names of Kyverno ClusterPolicies or OPA constraint templates to apply.
	PolicyRefs []string `json:"policyRefs,omitempty"`
}

// ─────────────────────────────────────────────
//  Status
// ─────────────────────────────────────────────

// KttmAppStatus reflects the last observed state of the KttmApp.
type KttmAppStatus struct {
	// Phase is the overall lifecycle state.
	// +kubebuilder:validation:Enum=Pending;Linting;Building;Deploying;Running;Succeeded;Failed;Unknown
	Phase string `json:"phase,omitempty"`

	// NodeStatuses tracks the execution state of each workflow node.
	NodeStatuses map[string]string `json:"nodeStatuses,omitempty"`

	// LinterResult contains the output of the pre-deploy GraphLinter.
	LinterResult *LinterResult `json:"linterResult,omitempty"`

	// ScanResult contains the Trivy CVE scan summary.
	ScanResult *ScanResult `json:"scanResult,omitempty"`

	// Conditions are standard Kubernetes condition arrays.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration tracks the last reconciled CRD generation.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ArgoWorkflowName is the name of the Argo Workflow CRD created for the last execution.
	ArgoWorkflowName string `json:"argoWorkflowName,omitempty"`

	// GitCommit is the SHA of the last GitOps commit pushed by the operator.
	GitCommit string `json:"gitCommit,omitempty"`
}

// LinterResult summarises the pre-deploy static analysis (KTTM-REQ-025).
type LinterResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ScanResult summarises the Trivy image CVE scan (KTTM-REQ-046).
type ScanResult struct {
	Clean     bool   `json:"clean"`
	Critical  int    `json:"critical"`
	High      int    `json:"high"`
	Medium    int    `json:"medium"`
	ReportURL string `json:"reportUrl,omitempty"`
}

// ─────────────────────────────────────────────
//  KttmApp List
// ─────────────────────────────────────────────

// +kubebuilder:object:root=true
type KttmAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KttmApp `json:"items"`
}
