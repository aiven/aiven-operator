// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"github.com/aiven/go-client-codegen/handler/kafka"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaNativeACLSpec defines the desired state of KafkaNativeACL
type KafkaNativeACLSpec struct {
	ServiceDependant `json:",inline"`

	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:default="*"
	// The host or `*` for all hosts
	Host string `json:"host,omitempty"`

	// +kubebuilder:validation:Enum=All;Alter;AlterConfigs;ClusterAction;Create;CreateTokens;Delete;Describe;DescribeConfigs;DescribeTokens;IdempotentWrite;Read;Write
	// Kafka ACL operation represents an operation which an ACL grants or denies permission to perform
	Operation kafka.OperationType `json:"operation"`

	// +kubebuilder:validation:Enum=LITERAL;PREFIXED
	// Kafka ACL pattern type of resource name
	PatternType kafka.PatternType `json:"patternType"`

	// +kubebuilder:validation:Enum=ALLOW;DENY
	// Kafka ACL permission type
	PermissionType kafka.ServiceKafkaNativeAclPermissionType `json:"permissionType"`

	// +kubebuilder:validation:MaxLength=256
	// Principal is in 'PrincipalType:name' format
	Principal string `json:"principal"`

	// +kubebuilder:validation:MaxLength=256
	// Resource pattern used to match specified resources
	ResourceName string `json:"resourceName"`

	// +kubebuilder:validation:Enum=Cluster;DelegationToken;Group;Topic;TransactionalId;User
	// Kafka ACL resource type represents a type of resource which an ACL can be applied to
	ResourceType kafka.ResourceType `json:"resourceType"`
}

// KafkaNativeACLStatus defines the observed state of KafkaNativeACL
type KafkaNativeACLStatus struct {
	// Conditions represent the latest available observations of an KafkaNativeACL state
	Conditions []metav1.Condition `json:"conditions"`

	// Kafka-native ACL ID
	ID string `json:"id"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KafkaNativeACL
// Creates and manages Kafka-native access control lists (ACLs) for an Aiven for Apache Kafka® service.
// ACLs control access to Kafka topics, consumer groups, clusters, and Schema Registry.
// Kafka-native ACLs provide advanced resource-level access control with fine-grained permissions, including ALLOW and DENY rules.
// For simplified topic-level control, you can use [KafkaACL](./kafkaacl.md).
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Host",type="string",JSONPath=".spec.host"
// +kubebuilder:printcolumn:name="Operation",type="string",JSONPath=".spec.operation"
// +kubebuilder:printcolumn:name="PatternType",type="string",JSONPath=".spec.patternType"
// +kubebuilder:printcolumn:name="PermissionType",type="string",JSONPath=".spec.permissionType"
type KafkaNativeACL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Spec   KafkaNativeACLSpec   `json:"spec,omitempty"`
	Status KafkaNativeACLStatus `json:"status,omitempty"`
}

func (in *KafkaNativeACL) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *KafkaNativeACL) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *KafkaNativeACL) NoSecret() bool {
	return true
}

var _ AivenManagedObject = &KafkaNativeACL{}

//+kubebuilder:object:root=true

// KafkaNativeACLList contains a list of KafkaNativeACL
type KafkaNativeACLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaNativeACL `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaNativeACL{}, &KafkaNativeACLList{})
}
