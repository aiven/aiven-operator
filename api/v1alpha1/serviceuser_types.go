// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"github.com/aiven/go-client-codegen/handler/service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceUserSpec defines the desired state of ServiceUser
type ServiceUserSpec struct {
	ServiceDependant `json:",inline"`
	SecretFields     `json:",inline"`

	// ConnInfoSecretSource allows specifying an existing secret to read credentials from.
	// The password from this secret will be used to modify the service user credentials.
	// Password must be 8-256 characters long as per Aiven API requirements.
	// This can be used to set passwords for new users or modify passwords for existing users (e.g., avnadmin).
	// The source secret is watched for changes, and reconciliation will be automatically triggered
	// when the secret data is updated.
	ConnInfoSecretSource *ConnInfoSecretSource `json:"connInfoSecretSource,omitempty"`

	// +kubebuilder:validation:Enum=caching_sha2_password;mysql_native_password
	// Authentication details
	Authentication service.AuthenticationType `json:"authentication,omitempty"`
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

// ServiceUser is the Schema for the serviceusers API.
// Creates a service user for accessing Aiven services. The ServiceUser resource name becomes the username in Aiven.
// Built-in users like 'avnadmin' cannot be deleted but their passwords can be modified using connInfoSecretSource.
// Info "Exposes secret keys": `SERVICEUSER_HOST`, `SERVICEUSER_PORT`, `SERVICEUSER_USERNAME`, `SERVICEUSER_PASSWORD`, `SERVICEUSER_CA_CERT`, `SERVICEUSER_ACCESS_CERT`, `SERVICEUSER_ACCESS_KEY`
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Connection Information Secret",type="string",JSONPath=".spec.connInfoSecretTarget.name"
type ServiceUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceUserSpec   `json:"spec,omitempty"`
	Status ServiceUserStatus `json:"status,omitempty"`
}

func (in *ServiceUser) GetConnInfoSecretSource() *ConnInfoSecretSource {
	return in.Spec.ConnInfoSecretSource
}

var _ AivenManagedObject = &ServiceUser{}

func (in *ServiceUser) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *ServiceUser) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *ServiceUser) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *ServiceUser) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
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
