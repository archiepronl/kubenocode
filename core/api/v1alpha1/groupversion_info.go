// Package v1alpha1 contains the v1alpha1 API group version definitions
// for the FlowEngine custom resources.
// +groupName=flowengine.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is the group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "flowengine.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add functions to this group's scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&FullStackApplication{}, &FullStackApplicationList{})
	metav1.AddToGroupVersion(scheme.Scheme, GroupVersion)
}

// schemeInit is a placeholder to satisfy the go runtime if needed.
var _ = runtime.ObjectCreater(nil)
