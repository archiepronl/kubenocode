package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WebhookTestSpec defines the desired state of WebhookTest
type WebhookTestSpec struct {
	Protocol  string   `json:"protocol,omitempty"` // HTTP, HTTPS, WebSocket, gRPC
	Port      int32    `json:"port,omitempty"`
	Host      string   `json:"host,omitempty"`
	Exposure  string   `json:"exposure,omitempty"`
	Path      string   `json:"path,omitempty"`
	Methods   []string `json:"methods,omitempty"`
	Validate  bool     `json:"validatePayload,omitempty"`
	SchemaDef string   `json:"schemaDefinition,omitempty"`
	AuthType  string   `json:"authentication,omitempty"`
	SecretRef string   `json:"secretRef,omitempty"`
}

// WebhookTestStatus defines the observed state of WebhookTest
type WebhookTestStatus struct {
	Phase   string `json:"phase,omitempty"`   // Pending, Running, Failed
	URL     string `json:"url,omitempty"`     // The public URL to test against
	Message string `json:"message,omitempty"` // Any error messages
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`

// WebhookTest is the Schema for the webhooktests API
type WebhookTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebhookTestSpec   `json:"spec,omitempty"`
	Status WebhookTestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WebhookTestList contains a list of WebhookTest
type WebhookTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebhookTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WebhookTest{}, &WebhookTestList{})
}
