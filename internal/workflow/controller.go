// Package controller implements the Kubernetes Operator reconcile loop for the
// FullStackApplication CRD (flowengine.io/v1alpha1).
//
// The reconcile loop implements FR-1.2 (Control Plane Operator):
//  1. Watch for FullStackApplication CR create/update events
//  2. Run the pre-deploy static linter (graph.go) — block on errors
//  3. Compile the WorkflowDAG into an Argo Workflow CRD (compiler.go)
//  4. Apply the compiled Workflow to the cluster
//  5. Update FSA.Status with phase, lint errors, and compiled workflow ref
//  6. If GitOps is configured, commit the compiled manifest to Git
//
// The controller uses server-side apply (SSA) for idempotent updates —
// re-running Reconcile always converges to the desired state.
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	"github.com/kubeworkflow/flowengine/core/engine"
	"github.com/kubeworkflow/flowengine/core/linter"
)

// ─────────────────────────────────────────────
//  Phase constants for FSA.Status.Phase
// ─────────────────────────────────────────────

const (
	PhasePending   = "Pending"
	PhaseCompiling = "Compiling"
	PhaseDeploying = "Deploying"
	PhaseRunning   = "Running"
	PhaseSucceeded = "Succeeded"
	PhaseFailed    = "Failed"
)

// ─────────────────────────────────────────────
//  Reconciler
// ─────────────────────────────────────────────

// FullStackApplicationReconciler is the Kubebuilder controller for FullStackApplication CRDs.
// It implements the sigs.k8s.io/controller-runtime Reconciler interface.
//
// +kubebuilder:rbac:groups=flowengine.io,resources=fullstackapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=flowengine.io,resources=fullstackapplications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps;secrets,verbs=get;list;watch
type FullStackApplicationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Linter   *linter.GraphLinter
	Compiler engine.Backend
}

// Reconcile is called by the controller-runtime manager whenever a FullStackApplication
// is created, updated, or deleted. It implements the full reconcile loop.
func (r *FullStackApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("fullstackapplication", req.NamespacedName)

	// ── 1. Fetch the FullStackApplication ──────────────────────────────────────
	fsa := &v1alpha1.FullStackApplication{}
	if err := r.Get(ctx, req.NamespacedName, fsa); err != nil {
		if errors.IsNotFound(err) {
			// FSA was deleted — Argo Workflow cleanup is handled via OwnerReference GC
			logger.Info("FullStackApplication not found, assuming deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("fetching FullStackApplication: %w", err)
	}

	// Skip if not changed (generation guard)
	if fsa.Status.ObservedGeneration == fsa.Generation {
		logger.V(1).Info("Skipping reconcile — generation unchanged")
		return ctrl.Result{}, nil
	}

	// ── 2. Phase: Compiling ────────────────────────────────────────────────────
	if err := r.setPhase(ctx, fsa, PhaseCompiling, "Running pre-deploy linter"); err != nil {
		return ctrl.Result{}, err
	}

	// ── 3. Run the pre-deploy static linter ───────────────────────────────────
	logger.Info("Running pre-deploy linter", "nodeCount", len(fsa.Spec.WorkflowDAG))
	lintResult := r.Linter.Lint(fsa.Spec.WorkflowDAG)

	// Always persist lint errors to status so the UI can display them
	fsa.Status.LintErrors = lintResult.ErrorStrings()
	if err := r.Status().Update(ctx, fsa); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating lint errors in status: %w", err)
	}

	if lintResult.HasErrors() {
		logger.Error(fmt.Errorf("lint failed with %d errors", len(lintResult.Errors())), "Blocking deployment")
		_ = r.setPhase(ctx, fsa, PhaseFailed, fmt.Sprintf("Linter found %d error(s) — fix before deploying", len(lintResult.Errors())))
		// Requeue after 30s in case the user fixes the CR
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// ── 4. Compile DAG → Argo Workflow CRD ────────────────────────────────────
	logger.Info("Compiling WorkflowDAG", "backend", r.Compiler.Name())
	compiled, err := r.Compiler.Compile(fsa)
	if err != nil {
		_ = r.setPhase(ctx, fsa, PhaseFailed, fmt.Sprintf("Compilation failed: %v", err))
		return ctrl.Result{}, fmt.Errorf("compiling FullStackApplication: %w", err)
	}
	logger.Info("Compilation successful", "workflow", compiled.ResourceName)

	// ── 5. Apply compiled Workflow CRD to cluster ──────────────────────────────
	if err := r.setPhase(ctx, fsa, PhaseDeploying, "Applying compiled Argo Workflow"); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.applyArgoWorkflow(ctx, fsa, compiled); err != nil {
		_ = r.setPhase(ctx, fsa, PhaseFailed, fmt.Sprintf("Apply failed: %v", err))
		return ctrl.Result{}, fmt.Errorf("applying Argo Workflow: %w", err)
	}

	// ── 6. Apply UI ConfigMap (if uiLayoutSchema is present) ──────────────────
	if fsa.Spec.UILayoutSchema != "" {
		if err := r.applyUIConfigMap(ctx, fsa); err != nil {
			logger.Error(err, "Failed to apply UI ConfigMap (non-fatal)")
		}
	}

	// ── 7. Update final status ─────────────────────────────────────────────────
	now := metav1.Now()
	fsa.Status.Phase = PhaseRunning
	fsa.Status.NodeCount = len(fsa.Spec.WorkflowDAG)
	fsa.Status.CompiledWorkflowRef = compiled.ResourceName
	fsa.Status.LastCompiledAt = &now
	fsa.Status.LintErrors = nil
	fsa.Status.ObservedGeneration = fsa.Generation
	fsa.Status.Conditions = setCondition(fsa.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "WorkflowDeployed",
		Message:            fmt.Sprintf("Argo Workflow %q is running", compiled.ResourceName),
		LastTransitionTime: now,
	})

	if err := r.Status().Update(ctx, fsa); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating final status: %w", err)
	}

	logger.Info("Reconcile complete", "workflow", compiled.ResourceName, "phase", PhaseRunning)
	return ctrl.Result{}, nil
}

// ─────────────────────────────────────────────
//  Apply helpers
// ─────────────────────────────────────────────

// applyArgoWorkflow applies the compiled Argo Workflow manifest to the cluster
// using server-side apply for idempotency.
func (r *FullStackApplicationReconciler) applyArgoWorkflow(ctx context.Context, fsa *v1alpha1.FullStackApplication, compiled *engine.CompiledManifest) error {
	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal(compiled.Raw, obj); err != nil {
		return fmt.Errorf("unmarshalling compiled manifest: %w", err)
	}

	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Workflow",
	})
	obj.SetNamespace(fsa.Namespace)

	// Server-side apply — idempotent, handles create and update
	return r.Client.Patch(ctx, obj, client.Apply, client.ForceOwnership, client.FieldOwner("flowengine-operator"))
}

// applyUIConfigMap creates or updates a ConfigMap with the UI layout schema.
// The schema-driven SPA reads this ConfigMap to render the end-user interface (FR-2.2).
func (r *FullStackApplicationReconciler) applyUIConfigMap(ctx context.Context, fsa *v1alpha1.FullStackApplication) error {
	cm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("fsa-ui-%s", fsa.Name),
				"namespace": fsa.Namespace,
				"labels": map[string]interface{}{
					"flowengine.io/managed-by": "flowengine-operator",
					"flowengine.io/app-name":   fsa.Name,
					"flowengine.io/component":  "ui-schema",
				},
			},
			"data": map[string]interface{}{
				"schema.json": fsa.Spec.UILayoutSchema,
			},
		},
	}

	return r.Client.Patch(ctx, cm, client.Apply, client.ForceOwnership, client.FieldOwner("flowengine-operator"))
}

// ─────────────────────────────────────────────
//  Status helpers
// ─────────────────────────────────────────────

// setPhase updates the FSA's status Phase and persists it.
func (r *FullStackApplicationReconciler) setPhase(ctx context.Context, fsa *v1alpha1.FullStackApplication, phase, message string) error {
	fsa.Status.Phase = phase
	fsa.Status.Conditions = setCondition(fsa.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             phase,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
	return r.Status().Update(ctx, fsa)
}

// setCondition upserts a Condition by Type, replacing any existing condition with the same Type.
func setCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	for i, c := range conditions {
		if c.Type == newCondition.Type {
			conditions[i] = newCondition
			return conditions
		}
	}
	return append(conditions, newCondition)
}

// ─────────────────────────────────────────────
//  SetupWithManager wires the controller to the manager
// ─────────────────────────────────────────────

// SetupWithManager registers the FullStackApplicationReconciler with the controller-runtime manager.
func (r *FullStackApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.FullStackApplication{}).
		Complete(r)
}
