// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grafanauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/grafana"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GrafanaSpec defines the desired state of Grafana
// +kubebuilder:validation:XValidation:rule="has(oldSelf.connInfoSecretTargetDisabled) == has(self.connInfoSecretTargetDisabled)",message="connInfoSecretTargetDisabled can only be set during resource creation."
type GrafanaSpec struct {
	ServiceCommonSpec `json:",inline"`

	// +kubebuilder:validation:Format="^[1-9][0-9]*(GiB|G)*"
	// The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.
	DiskSpace string `json:"disk_space,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`

	// Information regarding secret creation.
	// Exposed keys: `GRAFANA_HOST`, `GRAFANA_PORT`, `GRAFANA_USER`, `GRAFANA_PASSWORD`, `GRAFANA_URI`, `GRAFANA_HOSTS`
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="connInfoSecretTargetDisabled is immutable."
	// When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
	ConnInfoSecretTargetDisabled *bool `json:"connInfoSecretTargetDisabled,omitempty"`

	// Cassandra specific user configuration options
	UserConfig *grafanauserconfig.GrafanaUserConfig `json:"userConfig,omitempty"`
}

// Grafana is the Schema for the grafanas API
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
