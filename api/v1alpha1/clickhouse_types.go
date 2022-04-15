// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClickhouseSpec defines the desired state of Clickhouse
type ClickhouseSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef AuthSecretReference `json:"authSecretRef,omitempty"`

	// Information regarding secret creation
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// OpenSearch specific user configuration options
	UserConfig ClickhouseUserConfig `json:"userConfig,omitempty"`
}

type ClickhouseUserConfig struct {
	// Glob pattern and number of indexes matching that pattern to be kept Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like 'logs.?' and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note 'logs.?' does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.
	// IP filter Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'
	IpFilter []string `json:"ip_filter,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Name of another project to fork a service from. This has effect only when a new service is being created.
	ProjectToForkFrom string `json:"project_to_fork_from,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Name of another service to fork from. This has effect only when a new service is being created.
	ServiceToForkFrom string `json:"service_to_fork_from,omitempty"`
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

func (o Clickhouse) AuthSecretRef() AuthSecretReference {
	return o.Spec.AuthSecretRef
}

func init() {
	SchemeBuilder.Register(&Clickhouse{}, &ClickhouseList{})
}
