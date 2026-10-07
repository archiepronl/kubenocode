package bff

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
)

var k8sClient client.Client

// InitK8sClient initializes the kubernetes client for the BFF
func InitK8sClient() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := clientcmd.RecommendedHomeFile
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}

	s := scheme.Scheme
	if err := kttmv1.AddToScheme(s); err != nil {
		return fmt.Errorf("failed to add kttmv1 to scheme: %w", err)
	}

	cl, err := client.New(config, client.Options{Scheme: s})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	k8sClient = cl
	return nil
}

// HandleCreateWebhookTest handles POST /api/test/webhook
func HandleCreateWebhookTest(w http.ResponseWriter, r *http.Request) {
	if k8sClient == nil {
		http.Error(w, "Kubernetes client not initialized", http.StatusInternalServerError)
		return
	}

	var reqData struct {
		Server struct {
			Protocol string `json:"protocol"`
			Port     int32  `json:"port"`
			Host     string `json:"host"`
			Exposure string `json:"exposure"`
		} `json:"server"`
		Endpoint struct {
			Path    string   `json:"path"`
			Methods []string `json:"methods"`
		} `json:"endpoint"`
		Contract struct {
			ValidatePayload  bool   `json:"validatePayload"`
			SchemaDefinition string `json:"schemaDefinition"`
		} `json:"contract"`
		Security struct {
			Authentication string `json:"authentication"`
			SecretRef      string `json:"secretRef"`
		} `json:"security"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	testID := fmt.Sprintf("test-webhook-%04x", rand.Intn(0x10000))

	wt := &kttmv1.WebhookTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testID,
			Namespace: "default",
		},
		Spec: kttmv1.WebhookTestSpec{
			Protocol:  reqData.Server.Protocol,
			Port:      reqData.Server.Port,
			Host:      reqData.Server.Host,
			Exposure:  reqData.Server.Exposure,
			Path:      reqData.Endpoint.Path,
			Methods:   reqData.Endpoint.Methods,
			Validate:  reqData.Contract.ValidatePayload,
			SchemaDef: reqData.Contract.SchemaDefinition,
			AuthType:  reqData.Security.Authentication,
			SecretRef: reqData.Security.SecretRef,
		},
	}

	if err := k8sClient.Create(context.Background(), wt); err != nil {
		http.Error(w, fmt.Sprintf("failed to create WebhookTest: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": testID, "status": "pending"})
}

// HandleGetWebhookTest handles GET /api/test/webhook/{id}
func HandleGetWebhookTest(w http.ResponseWriter, r *http.Request) {
	if k8sClient == nil {
		http.Error(w, "Kubernetes client not initialized", http.StatusInternalServerError)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing test id", http.StatusBadRequest)
		return
	}

	var wt kttmv1.WebhookTest
	err := k8sClient.Get(context.Background(), client.ObjectKey{Name: id, Namespace: "default"}, &wt)
	if err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, "test not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to get WebhookTest: %v", err), http.StatusInternalServerError)
		return
	}

	phase := wt.Status.Phase
	if phase == "" {
		phase = "Pending"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":    wt.Name,
		"phase": phase,
		"url":   wt.Status.URL,
	})
}
