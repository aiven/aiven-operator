// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: Enable `username` validation rule (add a '+' to the XValidation:rule line below).
//
// Currently controller-gen has a bug that prevents the line below from working correctly.
// We use XValidation on connInfoSecretTargetDisabled which is on the same level in the generated CRD yaml as
// username. Kubebuilder has CRD flattening which merges the validation rules into a single allOf array
// which generates invalid CRD yaml that results in a "spec.validation.openAPIV3Schema.properties[spec].allOf[0].x-kubernetes-validations:
// Forbidden: must be empty to be structural" error when trying to install the CRDs.

// kubebuilder:validation:XValidation:rule="!has(oldSelf.username) || has(self.username)",message="Username is required once set"

// ClickhouseUserSpec defines the desired state of ClickhouseUser
type ClickhouseUserSpec struct {
	ServiceDependant `json:",inline"`
	SecretFields     `json:",inline"`

	// ConnInfoSecretSource allows specifying an existing secret to read credentials from.
	// The password from this secret will be used to modify the ClickHouse user credentials.
	// Password must be 8-256 characters long as per Aiven API requirements.
	// This can be used to set passwords for new users or modify passwords for existing users.
	// The source secret is watched for changes, and reconciliation will be automatically triggered
	// when the secret data is updated.
	ConnInfoSecretSource *ConnInfoSecretSource `json:"connInfoSecretSource,omitempty"`

	// Name of the Clickhouse user. Defaults to `metadata.name` if omitted.
	// Note: `metadata.name` is ASCII-only. For UTF-8 names, use `spec.username`, but ASCII is advised for compatibility.
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Username string `json:"username,omitempty"`
}

// ClickhouseUserStatus defines the observed state of ClickhouseUser
type ClickhouseUserStatus struct {
	// Clickhouse user UUID
	UUID string `json:"uuid"`

	// Conditions represent the latest available observations of an ClickhouseUser state
	// +kubebuilder:validation:type=array
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClickhouseUser is the Schema for the clickhouseusers API.
// Info "Exposes secret keys": `CLICKHOUSEUSER_HOST`, `CLICKHOUSEUSER_PORT`, `CLICKHOUSEUSER_USERNAME`, `CLICKHOUSEUSER_PASSWORD`
// +kubebuilder:printcolumn:name="Username",type="string",JSONPath=".spec.username"
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Connection Information Secret",type="string",JSONPath=".spec.connInfoSecretTarget.name"
type ClickhouseUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClickhouseUserSpec   `json:"spec,omitempty"`
	Status ClickhouseUserStatus `json:"status,omitempty"`
}

func (in *ClickhouseUser) GetConnInfoSecretSource() *ConnInfoSecretSource {
	return in.Spec.ConnInfoSecretSource
}

var _ AivenManagedObject = &ClickhouseUser{}

func (in *ClickhouseUser) GetUsername() string {
	// Default to Spec.Username and use ObjectMeta.Name if empty.
	// ObjectMeta.Name doesn't support UTF-8 characters, Spec.Username does.
	name := in.Spec.Username
	if name == "" {
		name = in.ObjectMeta.Name
	}
	return name
}

func (in *ClickhouseUser) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *ClickhouseUser) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *ClickhouseUser) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *ClickhouseUser) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *ClickhouseUser) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

//+kubebuilder:object:root=true

// ClickhouseUserList contains a list of ClickhouseUser
type ClickhouseUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClickhouseUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClickhouseUser{}, &ClickhouseUserList{})
}
