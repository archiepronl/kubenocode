package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestWebhookTestDeepCopyMethodsPreserveIsolation(t *testing.T) {
	original := &WebhookTest{
		ObjectMeta: metav1.ObjectMeta{Name: "hook", Labels: map[string]string{"team": "data"}},
		Spec:       WebhookTestSpec{Protocol: "HTTP", Methods: []string{"POST", "PUT"}},
		Status:     WebhookTestStatus{Phase: "Running", URL: "http://hook"},
	}

	copy := original.DeepCopy()
	copy.Labels["team"] = "changed"
	copy.Spec.Methods[0] = "DELETE"
	copy.Status.Phase = "Changed"
	if original.Labels["team"] != "data" || original.Spec.Methods[0] != "POST" || original.Status.Phase != "Running" {
		t.Fatal("WebhookTest.DeepCopy() aliases mutable source data")
	}
	var into WebhookTest
	original.DeepCopyInto(&into)
	if into.Name != original.Name || into.Spec.Methods[1] != "PUT" {
		t.Fatalf("WebhookTest.DeepCopyInto() lost fields: %+v", into)
	}
	if _, ok := original.DeepCopyObject().(runtime.Object); !ok {
		t.Fatal("WebhookTest.DeepCopyObject() did not return runtime.Object")
	}
	if ((*WebhookTest)(nil)).DeepCopy() != nil || ((*WebhookTest)(nil)).DeepCopyObject() != nil {
		t.Fatal("nil WebhookTest deep copy should return nil")
	}

	list := &WebhookTestList{Items: []WebhookTest{*original.DeepCopy()}}
	listCopy := list.DeepCopy()
	listCopy.Items[0].Spec.Methods[0] = "DELETE"
	if list.Items[0].Spec.Methods[0] != "POST" {
		t.Fatal("WebhookTestList.DeepCopy() aliases item slices")
	}
	var listInto WebhookTestList
	list.DeepCopyInto(&listInto)
	if len(listInto.Items) != 1 || listInto.Items[0].Name != "hook" {
		t.Fatalf("WebhookTestList.DeepCopyInto() lost items: %+v", listInto)
	}
	if _, ok := list.DeepCopyObject().(runtime.Object); !ok {
		t.Fatal("WebhookTestList.DeepCopyObject() did not return runtime.Object")
	}
	if ((*WebhookTestList)(nil)).DeepCopy() != nil || ((*WebhookTestList)(nil)).DeepCopyObject() != nil {
		t.Fatal("nil WebhookTestList deep copy should return nil")
	}

	var specInto WebhookTestSpec
	(&WebhookTestSpec{Methods: []string{"GET"}}).DeepCopyInto(&specInto)
	if specInto.DeepCopy().Methods[0] != "GET" || ((*WebhookTestSpec)(nil)).DeepCopy() != nil {
		t.Fatal("WebhookTestSpec deep-copy methods failed")
	}
	var statusInto WebhookTestStatus
	(&WebhookTestStatus{Phase: "Ready"}).DeepCopyInto(&statusInto)
	if statusInto.DeepCopy().Phase != "Ready" || ((*WebhookTestStatus)(nil)).DeepCopy() != nil {
		t.Fatal("WebhookTestStatus deep-copy methods failed")
	}
}
