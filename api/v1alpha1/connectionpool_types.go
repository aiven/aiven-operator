// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConnectionPoolSpec defines the desired state of ConnectionPool
type ConnectionPoolSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Target project.
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// Service name.
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=40
	// Name of the database the pool connects to
	DatabaseName string `json:"database_name"`

	// +kubebuilder:validation:MaxLength=60
	// Name of the pool
	PoolName string `json:"pool_name"`

	// +kubebuilder:validation:MaxLength=64
	// Name of the service user used to connect to the database
	Username string `json:"username"`

	// +kubebuilder:validation:Min=1
	// +kubebuilder:validation:Max=1000
	// Number of connections the pool may create towards the backend server
	PoolSize int `json:"pool_size,omitempty"`

	// +kubebuilder:validation:Enum=session;transaction;statement
	// Mode the pool operates in (session, transaction, statement)
	PoolMode string `json:"pool_mode,omitempty"`
}

// ConnectionPoolStatus defines the observed state of ConnectionPool
type ConnectionPoolStatus struct {
	ConnectionPoolSpec `json:",inline"`

	// URI for connecting to the pool
	ConnectionURI string `json:"connection_uri,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ConnectionPool is the Schema for the connectionpools API
type ConnectionPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConnectionPoolSpec   `json:"spec,omitempty"`
	Status ConnectionPoolStatus `json:"status,omitempty"`
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
