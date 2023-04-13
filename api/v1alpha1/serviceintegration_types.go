// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clickhousekafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/clickhouse_kafka"
	clickhousepostgresqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/clickhouse_postgresql"
	datadogintegration "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/datadog"
	externalawscloudwatchmetricsuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/external_aws_cloudwatch_metrics"
	kafkaconnectintegration "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/kafka_connect"
	kafkalogsintegration "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/kafka_logs"
	kafkamirrormakeruserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/kafka_mirrormaker"
	logsuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/logs"
	metricsintegration "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/metrics"
)

// ServiceIntegrationSpec defines the desired state of ServiceIntegration
type ServiceIntegrationSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Project the integration belongs to
	Project string `json:"project"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:Enum=datadog;kafka_logs;kafka_connect;metrics;dashboard;rsyslog;read_replica;schema_registry_proxy;jolokia;internal_connectivity;external_google_cloud_logging;datasource;clickhouse_postgresql;clickhouse_kafka;logs;external_aws_cloudwatch_metrics
	// Type of the service integration
	IntegrationType string `json:"integrationType"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Source endpoint for the integration (if any)
	SourceEndpointID string `json:"sourceEndpointID,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Source service for the integration (if any)
	SourceServiceName string `json:"sourceServiceName,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Destination endpoint for the integration (if any)
	DestinationEndpointID string `json:"destinationEndpointId,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Destination service for the integration (if any)
	DestinationServiceName string `json:"destinationServiceName,omitempty"`

	// Datadog specific user configuration options
	DatadogUserConfig *datadogintegration.DatadogUserConfig `json:"datadog,omitempty"`

	// Kafka Connect service configuration values
	KafkaConnectUserConfig *kafkaconnectintegration.KafkaConnectUserConfig `json:"kafkaConnect,omitempty"`

	// Kafka logs configuration values
	KafkaLogsUserConfig *kafkalogsintegration.KafkaLogsUserConfig `json:"kafkaLogs,omitempty"`

	// Metrics configuration values
	MetricsUserConfig *metricsintegration.MetricsUserConfig `json:"metrics,omitempty"`

	// Clickhouse PostgreSQL configuration values
	ClickhousePostgreSQLUserConfig *clickhousepostgresqluserconfig.ClickhousePostgresqlUserConfig `json:"clickhousePostgresql,omitempty"`

	// Clickhouse Kafka configuration values
	ClickhouseKafkaUserConfig *clickhousekafkauserconfig.ClickhouseKafkaUserConfig `json:"clickhouseKafka,omitempty"`

	// Kafka MirrorMaker configuration values
	KafkaMirrormakerUserConfig *kafkamirrormakeruserconfig.KafkaMirrormakerUserConfig `json:"kafkaMirrormaker,omitempty"`

	// Logs configuration values
	LogsUserConfig *logsuserconfig.LogsUserConfig `json:"logs,omitempty"`

	// External AWS CloudWatch Metrics integration Logs configuration values
	ExternalAWSCloudwatchMetricsUserConfig *externalawscloudwatchmetricsuserconfig.ExternalAwsCloudwatchMetricsUserConfig `json:"external_aws_cloudwatch_metrics,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`
}

// ServiceIntegrationStatus defines the observed state of ServiceIntegration
type ServiceIntegrationStatus struct {
	// Conditions represent the latest available observations of an ServiceIntegration state
	Conditions []metav1.Condition `json:"conditions"`

	// Service integration ID
	ID string `json:"id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceIntegration is the Schema for the serviceintegrations API
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.integrationType"
// +kubebuilder:printcolumn:name="Source Service Name",type="string",JSONPath=".spec.sourceServiceName"
// +kubebuilder:printcolumn:name="Destination Service Name",type="string",JSONPath=".spec.destinationServiceName"
// +kubebuilder:printcolumn:name="Source Endpoint ID",type="string",JSONPath=".spec.sourceEndpointId"
// +kubebuilder:printcolumn:name="Destination Endpoint ID",type="string",JSONPath=".spec.destinationEndpointId"
type ServiceIntegration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceIntegrationSpec   `json:"spec,omitempty"`
	Status ServiceIntegrationStatus `json:"status,omitempty"`
}

func (svcint ServiceIntegration) AuthSecretRef() *AuthSecretReference {
	return svcint.Spec.AuthSecretRef
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
