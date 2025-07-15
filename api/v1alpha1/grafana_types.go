// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grafanauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/grafana"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GrafanaSpec defines the desired state of Grafana
type GrafanaSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Cassandra specific user configuration options
	UserConfig *grafanauserconfig.GrafanaUserConfig `json:"userConfig,omitempty"`
}

// Grafana is the Schema for the grafanas API.
// Info "Exposes secret keys": `GRAFANA_HOST`, `GRAFANA_PORT`, `GRAFANA_USER`, `GRAFANA_PASSWORD`, `GRAFANA_URI`, `GRAFANA_HOSTS`
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type Grafana struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GrafanaSpec   `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &Grafana{}

func (in *Grafana) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *Grafana) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *Grafana) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *Grafana) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *Grafana) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func (in *Grafana) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

//+kubebuilder:object:root=true

// GrafanaList contains a list of Grafana
type GrafanaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Grafana `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Grafana{}, &GrafanaList{})
}
