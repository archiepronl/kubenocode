package v1alpha1

import (
	"testing"
)

func TestWebhookTestDeepCopy(t *testing.T) {
	wt := &WebhookTest{
		Spec: WebhookTestSpec{
			Protocol: "HTTP",
			Methods:  []string{"GET", "POST"},
		},
		Status: WebhookTestStatus{
			Phase: "Running",
		},
	}

	wt2 := wt.DeepCopy()
	if wt2.Spec.Protocol != wt.Spec.Protocol {
		t.Fatalf("expected protocol %s, got %s", wt.Spec.Protocol, wt2.Spec.Protocol)
	}
	if len(wt2.Spec.Methods) != len(wt.Spec.Methods) {
		t.Fatalf("expected methods length %d, got %d", len(wt.Spec.Methods), len(wt2.Spec.Methods))
	}
	if wt2.Status.Phase != wt.Status.Phase {
		t.Fatalf("expected phase %s, got %s", wt.Status.Phase, wt2.Status.Phase)
	}

	obj := wt.DeepCopyObject()
	if obj == nil {
		t.Fatal("DeepCopyObject returned nil")
	}

	wtList := &WebhookTestList{
		Items: []WebhookTest{*wt},
	}
	wtList2 := wtList.DeepCopy()
	if len(wtList2.Items) != 1 {
		t.Fatal("expected 1 item in list")
	}

	objList := wtList.DeepCopyObject()
	if objList == nil {
		t.Fatal("DeepCopyObject returned nil for list")
	}

	var nilWt *WebhookTest
	if nilWt.DeepCopy() != nil {
		t.Fatal("expected nil deepcopy for nil object")
	}

	var nilWtList *WebhookTestList
	if nilWtList.DeepCopy() != nil {
		t.Fatal("expected nil deepcopy for nil list")
	}
}
