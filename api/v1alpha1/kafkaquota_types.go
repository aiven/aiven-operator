// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaQuotaSpec defines the desired state of KafkaQuota
// +kubebuilder:validation:XValidation:rule="has(self.user) || has(self.clientId)",message="At least one of user or clientId must be set"
// +kubebuilder:validation:XValidation:rule="has(self.consumerByteRate) || has(self.producerByteRate) || has(self.requestPercentage)",message="At least one of consumerByteRate, producerByteRate or requestPercentage must be set"
type KafkaQuotaSpec struct {
	ServiceDependant `json:",inline"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Represents a logical group of clients, assigned a unique name by the client application.
	// Quotas can be applied based on user, client-id, or both.
	// The most relevant quota is chosen for each connection. All connections within a quota group share the same quota.
	// It is possible to set default quotas for each (user, client-id), user or client-id group by specifying 'default'.
	User string `json:"user,omitempty"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Represents a logical group of clients, assigned a unique name by the client application.
	// Quotas can be applied based on user, client-id, or both.
	// The most relevant quota is chosen for each connection. All connections within a quota group share the same quota.
	// It is possible to set default quotas for each (user, client-id), user or client-id group by specifying 'default'.
	ClientID string `json:"clientId,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1073741824
	// Defines the bandwidth limit in bytes/sec for each group of clients sharing a quota.
	// Every distinct client group is allocated a specific quota, as defined by the cluster, on a per-broker basis.
	// Exceeding this limit results in client throttling.
	ConsumerByteRate *int64 `json:"consumerByteRate,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1073741824
	// Defines the bandwidth limit in bytes/sec for each group of clients sharing a quota.
	// Every distinct client group is allocated a specific quota, as defined by the cluster, on a per-broker basis.
	// Exceeding this limit results in client throttling.
	ProducerByteRate *int64 `json:"producerByteRate,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// Sets the maximum percentage of CPU time that a client group can use on request handler I/O and network threads per broker within a quota window.
	// Exceeding this limit triggers throttling. The quota, expressed as a percentage, also indicates the total allowable CPU usage
	// for the client groups sharing the quota.
	RequestPercentage *float64 `json:"requestPercentage,omitempty"`
}

// KafkaQuotaStatus defines the observed state of KafkaQuota
type KafkaQuotaStatus struct {
	// Conditions represent the latest available observations of a KafkaQuota state
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KafkaQuota is the Schema for the kafkaquotas API.
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="User",type="string",JSONPath=".spec.user"
// +kubebuilder:printcolumn:name="Client ID",type="string",JSONPath=".spec.clientId"
type KafkaQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaQuotaSpec   `json:"spec,omitempty"`
	Status KafkaQuotaStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &KafkaQuota{}

func (*KafkaQuota) NoSecret() bool {
	return true
}

func (in *KafkaQuota) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *KafkaQuota) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *KafkaQuota) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

//+kubebuilder:object:root=true

// KafkaQuotaList contains a list of KafkaQuota
type KafkaQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaQuota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaQuota{}, &KafkaQuotaList{})
}
