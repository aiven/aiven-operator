// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaSchemaSpec defines the desired state of KafkaSchema
type KafkaSchemaSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Project to link the Kafka Schema to
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// Service to link the Kafka Schema to
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=63
	// Kafka Schema Subject name
	SubjectName string `json:"subject_name"`

	// Kafka Schema configuration should be a valid Avro Schema JSON format
	Schema string `json:"schema"`

	// +kubebuilder:validation:Enum=BACKWARD;BACKWARD_TRANSITIVE;FORWARD;FORWARD_TRANSITIVE;FULL;FULL_TRANSITIVE;NONE
	// Kafka Schemas compatibility level
	CompatibilityLevel string `json:"compatibility_level,omitempty"`
}

// KafkaSchemaStatus defines the observed state of KafkaSchema
type KafkaSchemaStatus struct {
	KafkaSchemaSpec `json:",inline"`

	// Kafka Schema configuration version
	Version int `json:"version"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KafkaSchema is the Schema for the kafkaschemas API
type KafkaSchema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSchemaSpec   `json:"spec,omitempty"`
	Status KafkaSchemaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KafkaSchemaList contains a list of KafkaSchema
type KafkaSchemaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaSchema `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaSchema{}, &KafkaSchemaList{})
}
