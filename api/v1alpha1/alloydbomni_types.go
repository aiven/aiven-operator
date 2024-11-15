// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	alloydbomni "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/alloydbomni"
)

// AlloyDBOmniSpec defines the desired state of AlloyDB Omni instance
type AlloyDBOmniSpec struct {
	ServiceCommonSpec `json:",inline"`

	// AlloyDBOmni specific user configuration options
	UserConfig *alloydbomni.AlloydbomniUserConfig `json:"userConfig,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AlloyDBOmni is the Schema for the alloydbomni API.
// Info "Exposes secret keys": `ALLOYDBOMNI_HOST`, `ALLOYDBOMNI_PORT`, `ALLOYDBOMNI_DATABASE`, `ALLOYDBOMNI_USER`, `ALLOYDBOMNI_PASSWORD`, `ALLOYDBOMNI_SSLMODE`, `ALLOYDBOMNI_DATABASE_URI`
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type AlloyDBOmni struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlloyDBOmniSpec `json:"spec,omitempty"`
	Status ServiceStatus   `json:"status,omitempty"`
}

var _ AivenManagedObject = &AlloyDBOmni{}

func (in *AlloyDBOmni) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *AlloyDBOmni) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *AlloyDBOmni) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *AlloyDBOmni) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func (in *AlloyDBOmni) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

// +kubebuilder:object:root=true

// AlloyDBOmniList contains a list of AlloyDBOmni instances
type AlloyDBOmniList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlloyDBOmni `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlloyDBOmni{}, &AlloyDBOmniList{})
}
