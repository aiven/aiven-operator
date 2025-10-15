// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"fmt"
	"reflect"

	"github.com/aiven/go-client-codegen/handler/service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalerintegration "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/autoscaler"
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
	ProjectDependant `json:",inline"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:Enum=alertmanager;autoscaler;caching;cassandra_cross_service_cluster;clickhouse_kafka;clickhouse_postgresql;dashboard;datadog;datasource;external_aws_cloudwatch_logs;external_aws_cloudwatch_metrics;external_elasticsearch_logs;external_google_cloud_logging;external_opensearch_logs;flink;flink_external_kafka;flink_external_postgresql;internal_connectivity;jolokia;kafka_connect;kafka_logs;kafka_mirrormaker;logs;m3aggregator;m3coordinator;metrics;opensearch_cross_cluster_replication;opensearch_cross_cluster_search;prometheus;read_replica;rsyslog;schema_registry_proxy;stresstester;thanosquery;thanosstore;vmalert
	// Type of the service integration accepted by Aiven API. Some values may not be supported by the operator
	IntegrationType service.IntegrationType `json:"integrationType"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:MaxLength=36
	// Source endpoint for the integration (if any)
	SourceEndpointID string `json:"sourceEndpointID,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:MaxLength=64
	// Source service for the integration (if any)
	SourceServiceName string `json:"sourceServiceName,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:MaxLength=63
	// Source project for the integration (if any)
	SourceProjectName string `json:"sourceProjectName,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:MaxLength=36
	// Destination endpoint for the integration (if any)
	DestinationEndpointID string `json:"destinationEndpointId,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:MaxLength=64
	// Destination service for the integration (if any)
	DestinationServiceName string `json:"destinationServiceName,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:MaxLength=63
	// Destination project for the integration (if any)
	DestinationProjectName string `json:"destinationProjectName,omitempty"`

	// Autoscaler specific user configuration options
	AutoscalerUserConfig *autoscalerintegration.AutoscalerUserConfig `json:"autoscaler,omitempty"`

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
	ExternalAWSCloudwatchMetricsUserConfig *externalawscloudwatchmetricsuserconfig.ExternalAwsCloudwatchMetricsUserConfig `json:"externalAWSCloudwatchMetrics,omitempty"`
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

// ServiceIntegration is the Schema for the serviceintegrations API.
//
// info "Adoption of existing integrations": If a ServiceIntegration resource is created with configuration matching an existing Aiven integration (created outside the operator), the operator will adopt the existing integration.
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

var _ AivenManagedObject = &ServiceIntegration{}

func (*ServiceIntegration) NoSecret() bool {
	return true
}

func (in *ServiceIntegration) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *ServiceIntegration) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *ServiceIntegration) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *ServiceIntegration) getUserConfigFields() map[service.IntegrationType]any {
	return map[service.IntegrationType]any{
		service.IntegrationTypeAutoscaler:                   in.Spec.AutoscalerUserConfig,
		service.IntegrationTypeClickhouseKafka:              in.Spec.ClickhouseKafkaUserConfig,
		service.IntegrationTypeClickhousePostgresql:         in.Spec.ClickhousePostgreSQLUserConfig,
		service.IntegrationTypeDatadog:                      in.Spec.DatadogUserConfig,
		service.IntegrationTypeExternalAwsCloudwatchMetrics: in.Spec.ExternalAWSCloudwatchMetricsUserConfig,
		service.IntegrationTypeKafkaConnect:                 in.Spec.KafkaConnectUserConfig,
		service.IntegrationTypeKafkaLogs:                    in.Spec.KafkaLogsUserConfig,
		service.IntegrationTypeKafkaMirrormaker:             in.Spec.KafkaMirrormakerUserConfig,
		service.IntegrationTypeLogs:                         in.Spec.LogsUserConfig,
		service.IntegrationTypeMetrics:                      in.Spec.MetricsUserConfig,
	}
}

func (in *ServiceIntegration) HasUserConfig() bool {
	_, ok := in.getUserConfigFields()[in.Spec.IntegrationType]
	return ok
}

func (in *ServiceIntegration) GetUserConfig() (any, error) {
	configs := in.getUserConfigFields()
	thisType := in.Spec.IntegrationType

	// Checks if it is the only configuration set
	for k, v := range configs {
		if k != thisType && !reflect.ValueOf(v).IsNil() {
			return nil, fmt.Errorf("got additional configuration for integration type %q", k)
		}
	}

	return configs[thisType], nil
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
