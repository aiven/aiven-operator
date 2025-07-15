// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kafkaconnectuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka_connect"
)

// KafkaConnectSpec defines the desired state of KafkaConnect
type KafkaConnectSpec struct {
	BaseServiceFields `json:",inline"`

	// KafkaConnect specific user configuration options
	UserConfig *kafkaconnectuserconfig.KafkaConnectUserConfig `json:"userConfig,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KafkaConnect is the Schema for the kafkaconnects API
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type KafkaConnect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaConnectSpec `json:"spec,omitempty"`
	Status ServiceStatus    `json:"status,omitempty"`
}

var _ AivenManagedObject = &KafkaConnect{}

func (*KafkaConnect) NoSecret() bool {
	return true
}

func (in *KafkaConnect) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *KafkaConnect) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *KafkaConnect) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *KafkaConnect) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

// +kubebuilder:object:root=true

// KafkaConnectList contains a list of KafkaConnect
type KafkaConnectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaConnect `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaConnect{}, &KafkaConnectList{})
}
