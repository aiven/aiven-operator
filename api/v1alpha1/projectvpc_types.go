// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectVPCSpec defines the desired state of ProjectVPC
type ProjectVPCSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// x-kubernetes-immutable: true
	// The project the VPC belongs to
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=256
	// x-kubernetes-immutable: true
	// Cloud the VPC is in
	CloudName string `json:"cloud_name"`

	// +kubebuilder:validation:MaxLength=36
	// x-kubernetes-immutable: true
	// Network address range used by the VPC like 192.168.0.0/24
	NetworkCidr string `json:"network_cidr"`
}

// ProjectVPCStatus defines the observed state of ProjectVPC
type ProjectVPCStatus struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// The project the VPC belongs to
	Project string `json:"project"`

	// Cloud the VPC is in
	// +kubebuilder:validation:MaxLength=256
	CloudName string `json:"cloud_name"`

	// +kubebuilder:validation:MaxLength=36
	// Network address range used by the VPC like 192.168.0.0/24
	NetworkCidr string `json:"network_cidr"`

	// State of the VPC (APPROVED, ACTIVE, DELETING, DELETED)
	State string `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ProjectVPC is the Schema for the projectvpcs API
type ProjectVPC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectVPCSpec   `json:"spec,omitempty"`
	Status ProjectVPCStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectVPCList contains a list of ProjectVPC
type ProjectVPCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectVPC `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectVPC{}, &ProjectVPCList{})
}
