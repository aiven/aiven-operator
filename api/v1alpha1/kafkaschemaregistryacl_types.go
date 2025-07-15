// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaSchemaRegistryACLSpec defines the desired state of KafkaSchemaRegistryACL
type KafkaSchemaRegistryACLSpec struct {
	ServiceDependant `json:",inline"`

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
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Resource",type="string",JSONPath=".spec.resource"
// +kubebuilder:printcolumn:name="Username",type="string",JSONPath=".spec.username"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
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

func (in *KafkaSchemaRegistryACL) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
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
