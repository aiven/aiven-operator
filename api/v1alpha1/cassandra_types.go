// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cassandrauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/cassandra"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CassandraSpec defines the desired state of Cassandra
type CassandraSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Cassandra specific user configuration options
	UserConfig *cassandrauserconfig.CassandraUserConfig `json:"userConfig,omitempty"`
}

// Cassandra is the Schema for the cassandras API.
// Info "Exposes secret keys": `CASSANDRA_HOST`, `CASSANDRA_PORT`, `CASSANDRA_USER`, `CASSANDRA_PASSWORD`, `CASSANDRA_URI`, `CASSANDRA_HOSTS`, `CASSANDRA_CA_CERT`
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion:warning="EOL date **December 31, 2025**, see [end-of-life](https://aiven.io/docs/platform/reference/end-of-life). To ensure uninterrupted service, complete your migration out of Aiven for Apache Cassandra before December 31, 2025."
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

var _ AivenManagedObject = &Cassandra{}

func (in *Cassandra) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *Cassandra) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *Cassandra) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *Cassandra) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func (in *Cassandra) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
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
