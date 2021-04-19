// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceIntegrationSpec defines the desired state of ServiceIntegration
type ServiceIntegrationSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Project the integration belongs to
	Project string `json:"project"`

	// +kubebuilder:validation:Enum=datadog;kafka_logs;kafka_connect;metrics;dashboard;rsyslog;read_replica;schema_registry_proxy;signalfx;jolokia;internal_connectivity;external_google_cloud_logging;datasource
	// Type of the service integration
	IntegrationType string `json:"integration_type"`

	// Source endpoint for the integration (if any)
	SourceEndpointID string `json:"source_endpoint_id,omitempty"`

	// Source service for the integration (if any)
	SourceServiceName string `json:"source_service_name,omitempty"`

	// Destination endpoint for the integration (if any)
	DestinationEndpointID string `json:"destination_endpoint_id,omitempty"`

	// Destination service for the integration (if any)
	DestinationServiceName string `json:"destination_service_name,omitempty"`

	// Datadog specific user configuration options
	DatadogUserConfig ServiceIntegrationDatadogUserConfig `json:"datadog,omitempty"`

	// Kafka Connect service configuration values
	KafkaConnectUserConfig ServiceIntegrationKafkaConnectUserConfig `json:"kafka_connect,omitempty"`

	// Kafka logs configuration values
	KafkaLogsUserConfig ServiceIntegrationKafkaLogsUserConfig `json:"kafka_logs,omitempty"`

	// Metrics configuration values
	MetricsUserConfig ServiceIntegrationMetricsUserConfig `json:"metrics,omitempty"`
}

// ServiceIntegrationStatus defines the observed state of ServiceIntegration
type ServiceIntegrationStatus struct {
	ServiceIntegrationSpec `json:",inline"`

	// Service integration ID
	ID string `json:"id"`
}

type ServiceIntegrationMetricsUserConfig struct {
	// +kubebuilder:validation:Format="^[_A-Za-z0-9][-_A-Za-z0-9]{0,39}$"
	// +kubebuilder:validation:MaxLength=40
	// Name of the database where to store metric datapoints. Only affects PostgreSQL destinations
	Database string `json:"database,omitempty"`

	// +kubebuilder:validation:Max=10000
	// Number of days to keep old metrics. Only affects PostgreSQL destinations. Set to 0 for no automatic cleanup. Defaults to 30 days.
	RetentionDays int `json:"retention_days,omitempty"`

	// +kubebuilder:validation:Format="^[_A-Za-z0-9][-._A-Za-z0-9]{0,39}$"
	// +kubebuilder:validation:MaxLength=40
	// Name of a user that can be used to read metrics. This will be used for Grafana integration (if enabled) to prevent Grafana users from making undesired changes. Only affects PostgreSQL destinations. Defaults to 'metrics_reader'. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service.
	RoUsername string `json:"ro_username,omitempty"`

	// +kubebuilder:validation:Format="^[_A-Za-z0-9][-._A-Za-z0-9]{0,39}$"
	// +kubebuilder:validation:MaxLength=40
	// Name of the user used to write metrics. Only affects PostgreSQL destinations. Defaults to 'metrics_writer'. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service.
	Username string `json:"username,omitempty"`
}

type ServiceIntegrationKafkaLogsUserConfig struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:MinLength=1
	// Topic name
	KafkaTopic string `json:"kafka_topic,omitempty"`
}

type ServiceIntegrationDatadogUserConfig struct {
	// Consumer groups to exclude
	ExcludeConsumerGroups []string `json:"exclude_consumer_groups,omitempty"`

	// List of topics to exclude
	ExcludeTopics []string `json:"exclude_topics,omitempty"`

	// Consumer groups to include
	IncludeConsumerGroups []string `json:"include_consumer_groups,omitempty"`

	// Topics to include
	IncludeTopics []string `json:"include_topics,omitempty"`

	// List of custom metrics
	KafkaCustomMetrics []string `json:"kafka_custom_metrics,omitempty"`
}

type ServiceIntegrationKafkaConnectUserConfig struct {
	KafkaConnect ServiceIntegrationKafkaConnect `json:"kafka_connect,omitempty"`
}

type ServiceIntegrationKafkaConnect struct {
	// +kubebuilder:validation:MaxLength=249
	// The name of the topic where connector and task configuration data are stored. This must be the same for all workers with the same group_id.
	ConfigStorageTopic string `json:"config_storage_topic,omitempty"`

	// +kubebuilder:validation:MaxLength=249
	// A unique string that identifies the Connect cluster group this worker belongs to.
	GroupId string `json:"group_id,omitempty"`

	// +kubebuilder:validation:MaxLength=249
	// The name of the topic where connector and task configuration offsets are stored. This must be the same for all workers with the same group_id.
	OffsetStorageTopic string `json:"offset_storage_topic,omitempty"`

	// +kubebuilder:validation:MaxLength=249
	// The name of the topic where connector and task configuration status updates are stored.This must be the same for all workers with the same group_id.
	StatusStorageTopic string `json:"status_storage_topic,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceIntegration is the Schema for the serviceintegrations API
type ServiceIntegration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceIntegrationSpec   `json:"spec,omitempty"`
	Status ServiceIntegrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceIntegrationList contains a list of ServiceIntegration
type ServiceIntegrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceIntegration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceIntegration{}, &ServiceIntegrationList{})
}
