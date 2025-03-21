// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConnectionPoolSpec defines the desired state of ConnectionPool
type ConnectionPoolSpec struct {
	ServiceDependant `json:",inline"`
	SecretFields     `json:",inline"`

	// +kubebuilder:validation:MaxLength=40
	// Name of the database the pool connects to
	DatabaseName string `json:"databaseName"`

	// +kubebuilder:validation:MaxLength=64
	// Name of the service user used to connect to the database
	Username string `json:"username,omitempty"`

	// +kubebuilder:validation:Min=1
	// +kubebuilder:validation:Max=1000
	// Number of connections the pool may create towards the backend server
	PoolSize int `json:"poolSize,omitempty"`

	// +kubebuilder:validation:Enum=session;transaction;statement
	// Mode the pool operates in (session, transaction, statement)
	PoolMode string `json:"poolMode,omitempty"`
}

// ConnectionPoolStatus defines the observed state of ConnectionPool
type ConnectionPoolStatus struct {
	// Conditions represent the latest available observations of an ConnectionPool state
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ConnectionPool is the Schema for the connectionpools API.
// Info "Exposes secret keys": `CONNECTIONPOOL_NAME`, `CONNECTIONPOOL_HOST`, `CONNECTIONPOOL_PORT`, `CONNECTIONPOOL_DATABASE`, `CONNECTIONPOOL_USER`, `CONNECTIONPOOL_PASSWORD`, `CONNECTIONPOOL_SSLMODE`, `CONNECTIONPOOL_DATABASE_URI`, `CONNECTIONPOOL_CA_CERT`
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Database",type="string",JSONPath=".spec.databaseName"
// +kubebuilder:printcolumn:name="Username",type="string",JSONPath=".spec.username"
// +kubebuilder:printcolumn:name="Pool Size",type="string",JSONPath=".spec.poolSize"
// +kubebuilder:printcolumn:name="Pool Mode",type="string",JSONPath=".spec.poolMode"
type ConnectionPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConnectionPoolSpec   `json:"spec,omitempty"`
	Status ConnectionPoolStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &ConnectionPool{}

func (in *ConnectionPool) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *ConnectionPool) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *ConnectionPool) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *ConnectionPool) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

// +kubebuilder:object:root=true

// ConnectionPoolList contains a list of ConnectionPool
type ConnectionPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConnectionPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConnectionPool{}, &ConnectionPoolList{})
}
