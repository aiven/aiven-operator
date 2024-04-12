// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClickhouseDatabaseSpec defines the desired state of ClickhouseDatabase
type ClickhouseDatabaseSpec struct {
	ServiceDependant `json:",inline"`
}

// ClickhouseDatabaseStatus defines the observed state of ClickhouseDatabase
type ClickhouseDatabaseStatus struct {
	// Conditions represent the latest available observations of an ClickhouseDatabase state
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ClickhouseDatabase is the Schema for the databases API
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
type ClickhouseDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClickhouseDatabaseSpec   `json:"spec,omitempty"`
	Status ClickhouseDatabaseStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &ClickhouseDatabase{}

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
