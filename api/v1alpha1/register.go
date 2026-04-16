// +groupName=governor.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	GroupName    = "governor.io"
	GroupVersion = "v1alpha1"
)

// SchemeBuilder registers the types with controller-runtime

var schemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
var (
	SchemeBuilder = &scheme.Builder{GroupVersion: schemeGroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&GovernorRoute{}, &GovernorRouteList{})
}
