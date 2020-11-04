// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceUserSpec defines the desired state of ServiceUser
type ServiceUserSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// x-kubernetes-immutable: true
	// Project name
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// x-kubernetes-immutable: true
	// Service name
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=63
	// x-kubernetes-immutable: true
	// Name of the user account
	Username string `json:"username"`
}

// ServiceUserStatus defines the observed state of ServiceUser
type ServiceUserStatus struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Project name
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// Service name
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=63
	// Name of the user account
	Username string `json:"username"`

	// Type of the user account
	Type string `json:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceUser is the Schema for the serviceusers API
type ServiceUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceUserSpec   `json:"spec,omitempty"`
	Status ServiceUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceUserList contains a list of ServiceUser
type ServiceUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceUser{}, &ServiceUserList{})
}
