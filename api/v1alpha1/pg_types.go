// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PGSpec defines the desired state of PG
type PGSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// x-kubernetes-immutable: true
	// Target project.
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// x-kubernetes-immutable: true
	// Service name.
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=128
	// Subscription plan.
	Plan string `json:"plan,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Cloud the service runs in.
	CloudName string `json:"cloud_name,omitempty"`

	// +kubebuilder:validation:MaxLength=36
	// Identifier of the VPC the service should be in, if any.
	ProjectVPCID string `json:"project_vpc_id,omitempty"`

	// +kubebuilder:validation:Enum=monday;tuesday;wednesday;thursday;friday;saturday;sunday;never
	// Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
	MaintenanceWindowDow string `json:"maintenance_window_dow,omitempty"`

	// +kubebuilder:validation:MaxLength=8
	// Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
	MaintenanceWindowTime string `json:"maintenance_window_time,omitempty"`
}

// PGStatus defines the observed state of PG
type PGStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Target project.
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// Service name.
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=128
	// Subscription plan.
	Plan string `json:"plan,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Cloud the service runs in.
	CloudName string `json:"cloud_name,omitempty"`

	// +kubebuilder:validation:MaxLength=36
	// Identifier of the VPC the service should be in, if any.
	ProjectVPCID string `json:"project_vpc_id,omitempty"`

	// +kubebuilder:validation:Enum=monday;tuesday;wednesday;thursday;friday;saturday;sunday;never
	// Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
	MaintenanceWindowDow string `json:"maintenance_window_dow,omitempty"`

	// +kubebuilder:validation:MaxLength=8
	// Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
	MaintenanceWindowTime string `json:"maintenance_window_time,omitempty"`

	// PostgreSQL Service state
	State string `json:"state,omitempty"`

	// URI for connecting to the PostgreSQL service.
	ServiceURI string `json:"service_uri,omitempty"`

	// PostgreSQL hostname
	ServiceHost string `json:"service_host,omitempty"`

	// Username used for connecting to the PostgreSQL service.
	ServiceUsername string `json:"service_username,omitempty"`

	// Password used for connecting to the PostgreSQL service.
	ServicePassword string `json:"service_password,omitempty"`

	// PostgreSQL service port.
	ServicePort int `json:"service_port,omitempty"`

	// Service status
	Status string `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// +kubebuilder:subresource:status
// PG is the Schema for the pgs API
type PG struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PGSpec   `json:"spec,omitempty"`
	Status PGStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PGList contains a list of PG
type PGList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PG `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PG{}, &PGList{})
}
