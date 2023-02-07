// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clickhouseuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/clickhouse"
)

// ClickhouseSpec defines the desired state of Clickhouse
type ClickhouseSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Information regarding secret creation
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// OpenSearch specific user configuration options
	UserConfig *clickhouseuserconfig.ClickhouseUserConfig `json:"userConfig,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Clickhouse is the Schema for the clickhouses API
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

func (in *Clickhouse) AuthSecretRef() AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *Clickhouse) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func init() {
	SchemeBuilder.Register(&Clickhouse{}, &ClickhouseList{})
}
