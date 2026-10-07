package controller

import (
	"context"
	"testing"
	"time"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestWebhookTestReconcileStableStatusIsIdempotent(t *testing.T) {
	reconciler, apiClient, request := webhookTestReconciler(t, time.Now())
	result, err := reconciler.Reconcile(context.Background(), request)
	if err != nil || result.RequeueAfter != time.Minute {
		t.Fatalf("Reconcile() = (%+v, %v), want one-minute requeue", result, err)
	}
	var current kttmv1.WebhookTest
	if err := apiClient.Get(context.Background(), request.NamespacedName, &current); err != nil {
		t.Fatalf("fetch stable WebhookTest: %v", err)
	}
	if current.Status.Phase != "Running" || current.Status.URL != webhookTestURL() {
		t.Fatalf("stable status changed: %+v", current.Status)
	}
}

func TestWebhookTestReconcileDeletesExpiredResource(t *testing.T) {
	reconciler, apiClient, request := webhookTestReconciler(t, time.Now().Add(-11*time.Minute))
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("Reconcile(expired) error = %v", err)
	}
	var current kttmv1.WebhookTest
	if err := apiClient.Get(context.Background(), request.NamespacedName, &current); !apierrors.IsNotFound(err) {
		t.Fatalf("expired WebhookTest still exists, get error = %v", err)
	}
}

func webhookTestReconciler(t *testing.T, createdAt time.Time) (*WebhookTestReconciler, client.Client, ctrl.Request) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("register core scheme: %v", err)
	}
	if err := kttmv1.AddToScheme(scheme); err != nil {
		t.Fatalf("register API scheme: %v", err)
	}
	objects := webhookTestObjects("Running", createdAt)
	apiClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kttmv1.WebhookTest{}).WithObjects(objects...).Build()
	reconciler := &WebhookTestReconciler{Client: apiClient, Scheme: scheme}
	return reconciler, apiClient, ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-wt", Namespace: "default"}}
}

func webhookTestObjects(phase string, createdAt time.Time) []client.Object {
	return []client.Object{
		&kttmv1.WebhookTest{
			ObjectMeta: metav1.ObjectMeta{Name: "test-wt", Namespace: "default", CreationTimestamp: metav1.NewTime(createdAt)},
			Spec:       kttmv1.WebhookTestSpec{Port: 8080},
			Status:     kttmv1.WebhookTestStatus{Phase: phase, URL: webhookTestURL()},
		},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-wt-webhook-pod", Namespace: "default"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-wt-webhook-svc", Namespace: "default"}},
	}
}

func webhookTestURL() string {
	return "http://test-wt-webhook-svc.default.svc.cluster.local:8080"
}
