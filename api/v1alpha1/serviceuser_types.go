// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceUserSpec defines the desired state of ServiceUser
type ServiceUserSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Project to link the user to
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// Service to link the user to
	ServiceName string `json:"serviceName"`

	// +kubebuilder:validation:Enum=caching_sha2_password;mysql_native_password
	// Authentication details
	Authentication string `json:"authentication,omitempty"`
}

// ServiceUserStatus defines the observed state of ServiceUser
type ServiceUserStatus struct {
	ServiceUserSpec `json:",inline"`

	// Type of the user account
	Type string `json:"type,omitempty"`
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
