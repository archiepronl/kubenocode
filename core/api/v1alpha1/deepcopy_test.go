package v1alpha1

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestFullStackApplicationDeepCopiesNestedFields(t *testing.T) {
	ttl := int32(120)
	compiledAt := metav1.NewTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	original := &FullStackApplication{
		TypeMeta:   metav1.TypeMeta{APIVersion: GroupVersion.String(), Kind: "FullStackApplication"},
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "apps", Labels: map[string]string{"team": "data"}},
		Spec: FullStackApplicationSpec{
			WorkflowDAG: []WorkflowNode{{
				ID: "source", Params: map[string]string{"bucket": "orders"}, Outputs: []string{"sink"},
				Resources: &corev1.ResourceRequirements{}, NodeAffinity: &corev1.NodeAffinity{},
				Tolerations: []corev1.Toleration{{Key: "dedicated", Value: "data"}},
			}},
			GitOps:    &GitOpsConfig{Repo: "https://example.invalid/flows"},
			Execution: &ExecutionConfig{ImagePullSecrets: []corev1.LocalObjectReference{{Name: "registry"}}, TTLSecondsAfterFinished: &ttl},
		},
		Status: FullStackApplicationStatus{
			Phase: "Running", LastCompiledAt: &compiledAt, LintErrors: []string{"warning"},
			Conditions: []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}},
		},
	}

	copy := original.DeepCopy()
	if copy == original || copy.Spec.WorkflowDAG[0].Params["bucket"] != "orders" || copy.Spec.GitOps == original.Spec.GitOps {
		t.Fatalf("DeepCopy() did not produce an independent object: %+v", copy)
	}
	copy.ObjectMeta.Labels["team"] = "changed"
	copy.Spec.WorkflowDAG[0].Params["bucket"] = "changed"
	copy.Spec.WorkflowDAG[0].Outputs[0] = "changed"
	copy.Spec.WorkflowDAG[0].Tolerations[0].Value = "changed"
	*copy.Spec.Execution.TTLSecondsAfterFinished = 999
	copy.Status.LintErrors[0] = "changed"
	copy.Status.Conditions[0].Type = "changed"
	if original.Labels["team"] != "data" || original.Spec.WorkflowDAG[0].Params["bucket"] != "orders" ||
		original.Spec.WorkflowDAG[0].Outputs[0] != "sink" || original.Spec.WorkflowDAG[0].Tolerations[0].Value != "data" ||
		*original.Spec.Execution.TTLSecondsAfterFinished != ttl || original.Status.LintErrors[0] != "warning" || original.Status.Conditions[0].Type != "Ready" {
		t.Fatal("FullStackApplication deep copy aliases mutable source data")
	}

	into := &FullStackApplication{}
	original.DeepCopyInto(into)
	if into.Name != original.Name || into.Spec.WorkflowDAG[0].Resources == original.Spec.WorkflowDAG[0].Resources {
		t.Fatal("DeepCopyInto() did not copy metadata and resources")
	}
	if _, ok := original.DeepCopyObject().(*FullStackApplication); !ok {
		t.Fatal("DeepCopyObject() returned the wrong object type")
	}
	if ((*FullStackApplication)(nil)).DeepCopy() != nil || ((*FullStackApplication)(nil)).DeepCopyObject() != nil {
		t.Fatal("nil FullStackApplication deep copy should return nil")
	}
}

func TestFullStackApplicationListDeepCopiesItems(t *testing.T) {
	original := &FullStackApplicationList{Items: []FullStackApplication{{ObjectMeta: metav1.ObjectMeta{Name: "one"}, Spec: FullStackApplicationSpec{WorkflowDAG: []WorkflowNode{{ID: "node"}}}}}}
	copy := original.DeepCopy()
	copy.Items[0].Name = "changed"
	copy.Items[0].Spec.WorkflowDAG[0].ID = "changed"
	if original.Items[0].Name != "one" || original.Items[0].Spec.WorkflowDAG[0].ID != "node" {
		t.Fatal("FullStackApplicationList deep copy aliases items")
	}
	into := &FullStackApplicationList{}
	original.DeepCopyInto(into)
	if into.Items[0].Name != "one" {
		t.Fatal("FullStackApplicationList.DeepCopyInto() lost items")
	}
	if _, ok := original.DeepCopyObject().(*FullStackApplicationList); !ok {
		t.Fatal("FullStackApplicationList.DeepCopyObject() returned wrong type")
	}
	if ((*FullStackApplicationList)(nil)).DeepCopy() != nil || ((*FullStackApplicationList)(nil)).DeepCopyObject() != nil {
		t.Fatal("nil FullStackApplicationList deep copy should return nil")
	}
}

func TestKttmAppDeepCopiesNestedFields(t *testing.T) {
	ttl := int32(45)
	original := &KttmApp{
		TypeMeta:   metav1.TypeMeta{APIVersion: GroupVersion.String(), Kind: Kind},
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "team-a", Annotations: map[string]string{"owner": "data"}},
		Spec: KttmAppSpec{
			DisplayName: "Orders", Version: "1.0.0", Mode: AppModeWorkflow,
			RBAC: KttmRBACSpec{ServiceAccountName: "runner", Roles: []KttmRole{{Name: "operator", Permissions: []string{"app:execute"}}}},
			UILayout: &UILayoutSpec{
				Schema: `{"type":"object"}`,
				SchemaConfigMapRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "schema"}, Key: "schema.json"},
				CustomReactBundle:  &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "bundle"}, Key: "remote.js"},
			},
			WorkflowDAG: WorkflowDAGSpec{
				Nodes:          []WorkflowNode{{ID: "script", Params: map[string]string{"mode": "fast"}, Outputs: []string{"sink"}, Tolerations: []corev1.Toleration{{Key: "gpu"}}}},
				ParallelGroups: []ParallelGroup{{ID: "fanout", Nodes: []string{"script", "sink"}}},
				Edges:          []DAGEdge{{From: "script", To: "sink"}},
			},
			Execution: ExecutionSpec{
				Backend: "argo", Scaling: &ScalingSpec{Type: "keda", MaxReplicas: 4},
				TTLSecondsAfterFinished: &ttl, ImagePullSecrets: []corev1.LocalObjectReference{{Name: "registry"}},
			},
			GitOps: &GitOpsSpec{Repo: "ssh://git@example.invalid/flows", SSHKeySecretRef: "git-key"},
			Debug:  &DebugSpec{Enabled: true, StreamLogs: true},
			Policy: &PolicySpec{Engine: "kyverno", PolicyRefs: []string{"restricted"}},
		},
		Status: KttmAppStatus{
			Phase: "Running", NodeStatuses: map[string]string{"script": "Succeeded"},
			LinterResult: &LinterResult{Valid: true, Warnings: []string{"notice"}},
			ScanResult:   &ScanResult{Clean: true},
			Conditions:   []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}},
		},
	}

	copy := original.DeepCopy()
	if copy == original || copy.Spec.UILayout == original.Spec.UILayout || copy.Spec.Execution.Scaling == original.Spec.Execution.Scaling {
		t.Fatal("KttmApp.DeepCopy() did not allocate nested pointers")
	}
	copy.Annotations["owner"] = "changed"
	copy.Spec.RBAC.Roles[0].Permissions[0] = "changed"
	copy.Spec.UILayout.SchemaConfigMapRef.Name = "changed"
	copy.Spec.WorkflowDAG.Nodes[0].Params["mode"] = "changed"
	copy.Spec.WorkflowDAG.Nodes[0].Tolerations[0].Key = "changed"
	copy.Spec.WorkflowDAG.ParallelGroups[0].Nodes[0] = "changed"
	copy.Spec.WorkflowDAG.Edges[0].From = "changed"
	*copy.Spec.Execution.TTLSecondsAfterFinished = 1
	copy.Spec.Execution.ImagePullSecrets[0].Name = "changed"
	copy.Spec.Policy.PolicyRefs[0] = "changed"
	copy.Status.NodeStatuses["script"] = "changed"
	copy.Status.LinterResult.Warnings[0] = "changed"
	copy.Status.Conditions[0].Type = "changed"
	if original.Annotations["owner"] != "data" || original.Spec.RBAC.Roles[0].Permissions[0] != "app:execute" ||
		original.Spec.UILayout.SchemaConfigMapRef.Name != "schema" || original.Spec.WorkflowDAG.Nodes[0].Params["mode"] != "fast" ||
		original.Spec.WorkflowDAG.Nodes[0].Tolerations[0].Key != "gpu" || original.Spec.WorkflowDAG.ParallelGroups[0].Nodes[0] != "script" ||
		original.Spec.WorkflowDAG.Edges[0].From != "script" || *original.Spec.Execution.TTLSecondsAfterFinished != ttl ||
		original.Spec.Execution.ImagePullSecrets[0].Name != "registry" || original.Spec.Policy.PolicyRefs[0] != "restricted" ||
		original.Status.NodeStatuses["script"] != "Succeeded" || original.Status.LinterResult.Warnings[0] != "notice" || original.Status.Conditions[0].Type != "Ready" {
		t.Fatal("KttmApp deep copy aliases mutable source data")
	}

	into := &KttmApp{}
	original.DeepCopyInto(into)
	if into.Name != original.Name || into.Spec.WorkflowDAG.Nodes[0].ID != "script" {
		t.Fatal("KttmApp.DeepCopyInto() lost source fields")
	}
	if _, ok := original.DeepCopyObject().(*KttmApp); !ok {
		t.Fatal("KttmApp.DeepCopyObject() returned wrong type")
	}
	if ((*KttmApp)(nil)).DeepCopy() != nil || ((*KttmApp)(nil)).DeepCopyObject() != nil || ((*KttmAppStatus)(nil)).DeepCopy() != nil {
		t.Fatal("nil KttmApp deep copy should return nil")
	}
}

func TestKttmAppListAndNestedCopyMethods(t *testing.T) {
	list := &KttmAppList{Items: []KttmApp{{ObjectMeta: metav1.ObjectMeta{Name: "one"}, Spec: KttmAppSpec{WorkflowDAG: WorkflowDAGSpec{Nodes: []WorkflowNode{{ID: "node"}}}}}}}
	copy := list.DeepCopy()
	copy.Items[0].Spec.WorkflowDAG.Nodes[0].ID = "changed"
	if list.Items[0].Spec.WorkflowDAG.Nodes[0].ID != "node" {
		t.Fatal("KttmAppList deep copy aliases items")
	}
	intoList := &KttmAppList{}
	list.DeepCopyInto(intoList)
	if intoList.Items[0].Name != "one" {
		t.Fatal("KttmAppList.DeepCopyInto() lost items")
	}
	if _, ok := list.DeepCopyObject().(*KttmAppList); !ok {
		t.Fatal("KttmAppList.DeepCopyObject() returned wrong type")
	}
	if ((*KttmAppList)(nil)).DeepCopy() != nil || ((*KttmAppList)(nil)).DeepCopyObject() != nil || ((*KttmAppSpec)(nil)).DeepCopy() != nil || ((*KttmAppStatus)(nil)).DeepCopy() != nil {
		t.Fatal("nil KttmApp nested deep copy should return nil")
	}

	var role KttmRole
	(&KttmRole{Permissions: []string{"app:create"}}).DeepCopyInto(&role)
	role.Permissions[0] = "changed"
	var roles KttmRBACSpec
	(&KttmRBACSpec{Roles: []KttmRole{{Name: "admin", Permissions: []string{"rbac:manage"}}}}).DeepCopyInto(&roles)
	roles.Roles[0].Permissions[0] = "changed"
	var layout UILayoutSpec
	(&UILayoutSpec{SchemaConfigMapRef: &corev1.ConfigMapKeySelector{Key: "schema"}, CustomReactBundle: &corev1.ConfigMapKeySelector{Key: "bundle"}}).DeepCopyInto(&layout)
	var group ParallelGroup
	(&ParallelGroup{Nodes: []string{"one"}}).DeepCopyInto(&group)
	group.Nodes[0] = "changed"
	var execution ExecutionSpec
	(&ExecutionSpec{Scaling: &ScalingSpec{MaxReplicas: 2}, TTLSecondsAfterFinished: ptrInt32(7), ImagePullSecrets: []corev1.LocalObjectReference{{Name: "secret"}}}).DeepCopyInto(&execution)
	var result LinterResult
	(&LinterResult{Errors: []string{"error"}, Warnings: []string{"warning"}}).DeepCopyInto(&result)
	var policy PolicySpec
	(&PolicySpec{PolicyRefs: []string{"policy"}}).DeepCopyInto(&policy)
	if role.Permissions[0] != "changed" || roles.Roles[0].Permissions[0] != "changed" || layout.SchemaConfigMapRef == nil ||
		group.Nodes[0] != "changed" || execution.Scaling == nil || execution.TTLSecondsAfterFinished == nil ||
		len(result.Errors) != 1 || len(result.Warnings) != 1 || len(policy.PolicyRefs) != 1 {
		t.Fatal("nested DeepCopyInto methods did not retain source fields")
	}
}

func TestAddToSchemeRegistersBothResourceKinds(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}
	for _, kind := range []string{"FullStackApplication", "FullStackApplicationList", "KttmApp", "KttmAppList"} {
		if _, err := scheme.New(GroupVersion.WithKind(kind)); err != nil {
			t.Errorf("scheme.New(%s) error = %v", kind, err)
		}
	}
}

func TestIndividualDeepCopyConvenienceMethods(t *testing.T) {
	if (&FullStackApplicationSpec{WorkflowDAG: []WorkflowNode{{ID: "node"}}}).DeepCopy().WorkflowDAG[0].ID != "node" {
		t.Fatal("FullStackApplicationSpec.DeepCopy() lost nested fields")
	}
	if (&WorkflowNode{Params: map[string]string{"key": "value"}}).DeepCopy().Params["key"] != "value" {
		t.Fatal("WorkflowNode.DeepCopy() lost params")
	}
	if (&GitOpsConfig{Repo: "https://example.invalid"}).DeepCopy().Repo == "" {
		t.Fatal("GitOpsConfig.DeepCopy() lost repo")
	}
	var gitOpsCopy GitOpsConfig
	(&GitOpsConfig{Repo: "https://example.invalid"}).DeepCopyInto(&gitOpsCopy)
	if gitOpsCopy.Repo == "" {
		t.Fatal("GitOpsConfig.DeepCopyInto() lost repo")
	}
	if (&ExecutionConfig{ImagePullSecrets: []corev1.LocalObjectReference{{Name: "pull"}}}).DeepCopy().ImagePullSecrets[0].Name != "pull" {
		t.Fatal("ExecutionConfig.DeepCopy() lost image pull secrets")
	}
	if (&FullStackApplicationStatus{LintErrors: []string{"error"}}).DeepCopy().LintErrors[0] != "error" {
		t.Fatal("FullStackApplicationStatus.DeepCopy() lost lint errors")
	}
	if (&KttmAppSpec{DisplayName: "Orders"}).DeepCopy().DisplayName != "Orders" {
		t.Fatal("KttmAppSpec.DeepCopy() lost display name")
	}
	if (&KttmAppStatus{NodeStatuses: map[string]string{"node": "Ready"}}).DeepCopy().NodeStatuses["node"] != "Ready" {
		t.Fatal("KttmAppStatus.DeepCopy() lost node statuses")
	}
	if ((*FullStackApplicationSpec)(nil)).DeepCopy() != nil || ((*WorkflowNode)(nil)).DeepCopy() != nil ||
		((*GitOpsConfig)(nil)).DeepCopy() != nil || ((*ExecutionConfig)(nil)).DeepCopy() != nil ||
		((*FullStackApplicationStatus)(nil)).DeepCopy() != nil {
		t.Fatal("nil generated DeepCopy() method should return nil")
	}
}

func ptrInt32(value int32) *int32 { return &value }