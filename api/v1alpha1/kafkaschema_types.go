// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

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
	ServiceName string `json:"serviceName"`

	// +kubebuilder:validation:MaxLength=63
	// Kafka Schema Subject name
	SubjectName string `json:"subjectName"`

	// Kafka Schema configuration should be a valid Avro Schema JSON format
	Schema string `json:"schema"`

	// +kubebuilder:validation:Enum=BACKWARD;BACKWARD_TRANSITIVE;FORWARD;FORWARD_TRANSITIVE;FULL;FULL_TRANSITIVE;NONE
	// Kafka Schemas compatibility level
	CompatibilityLevel string `json:"compatibilityLevel,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`
}

// KafkaSchemaStatus defines the observed state of KafkaSchema
type KafkaSchemaStatus struct {
	// Conditions represent the latest available observations of an KafkaSchema state
	Conditions []metav1.Condition `json:"conditions"`

	// Kafka Schema configuration version
	Version int `json:"version"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KafkaSchema is the Schema for the kafkaschemas API
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Subject",type="string",JSONPath=".spec.subjectName"
// +kubebuilder:printcolumn:name="Compatibility Level",type="string",JSONPath=".spec.compatibilityLevel"
// +kubebuilder:printcolumn:name="Version",type="number",JSONPath=".status.version"
type KafkaSchema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSchemaSpec   `json:"spec,omitempty"`
	Status KafkaSchemaStatus `json:"status,omitempty"`
}

func (in *KafkaSchema) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
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
