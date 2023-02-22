// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cassandrauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/cassandra"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CassandraSpec defines the desired state of Cassandra
type CassandraSpec struct {
	ServiceCommonSpec `json:",inline"`

	// +kubebuilder:validation:Format="^[1-9][0-9]*(GiB|G)*"
	// The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.
	DiskSpace string `json:"disk_space,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`

	// Information regarding secret creation
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// Cassandra specific user configuration options
	UserConfig *cassandrauserconfig.CassandraUserConfig `json:"userConfig,omitempty"`
}

// Cassandra is the Schema for the cassandras API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type Cassandra struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CassandraSpec `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

func (in *Cassandra) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *Cassandra) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

//+kubebuilder:object:root=true

// CassandraList contains a list of Cassandra
type CassandraList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cassandra `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cassandra{}, &CassandraList{})
}
