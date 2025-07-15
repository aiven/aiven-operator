// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"github.com/aiven/go-client-codegen/handler/kafka"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaACLSpec defines the desired state of KafkaACL
type KafkaACLSpec struct {
	ServiceDependant `json:",inline"`

	// +kubebuilder:validation:Enum=admin;read;readwrite;write
	// Kafka permission to grant (admin, read, readwrite, write)
	Permission kafka.PermissionType `json:"permission"`

	// Topic name pattern for the ACL entry
	Topic string `json:"topic"`

	// Username pattern for the ACL entry
	Username string `json:"username"`
}

// KafkaACLStatus defines the observed state of KafkaACL
type KafkaACLStatus struct {
	// Conditions represent the latest available observations of an KafkaACL state
	Conditions []metav1.Condition `json:"conditions"`

	// Kafka ACL ID
	ID string `json:"id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KafkaACL is the Schema for the kafkaacls API
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Username",type="string",JSONPath=".spec.username"
// +kubebuilder:printcolumn:name="Permission",type="string",JSONPath=".spec.permission"
// +kubebuilder:printcolumn:name="Topic",type="string",JSONPath=".spec.topic"
type KafkaACL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaACLSpec   `json:"spec,omitempty"`
	Status KafkaACLStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &KafkaACL{}

func (in *KafkaACL) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *KafkaACL) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *KafkaACL) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (*KafkaACL) NoSecret() bool {
	return true
}

// +kubebuilder:object:root=true

// KafkaACLList contains a list of KafkaACL
type KafkaACLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaACL `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaACL{}, &KafkaACLList{})
}
