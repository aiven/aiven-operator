// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClickhouseDatabaseSpec defines the desired state of ClickhouseDatabase
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.databaseName) || has(self.databaseName)", message="databaseName is required once set"
type ClickhouseDatabaseSpec struct {
	ServiceDependant `json:",inline"`

	// Specifies the Clickhouse database name. Defaults to `metadata.name` if omitted.
	// Note: `metadata.name` is ASCII-only. For UTF-8 names, use `spec.databaseName`, but ASCII is advised for compatibility.
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	DatabaseName string `json:"databaseName,omitempty"`
}

// ClickhouseDatabaseStatus defines the observed state of ClickhouseDatabase
type ClickhouseDatabaseStatus struct {
	// Conditions represent the latest available observations of an ClickhouseDatabase state
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ClickhouseDatabase is the Schema for the databases API
// +kubebuilder:printcolumn:name="Database name",type="string",JSONPath=".spec.databaseName"
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
type ClickhouseDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClickhouseDatabaseSpec   `json:"spec,omitempty"`
	Status ClickhouseDatabaseStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &ClickhouseDatabase{}

func (in *ClickhouseDatabase) GetDatabaseName() string {
	// Default to Spec.DatabaseName and use ObjectMeta.Name if empty.
	// ObjectMeta.Name doesn't support UTF-8 characters, Spec.DatabaseName does.
	name := in.Spec.DatabaseName
	if name == "" {
		name = in.ObjectMeta.Name
	}
	return name
}

func (*ClickhouseDatabase) NoSecret() bool {
	return true
}

func (in *ClickhouseDatabase) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *ClickhouseDatabase) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// +kubebuilder:object:root=true

// ClickhouseDatabaseList contains a list of ClickhouseDatabase
type ClickhouseDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClickhouseDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClickhouseDatabase{}, &ClickhouseDatabaseList{})
}
