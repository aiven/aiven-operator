// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clickhouseuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/clickhouse"
)

// ClickhouseSpec defines the desired state of Clickhouse
type ClickhouseSpec struct {
	ServiceCommonSpec `json:",inline"`

	// OpenSearch specific user configuration options
	UserConfig *clickhouseuserconfig.ClickhouseUserConfig `json:"userConfig,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Clickhouse is the Schema for the clickhouses API.
// Info "Exposes secret keys": `CLICKHOUSE_HOST`, `CLICKHOUSE_PORT`, `CLICKHOUSE_USER`, `CLICKHOUSE_PASSWORD`
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type Clickhouse struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClickhouseSpec `json:"spec,omitempty"`
	Status ServiceStatus  `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClickhouseList contains a list of Clickhouse
type ClickhouseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Clickhouse `json:"items"`
}

var _ AivenManagedObject = &Clickhouse{}

func (in *Clickhouse) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *Clickhouse) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *Clickhouse) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *Clickhouse) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *Clickhouse) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func (in *Clickhouse) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

func init() {
	SchemeBuilder.Register(&Clickhouse{}, &ClickhouseList{})
}
