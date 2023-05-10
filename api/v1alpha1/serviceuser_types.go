// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

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

	// Information regarding secret creation.
	//
	// Exposed keys: `SERVICEUSER_HOST`, `SERVICEUSER_PORT`, `SERVICEUSER_USERNAME`, `SERVICEUSER_PASSWORD`, `SERVICEUSER_CA_CERT`, `SERVICEUSER_ACCESS_CERT`, `SERVICEUSER_ACCESS_KEY`
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`
}

// ServiceUserStatus defines the observed state of ServiceUser
type ServiceUserStatus struct {
	// Conditions represent the latest available observations of an ServiceUser state
	Conditions []metav1.Condition `json:"conditions"`

	// Type of the user account
	Type string `json:"type,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceUser is the Schema for the serviceusers API
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Connection Information Secret",type="string",JSONPath=".spec.connInfoSecretTarget.name"
type ServiceUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceUserSpec   `json:"spec,omitempty"`
	Status ServiceUserStatus `json:"status,omitempty"`
}

func (svcusr ServiceUser) AuthSecretRef() *AuthSecretReference {
	return svcusr.Spec.AuthSecretRef
}

func (in *ServiceUser) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
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
