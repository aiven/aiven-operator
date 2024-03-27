// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaSchemaRegistryACLSpec defines the desired state of KafkaSchemaRegistryACL
type KafkaSchemaRegistryACLSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Identifies the project this resource belongs to
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Specifies the name of the service that this resource belongs to
	ServiceName string `json:"serviceName"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`

	// +kubebuilder:validation:Enum=schema_registry_read;schema_registry_write
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Permission string `json:"permission"`

	// +kubebuilder:validation:MaxLength=249
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Resource name pattern for the Schema Registry ACL entry
	Resource string `json:"resource"`

	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Username pattern for the ACL entry
	Username string `json:"username"`
}

// KafkaSchemaRegistryACLStatus defines the observed state of KafkaSchemaRegistryACL
type KafkaSchemaRegistryACLStatus struct {
	// Conditions represent the latest available observations of an KafkaSchemaRegistryACL state
	Conditions []metav1.Condition `json:"conditions"`

	// Kafka ACL ID
	ACLId string `json:"acl_id"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KafkaSchemaRegistryACL is the Schema for the kafkaschemaregistryacls API
type KafkaSchemaRegistryACL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSchemaRegistryACLSpec   `json:"spec,omitempty"`
	Status KafkaSchemaRegistryACLStatus `json:"status,omitempty"`
}

func (in *KafkaSchemaRegistryACL) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *KafkaSchemaRegistryACL) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *KafkaSchemaRegistryACL) NoSecret() bool {
	return true
}

var _ AivenManagedObject = &KafkaSchemaRegistryACL{}

//+kubebuilder:object:root=true

// KafkaSchemaRegistryACLList contains a list of KafkaSchemaRegistryACL
type KafkaSchemaRegistryACLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaSchemaRegistryACL `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaSchemaRegistryACL{}, &KafkaSchemaRegistryACLList{})
}
