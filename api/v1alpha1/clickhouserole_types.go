// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClickhouseRoleSpec defines the desired state of ClickhouseRole
type ClickhouseRoleSpec struct {
	ProjectServiceFields `json:",inline"`

	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Format="^[a-zA-Z_][0-9a-zA-Z_]*$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// The role that is to be created
	Role string `json:"role"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`
}

// ClickhouseRoleStatus defines the observed state of ClickhouseRole
type ClickhouseRoleStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClickhouseRole is the Schema for the clickhouseroles API
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Role",type="string",JSONPath=".spec.role"
type ClickhouseRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClickhouseRoleSpec   `json:"spec,omitempty"`
	Status ClickhouseRoleStatus `json:"status,omitempty"`
}

func (in *ClickhouseRole) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *ClickhouseRole) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *ClickhouseRole) NoSecret() bool {
	return true
}

var _ AivenManagedObject = &ClickhouseRole{}

//+kubebuilder:object:root=true

// ClickhouseRoleList contains a list of ClickhouseRole
type ClickhouseRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClickhouseRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClickhouseRole{}, &ClickhouseRoleList{})
}
