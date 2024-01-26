// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

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

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`

	// +kubebuilder:validation:MaxLength=1024
	// The Java class of the connector.
	ConnectorClass string `json:"connectorClass"`

	// The connector specific configuration
	// To build config values from secret the template function `{{ fromSecret "name" "key" }}`
	// is provided when interpreting the keys
	UserConfig map[string]string `json:"userConfig"`
}

// KafkaConnectorStatus defines the observed state of KafkaConnector
type KafkaConnectorStatus struct {
	// Conditions represent the latest available observations of an kafka connector state
	Conditions []metav1.Condition `json:"conditions"`

	// Connector state
	State string `json:"state"`

	// PluginStatus contains metadata about the configured connector plugin
	PluginStatus KafkaConnectorPluginStatus `json:"pluginStatus"`

	// TasksStatus contains metadata about the running tasks
	TasksStatus KafkaConnectorTasksStatus `json:"tasksStatus"`
}

// KafkaConnectorPluginStatus describes the observed state of a Kafka Connector Plugin
type KafkaConnectorPluginStatus struct {
	Author  string `json:"author"`
	Class   string `json:"class"`
	DocURL  string `json:"docUrl"`
	Title   string `json:"title"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

// KafkaConnectorTasksStatus describes the observed state of the Kafka Connector Tasks
type KafkaConnectorTasksStatus struct {
	Total      uint   `json:"total"`
	Running    uint   `json:"running,omitempty"`
	Failed     uint   `json:"failed,omitempty"`
	Paused     uint   `json:"paused,omitempty"`
	Unassigned uint   `json:"unassigned,omitempty"`
	Unknown    uint   `json:"unknown,omitempty"`
	StackTrace string `json:"stackTrace,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KafkaConnector is the Schema for the kafkaconnectors API
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Connector Class",type="string",JSONPath=".spec.connectorClass"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Tasks Total",type="integer",JSONPath=".status.tasksStatus.total"
// +kubebuilder:printcolumn:name="Tasks Running",type="integer",JSONPath=".status.tasksStatus.running"
type KafkaConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaConnectorSpec   `json:"spec,omitempty"`
	Status KafkaConnectorStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &KafkaConnector{}

func (in *KafkaConnector) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *KafkaConnector) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (*KafkaConnector) NoSecret() bool {
	return false
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
