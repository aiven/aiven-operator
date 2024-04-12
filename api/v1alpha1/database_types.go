// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseSpec defines the desired state of Database
type DatabaseSpec struct {
	ServiceDependant `json:",inline"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:default=en_US.UTF-8
	// Default string sort order (LC_COLLATE) of the database. Default value: en_US.UTF-8
	LcCollate string `json:"lcCollate,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:default=en_US.UTF-8
	// Default character classification (LC_CTYPE) of the database. Default value: en_US.UTF-8
	LcCtype string `json:"lcCtype,omitempty"`

	// It is a Kubernetes side deletion protections, which prevents the database
	// from being deleted by Kubernetes. It is recommended to enable this for any production
	// databases containing critical data.
	TerminationProtection *bool `json:"terminationProtection,omitempty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// Conditions represent the latest available observations of an Database state
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Database is the Schema for the databases API
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &Database{}

func (*Database) NoSecret() bool {
	return true
}

func (in *Database) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *Database) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// +kubebuilder:object:root=true

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}
