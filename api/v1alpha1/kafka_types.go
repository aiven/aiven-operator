// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
)

// KafkaSpec defines the desired state of Kafka
// +kubebuilder:validation:XValidation:rule="has(oldSelf.connInfoSecretTargetDisabled) == has(self.connInfoSecretTargetDisabled)",message="connInfoSecretTargetDisabled can only be set during resource creation."
type KafkaSpec struct {
	ServiceCommonSpec `json:",inline"`

	// +kubebuilder:validation:Format="^[1-9][0-9]*(GiB|G)*"
	// The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.
	DiskSpace string `json:"disk_space,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`

	// Information regarding secret creation.
	// Exposed keys: `KAFKA_HOST`, `KAFKA_PORT`, `KAFKA_USERNAME`, `KAFKA_PASSWORD`, `KAFKA_ACCESS_CERT`, `KAFKA_ACCESS_KEY`, `KAFKA_SASL_HOST`, `KAFKA_SASL_PORT`, `KAFKA_SCHEMA_REGISTRY_HOST`, `KAFKA_SCHEMA_REGISTRY_PORT`, `KAFKA_CONNECT_HOST`, `KAFKA_CONNECT_PORT`, `KAFKA_REST_HOST`, `KAFKA_REST_PORT`
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="connInfoSecretTargetDisabled is immutable."
	// When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
	ConnInfoSecretTargetDisabled *bool `json:"connInfoSecretTargetDisabled,omitempty"`

	// Switch the service to use Karapace for schema registry and REST proxy
	Karapace *bool `json:"karapace,omitempty"`

	// Kafka specific user configuration options
	UserConfig *kafkauserconfig.KafkaUserConfig `json:"userConfig,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Kafka is the Schema for the kafkas API
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

func (in *Kafka) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *Kafka) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
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
