// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"github.com/aiven/go-client-codegen/handler/vpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectVPCSpec defines the desired state of ProjectVPC
type ProjectVPCSpec struct {
	ProjectDependant `json:",inline"`

	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Cloud the VPC is in
	CloudName string `json:"cloudName"`

	// +kubebuilder:validation:MaxLength=36
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Network address range used by the VPC like 192.168.0.0/24
	NetworkCidr string `json:"networkCidr"`
}

// ProjectVPCStatus defines the observed state of ProjectVPC
type ProjectVPCStatus struct {
	// Conditions represent the latest available observations of an ProjectVPC state
	Conditions []metav1.Condition `json:"conditions"`

	// State of VPC
	State vpc.VpcStateType `json:"state"`

	// Project VPC id
	ID string `json:"id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ProjectVPC is the Schema for the projectvpcs API
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Cloud",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Network CIDR",type="string",JSONPath=".spec.networkCidr"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type ProjectVPC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectVPCSpec   `json:"spec,omitempty"`
	Status ProjectVPCStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &ProjectVPC{}

func (*ProjectVPC) NoSecret() bool {
	return true
}

func (in *ProjectVPC) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *ProjectVPC) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *ProjectVPC) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
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
