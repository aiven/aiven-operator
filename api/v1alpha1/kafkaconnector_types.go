// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaConnectorSpec defines the desired state of KafkaConnector
type KafkaConnectorSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Target project.
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// Service name.
	ServiceName string `json:"serviceName"`

	// +kubebuilder:validation:MaxLength=1024
	// The Java class of the connector.
	ConnectorClass string `json:"connectorClass"`

	// The connector specific configuration
	// To use secrets as sources for values you should write
	// `configOption: secretRef:key:value`
	ConnectorSpecificConfig map[string]string `json:"connectorSpecificConfig"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef AuthSecretReference `json:"authSecretRef"`
}

type KafkaConnectorConnectorSpecificConfig map[string]string

// KafkaConnectorStatus defines the observed state of KafkaConnector
type KafkaConnectorStatus struct {
	// Conditions represent the latest available observations of an kafka connector state
	Conditions []metav1.Condition `json:"conditions"`

	// State represents the state of the kafka connector
	State string `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KafkaConnector is the Schema for the kafkaconnectors API
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Connector Class",type="string",JSONPath=".spec.ConnectorClass"
type KafkaConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaConnectorSpec   `json:"spec,omitempty"`
	Status KafkaConnectorStatus `json:"status,omitempty"`
}

func (kfk KafkaConnector) AuthSecretRef() AuthSecretReference {
	return kfk.Spec.AuthSecretRef
}

//+kubebuilder:object:root=true

// KafkaConnectorList contains a list of KafkaConnector
type KafkaConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaConnector{}, &KafkaConnectorList{})
}
