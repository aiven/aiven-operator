// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	flinkuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/flink"
)

// FlinkSpec defines the desired state of Flink
type FlinkSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Cassandra specific user configuration options
	UserConfig *flinkuserconfig.FlinkUserConfig `json:"userConfig,omitempty"`
}

// Flink is the Schema for the flinks API.
// Info "Exposes secret keys": `FLINK_HOST`, `FLINK_PORT`, `FLINK_USER`, `FLINK_PASSWORD`, `FLINK_URI`, `FLINK_HOSTS`
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type Flink struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlinkSpec     `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &Flink{}

func (in *Flink) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *Flink) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *Flink) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *Flink) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func (in *Flink) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

//+kubebuilder:object:root=true

// FlinkList contains a list of Flink
type FlinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Flink `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Flink{}, &FlinkList{})
}
