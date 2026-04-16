// Package v1alpha1 contains API Schema definitions for the Governor Operator
// +kubebuilder:object:generate=true
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GovernorRouteSpec Mirrors Existing models\service.go RegisterServiceV2 struct for now but with alignment to Kubernetes Operator Standards hence the old Object can't be reused
type GovernorRouteSpec struct {
	ServiceName    string            `json:"serviceName"`
	TeamName       string            `json:"teamName,omitempty"`
	Namespace      string            `json:"namespace"`
	ContactEmail   string            `json:"contactEmail,omitempty"`
	ConfigEndpoint string            `json:"configEndpoint"`
	WebhookURL     string            `json:"webhookUrl,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type GovernorRouteStatus struct {
	Conditions   []metav1.Condition `json:"conditions,omitempty"`
	LastSyncedAt *metav1.Time       `json:"lastSyncedAt,omitempty"`
	Message      string             `json:"message,omitempty"`
}

// GovernorRoute is the CR teams apply
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Service",type=string,JSONPath=`.spec.serviceName`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.namespace`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type GovernorRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GovernorRouteSpec   `json:"spec,omitempty"`
	Status GovernorRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type GovernorRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GovernorRoute `json:"items"`
}
