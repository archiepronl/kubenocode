package controller
import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	v1alpha1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	"github.com/kubeworkflow/flowengine/core/engine"
	"github.com/kubeworkflow/flowengine/core/linter"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type patchCaptureClient struct {
	client.Client
	app          *v1alpha1.FullStackApplication
	key          types.NamespacedName
	patches      []client.Object
	patchErr     error
	getErr       error
	statusErrors []error
	statusCalls  int
	patchErrors  map[string]error
}

func (c *patchCaptureClient) Get(_ context.Context, key types.NamespacedName, obj client.Object, _ ...client.GetOption) error {
	if c.getErr != nil {
		return c.getErr
	}
	if c.app == nil || key != c.key {
		return apierrors.NewNotFound(schema.GroupResource{Group: v1alpha1.GroupVersion.Group, Resource: "fullstackapplications"}, key.Name)
	}
	app, ok := obj.(*v1alpha1.FullStackApplication)
	if !ok {
		return fmt.Errorf("unexpected Get object %T", obj)
	}
	c.app.DeepCopyInto(app)
	return nil
}

func (c *patchCaptureClient) Patch(_ context.Context, obj client.Object, _ client.Patch, _ ...client.PatchOption) error {
	copy := obj.DeepCopyObject().(client.Object)
	c.patches = append(c.patches, copy)
	if unstructuredObject, ok := copy.(*unstructured.Unstructured); ok && c.patchErrors != nil {
		if err := c.patchErrors[unstructuredObject.GetKind()]; err != nil {
			return err
		}
	}
	return c.patchErr
}

func (c *patchCaptureClient) Status() client.SubResourceWriter {
	return &statusCaptureWriter{parent: c}
}

type statusCaptureWriter struct {
	client.SubResourceWriter
	parent *patchCaptureClient
}

func (w *statusCaptureWriter) Update(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
	w.parent.statusCalls++
	if index := w.parent.statusCalls - 1; index < len(w.parent.statusErrors) && w.parent.statusErrors[index] != nil {
		return w.parent.statusErrors[index]
	}
	app, ok := obj.(*v1alpha1.FullStackApplication)
	if !ok {
		return fmt.Errorf("unexpected status object %T", obj)
	}
	if w.parent.app == nil {
		return apierrors.NewNotFound(schema.GroupResource{Group: v1alpha1.GroupVersion.Group, Resource: "fullstackapplications"}, app.Name)
	}
	app.Status.DeepCopyInto(&w.parent.app.Status)
	return nil
}

type failingCompiler struct{ err error }

func (c failingCompiler) Name() string { return "failing" }

func (c failingCompiler) Compile(*v1alpha1.FullStackApplication) (*engine.CompiledManifest, error) {
	return nil, c.err
}

func TestReconcileAppliesDesiredStateOncePerGeneration(t *testing.T) {
	reconciler, capture, key := newReconciler(t, validApplication())
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})
	if err != nil || result != (ctrl.Result{}) {
		t.Fatalf("Reconcile() = (%+v, %v), want success", result, err)
	}
	updated := &v1alpha1.FullStackApplication{}
	if err := reconciler.Get(context.Background(), key, updated); err != nil {
		t.Fatalf("fetch reconciled app: %v", err)
	}
	if updated.Status.Phase != PhaseRunning || updated.Status.ObservedGeneration != updated.Generation ||
		updated.Status.NodeCount != 1 || updated.Status.CompiledWorkflowRef != "fsa-orders" || len(updated.Status.Conditions) != 1 {
		t.Fatalf("unexpected final status: %+v", updated.Status)
	}
	if len(capture.patches) != 2 {
		t.Fatalf("patch count = %d, want workflow and UI configmap", len(capture.patches))
	}
	workflow, ok := capture.patches[0].(*unstructured.Unstructured)
	if !ok || workflow.GetKind() != "Workflow" || workflow.GetNamespace() != "team-a" {
		t.Fatalf("unexpected workflow apply object: %#v", capture.patches[0])
	}
	configMap, ok := capture.patches[1].(*unstructured.Unstructured)
	if !ok || configMap.GetKind() != "ConfigMap" || configMap.GetName() != "fsa-ui-orders" {
		t.Fatalf("unexpected UI schema apply object: %#v", capture.patches[1])
	}

	result, err = reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})
	if err != nil || result != (ctrl.Result{}) || len(capture.patches) != 2 {
		t.Fatalf("unchanged generation was not idempotent: result=%+v err=%v patches=%d", result, err, len(capture.patches))
	}
}

func TestReconcileNotFoundAndLintFailure(t *testing.T) {
	reconciler, _, key := newReconciler(t)
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})
	if err != nil || result != (ctrl.Result{}) {
		t.Fatalf("Reconcile(missing) = (%+v, %v), want clean no-op", result, err)
	}

	app := validApplication()
	app.Spec.WorkflowDAG[0].Type = "script/python"
	app.Spec.WorkflowDAG[0].Script = ""
	reconciler, capture, key := newReconciler(t, app)
	result, err = reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})
	if err != nil || result.RequeueAfter != 30*time.Second {
		t.Fatalf("Reconcile(invalid DAG) = (%+v, %v), want 30s requeue", result, err)
	}
	updated := &v1alpha1.FullStackApplication{}
	if err := reconciler.Get(context.Background(), key, updated); err != nil {
		t.Fatalf("fetch lint-failed app: %v", err)
	}
	if updated.Status.Phase != PhaseFailed || len(updated.Status.LintErrors) == 0 || len(capture.patches) != 0 {
		t.Fatalf("lint failure was not persisted or deployed: status=%+v patches=%d", updated.Status, len(capture.patches))
	}
}

func TestReconcileFetchAndStatusFailures(t *testing.T) {
	failure := errors.New("status update failed")
	t.Run("fetch failure", func(t *testing.T) {
		reconciler, capture, key := newReconciler(t, validApplication())
		capture.getErr = errors.New("api unavailable")
		if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); err == nil || !strings.Contains(err.Error(), "fetching FullStackApplication") {
			t.Fatalf("Reconcile() error = %v, want fetch error", err)
		}
	})
	t.Run("compiling phase failure", func(t *testing.T) {
		reconciler, capture, key := newReconciler(t, validApplication())
		capture.statusErrors = []error{failure}
		if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); !errors.Is(err, failure) {
			t.Fatalf("Reconcile() error = %v, want status error", err)
		}
	})
	t.Run("lint status failure", func(t *testing.T) {
		app := validApplication()
		app.Spec.WorkflowDAG[0].Type = "script/python"
		reconciler, capture, key := newReconciler(t, app)
		capture.statusErrors = []error{nil, failure}
		if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); !errors.Is(err, failure) {
			t.Fatalf("Reconcile() error = %v, want lint status error", err)
		}
	})
	t.Run("deploying phase failure", func(t *testing.T) {
		reconciler, capture, key := newReconciler(t, validApplication())
		capture.statusErrors = []error{nil, nil, failure}
		if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); !errors.Is(err, failure) {
			t.Fatalf("Reconcile() error = %v, want deploying phase error", err)
		}
	})
	t.Run("failed phase status error after compiler failure", func(t *testing.T) {
		reconciler, capture, key := newReconciler(t, validApplication())
		compileErr := errors.New("compile failed")
		reconciler.Compiler = failingCompiler{err: compileErr}
		capture.statusErrors = []error{nil, nil, failure}
		if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); !errors.Is(err, compileErr) {
			t.Fatalf("Reconcile() error = %v, want original compile error", err)
		}
	})
	t.Run("final status failure", func(t *testing.T) {
		reconciler, capture, key := newReconciler(t, validApplication())
		capture.statusErrors = []error{nil, nil, nil, failure}
		if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); !errors.Is(err, failure) {
			t.Fatalf("Reconcile() error = %v, want final status error", err)
		}
	})
}

func TestReconcileCompilationAndApplyFailures(t *testing.T) {
	compileFailure := errors.New("compile failed")
	reconciler, _, key := newReconciler(t, validApplication())
	reconciler.Compiler = failingCompiler{err: compileFailure}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); !errors.Is(err, compileFailure) {
		t.Fatalf("Reconcile(compile failure) error = %v", err)
	}

	reconciler, capture, key := newReconciler(t, validApplication())
	capture.patchErr = errors.New("apply failed")
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); err == nil {
		t.Fatal("Reconcile() succeeded after apply failure")
	}
	updated := &v1alpha1.FullStackApplication{}
	if err := reconciler.Get(context.Background(), key, updated); err != nil {
		t.Fatalf("fetch apply-failed app: %v", err)
	}
	if updated.Status.Phase != PhaseFailed {
		t.Fatalf("apply failure phase = %q, want Failed", updated.Status.Phase)
	}
}

func TestReconcileContinuesWhenUIConfigMapApplyFails(t *testing.T) {
	reconciler, capture, key := newReconciler(t, validApplication())
	capture.patchErrors = map[string]error{"ConfigMap": errors.New("configmap apply failed")}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile() returned non-fatal UI apply error: %v", err)
	}
	updated := &v1alpha1.FullStackApplication{}
	if err := reconciler.Get(context.Background(), key, updated); err != nil {
		t.Fatalf("fetch app after UI apply failure: %v", err)
	}
	if updated.Status.Phase != PhaseRunning || updated.Status.ObservedGeneration != updated.Generation {
		t.Fatalf("UI apply failure blocked workflow status: %+v", updated.Status)
	}
}

func TestApplyHelpersAndPhaseConditionUpdates(t *testing.T) {
	app := validApplication()
	reconciler, capture, _ := newReconciler(t, app)
	compiled := &engine.CompiledManifest{Raw: []byte(`{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","metadata":{"name":"workflow"}}`)}
	if err := reconciler.applyArgoWorkflow(context.Background(), app, compiled); err != nil {
		t.Fatalf("applyArgoWorkflow() error = %v", err)
	}
	if err := reconciler.applyArgoWorkflow(context.Background(), app, &engine.CompiledManifest{Raw: []byte("{" )}); err == nil {
		t.Fatal("applyArgoWorkflow() accepted invalid JSON")
	}
	app.Spec.UILayoutSchema = `{"type":"object"}`
	if err := reconciler.applyUIConfigMap(context.Background(), app); err != nil {
		t.Fatalf("applyUIConfigMap() error = %v", err)
	}
	if err := reconciler.setPhase(context.Background(), app, PhaseCompiling, "compile"); err != nil {
		t.Fatalf("setPhase() error = %v", err)
	}
	if app.Status.Phase != PhaseCompiling || len(app.Status.Conditions) != 1 || len(capture.patches) != 2 {
		t.Fatalf("unexpected helper results: status=%+v patches=%d", app.Status, len(capture.patches))
	}
	updated := setCondition(app.Status.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Ready"})
	if len(updated) != 1 || updated[0].Status != metav1.ConditionTrue {
		t.Fatalf("setCondition() did not replace existing condition: %+v", updated)
	}
	appended := setCondition(nil, metav1.Condition{Type: "Ready"})
	if len(appended) != 1 {
		t.Fatalf("setCondition() did not append new condition: %+v", appended)
	}
}

func newReconciler(t *testing.T, apps ...*v1alpha1.FullStackApplication) (*FullStackApplicationReconciler, *patchCaptureClient, types.NamespacedName) {
	t.Helper()
	key := types.NamespacedName{Namespace: "team-a", Name: "orders"}
	var app *v1alpha1.FullStackApplication
	if len(apps) > 0 {
		app = apps[0].DeepCopy()
	}
	capture := &patchCaptureClient{app: app, key: key}
	reconciler := &FullStackApplicationReconciler{
		Client: capture,
		Linter: linter.New(),
		Compiler: engine.NewArgoCompiler(),
	}
	return reconciler, capture, key
}

func validApplication() *v1alpha1.FullStackApplication {
	return &v1alpha1.FullStackApplication{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "team-a", Generation: 1},
		Spec: v1alpha1.FullStackApplicationSpec{
			UILayoutSchema: `{"type":"object"}`,
			WorkflowDAG:    []v1alpha1.WorkflowNode{{ID: "source", Type: "trigger/webhook"}},
		},
	}
}

var _ client.Client = (*patchCaptureClient)(nil)
var _ client.Object = (*unstructured.Unstructured)(nil)