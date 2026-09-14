// Package v1alpha1 contains the v1alpha1 API group version definitions
// for the FlowEngine and KttmApp custom resources.
// +groupName=flowengine.io
package v1alpha1

import (
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
	// Register FullStackApplication CRD types.
	SchemeBuilder.Register(&FullStackApplication{}, &FullStackApplicationList{})
	// Register KttmApp CRD types.
	SchemeBuilder.Register(&KttmApp{}, &KttmAppList{})
}
