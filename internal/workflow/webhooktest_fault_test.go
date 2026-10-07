package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type webhookFaultClient struct {
	client.Client
	getErrors  map[string]error
	createKind string
	createErr  error
	statusErr  error
	deleteErr  error
}

func (c *webhookFaultClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	if err := c.getErrors[key.Name]; err != nil {
		return err
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

func (c *webhookFaultClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if c.createErr != nil {
		switch c.createKind {
		case "Pod":
			if _, ok := obj.(*corev1.Pod); ok {
				return c.createErr
			}
		case "Service":
			if _, ok := obj.(*corev1.Service); ok {
				return c.createErr
			}
		}
	}
	return c.Client.Create(ctx, obj, opts...)
}

func (c *webhookFaultClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if c.deleteErr != nil {
		return c.deleteErr
	}
	return c.Client.Delete(ctx, obj, opts...)
}

func (c *webhookFaultClient) Status() client.SubResourceWriter {
	return webhookFaultStatusWriter{SubResourceWriter: c.Client.Status(), err: c.statusErr}
}

type webhookFaultStatusWriter struct {
	client.SubResourceWriter
	err error
}

func (w webhookFaultStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if w.err != nil {
		return w.err
	}
	return w.SubResourceWriter.Update(ctx, obj, opts...)
}

func TestWebhookTestReconcilePropagatesClientFailures(t *testing.T) {
	failure := errors.New("Kubernetes client failed")
	tests := []struct {
		name          string
		objects       []client.Object
		getErrorKey   string
		createKind    string
		statusError   error
		deleteError   error
		limitedScheme bool
	}{
		{name: "get custom resource", objects: webhookTestObjects("Running", time.Now()), getErrorKey: "test-wt"},
		{name: "get pod", objects: webhookTestObjects("Running", time.Now()), getErrorKey: "test-wt-webhook-pod"},
		{name: "create pod", objects: webhookTestObjects("Running", time.Now())[:1], createKind: "Pod"},
		{name: "set pod owner reference", objects: webhookTestObjects("Running", time.Now())[:1], limitedScheme: true},
		{name: "get service", objects: webhookTestObjects("Running", time.Now())[:2], getErrorKey: "test-wt-webhook-svc"},
		{name: "create service", objects: webhookTestObjects("Running", time.Now())[:2], createKind: "Service"},
		{name: "set service owner reference", objects: webhookTestObjects("Running", time.Now())[:2], limitedScheme: true},
		{name: "update status", objects: webhookTestObjects("Pending", time.Now()), statusError: failure},
		{name: "delete expired", objects: webhookTestObjects("Running", time.Now().Add(-11*time.Minute)), deleteError: failure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler, request := newFaultWebhookReconciler(t, tt.objects)
			if tt.getErrorKey != "" {
				reconciler.Client.(*webhookFaultClient).getErrors = map[string]error{tt.getErrorKey: failure}
			}
			reconciler.Client.(*webhookFaultClient).createKind = tt.createKind
			if tt.createKind != "" {
				reconciler.Client.(*webhookFaultClient).createErr = failure
			}
			reconciler.Client.(*webhookFaultClient).statusErr = tt.statusError
			reconciler.Client.(*webhookFaultClient).deleteErr = tt.deleteError
			if tt.limitedScheme {
				limited := runtime.NewScheme()
				if err := corev1.AddToScheme(limited); err != nil {
					t.Fatalf("register limited scheme: %v", err)
				}
				reconciler.Scheme = limited
			}
			if _, err := reconciler.Reconcile(context.Background(), request); err == nil {
				t.Fatal("Reconcile() succeeded despite an injected failure")
			} else if !tt.limitedScheme && !errors.Is(err, failure) {
				t.Fatalf("Reconcile() error = %v, want injected client error", err)
			}
		})
	}
}

func TestWebhookTestSetupWithManager(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("register core scheme: %v", err)
	}
	if err := kttmv1.AddToScheme(scheme); err != nil {
		t.Fatalf("register API scheme: %v", err)
	}
	mgr := &setupManager{scheme: scheme}
	reconciler := &WebhookTestReconciler{Scheme: scheme}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		t.Fatalf("SetupWithManager() error = %v", err)
	}
	if mgr.controllers != 1 {
		t.Fatalf("manager received %d controllers, want 1", mgr.controllers)
	}
}

func newFaultWebhookReconciler(t *testing.T, objects []client.Object) (*WebhookTestReconciler, ctrl.Request) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("register core scheme: %v", err)
	}
	if err := kttmv1.AddToScheme(scheme); err != nil {
		t.Fatalf("register API scheme: %v", err)
	}
	base := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kttmv1.WebhookTest{}).WithObjects(objects...).Build()
	client := &webhookFaultClient{Client: base}
	reconciler := &WebhookTestReconciler{Client: client, Scheme: scheme}
	return reconciler, ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-wt", Namespace: "default"}}
}

var _ = metav1.Now
