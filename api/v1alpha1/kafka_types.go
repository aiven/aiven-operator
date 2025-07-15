// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
)

// KafkaSpec defines the desired state of Kafka
type KafkaSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Switch the service to use Karapace for schema registry and REST proxy
	Karapace *bool `json:"karapace,omitempty"`

	// Kafka specific user configuration options
	UserConfig *kafkauserconfig.KafkaUserConfig `json:"userConfig,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Kafka is the Schema for the kafkas API.
// Info "Exposes secret keys": `KAFKA_HOST`, `KAFKA_PORT`, `KAFKA_USERNAME`, `KAFKA_PASSWORD`, `KAFKA_ACCESS_CERT`, `KAFKA_ACCESS_KEY`, `KAFKA_SASL_HOST`, `KAFKA_SASL_PORT`, `KAFKA_SCHEMA_REGISTRY_HOST`, `KAFKA_SCHEMA_REGISTRY_PORT`, `KAFKA_CONNECT_HOST`, `KAFKA_CONNECT_PORT`, `KAFKA_REST_HOST`, `KAFKA_REST_PORT`, `KAFKA_CA_CERT`
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type Kafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSpec     `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &Kafka{}

func (in *Kafka) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *Kafka) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *Kafka) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *Kafka) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *Kafka) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func (in *Kafka) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

// +kubebuilder:object:root=true

// KafkaList contains a list of Kafka
type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kafka `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kafka{}, &KafkaList{})
}
