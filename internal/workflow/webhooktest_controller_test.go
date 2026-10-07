package controller

import (
	"context"
	"testing"

	"fmt"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
)

func TestWebhookTestReconciler(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kttmv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	wt := &kttmv1.WebhookTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wt",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: kttmv1.WebhookTestSpec{
			Port: 8080,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kttmv1.WebhookTest{}).WithObjects(wt).Build()
	reconciler := &WebhookTestReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-wt",
			Namespace: "default",
		},
	}

	// 1st reconcile: creates pod
	_, err := reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// 2nd reconcile: creates service
	_, err = reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// 3rd reconcile: updates status
	_, err = reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	var updatedWt kttmv1.WebhookTest
	if err := client.Get(context.Background(), req.NamespacedName, &updatedWt); err != nil {
		t.Fatalf("failed to get updated wt: %v", err)
	}
	if updatedWt.Status.Phase != "Running" {
		t.Fatalf("expected phase Running, got %s", updatedWt.Status.Phase)
	}

	// test missing
	missingReq := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "missing",
			Namespace: "default",
		},
	}
	_, err = reconciler.Reconcile(context.Background(), missingReq)
	if err != nil {
		t.Fatalf("expected no error for missing wt, got %v", err)
	}
}

func TestWebhookTestReconcilerExternal(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kttmv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)

	wt := &kttmv1.WebhookTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wt-ext",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: kttmv1.WebhookTestSpec{
			Port:     8080,
			Exposure: "External (Public)",
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kttmv1.WebhookTest{}).WithObjects(wt).Build()
	reconciler := &WebhookTestReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-wt-ext",
			Namespace: "default",
		},
	}

	// 1st reconcile: creates pod
	_, err := reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// 2nd reconcile: creates service
	_, err = reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// 3rd reconcile: creates ingress
	_, err = reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// 4th reconcile: updates status
	_, err = reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	var updatedWt kttmv1.WebhookTest
	if err := client.Get(context.Background(), req.NamespacedName, &updatedWt); err != nil {
		t.Fatalf("failed to get updated wt: %v", err)
	}

	if updatedWt.Status.Phase != "Running" {
		t.Fatalf("expected phase Running, got %s", updatedWt.Status.Phase)
	}
}

func TestWebhookTestReconcilerExternal_Errors(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kttmv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)

	wt := &kttmv1.WebhookTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wt-err",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: kttmv1.WebhookTestSpec{
			Port:     8080,
			Exposure: "External (Public)",
		},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kttmv1.WebhookTest{}).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*networkingv1.Ingress); ok {
					return fmt.Errorf("injected create error")
				}
				return c.Create(ctx, obj, opts...)
			},
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if key.Name == "test-wt-err2-webhook-ing" {
					if _, ok := obj.(*networkingv1.Ingress); ok {
						return fmt.Errorf("injected get error")
					}
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).WithObjects(wt).Build()

	reconciler := &WebhookTestReconciler{
		Client: cl,
		Scheme: scheme, // Valid scheme
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-wt-err",
			Namespace: "default",
		},
	}

	// 1st reconcile: creates pod
	_, _ = reconciler.Reconcile(context.Background(), req)
	// 2nd reconcile: creates service
	_, _ = reconciler.Reconcile(context.Background(), req)
	// 3rd reconcile: fails to create ingress due to interceptor
	_, err := reconciler.Reconcile(context.Background(), req)
	if err == nil || err.Error() != "injected create error" {
		t.Fatalf("expected injected create error, got %v", err)
	}

	// Test Get error branch
	wt2 := &kttmv1.WebhookTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-wt-err2",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: kttmv1.WebhookTestSpec{
			Port:     8080,
			Exposure: "External (Public)",
		},
	}
	_ = cl.Create(context.Background(), wt2)

	req2 := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-wt-err2",
			Namespace: "default",
		},
	}
	_, _ = reconciler.Reconcile(context.Background(), req2)   // pod
	_, _ = reconciler.Reconcile(context.Background(), req2)   // svc
	_, err = reconciler.Reconcile(context.Background(), req2) // get ingress error
	if err == nil || err.Error() != "injected get error" {
		t.Fatalf("expected injected get error, got %v", err)
	}

}
