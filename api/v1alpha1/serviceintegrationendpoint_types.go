// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscaleruserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/autoscaler"
	datadoguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/datadog"
	externalawscloudwatchlogsuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/external_aws_cloudwatch_logs"
	externalawscloudwatchmetricsuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/external_aws_cloudwatch_metrics"
	externalelasticsearchlogsuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/external_elasticsearch_logs"
	externalgooglecloudbigqueryuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/external_google_cloud_bigquery"
	externalgooglecloudlogginguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/external_google_cloud_logging"
	externalkafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/external_kafka"
	externalopensearchlogsuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/external_opensearch_logs"
	externalpostgresqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/external_postgresql"
	externalschemaregistryuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/external_schema_registry"
	jolokiauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/jolokia"
	prometheususerconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/prometheus"
	rsysloguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/rsyslog"
)

// ServiceIntegrationEndpointSpec defines the desired state of ServiceIntegrationEndpoint
type ServiceIntegrationEndpointSpec struct {
	ProjectDependant `json:",inline"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:Enum=autoscaler;datadog;external_aws_cloudwatch_logs;external_aws_cloudwatch_metrics;external_aws_s3;external_clickhouse;external_elasticsearch_logs;external_google_cloud_bigquery;external_google_cloud_logging;external_kafka;external_mysql;external_opensearch_logs;external_postgresql;external_redis;external_schema_registry;external_sumologic_logs;jolokia;prometheus;rsyslog
	// Type of the service integration endpoint
	EndpointType string `json:"endpointType"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// +kubebuilder:validation:MaxLength=36
	// Source endpoint for the integration (if any)
	EndpointName string `json:"endpointName,omitempty"`

	// Autoscaler configuration values
	Autoscaler *autoscaleruserconfig.AutoscalerUserConfig `json:"autoscaler,omitempty"`

	// Datadog configuration values
	Datadog *datadoguserconfig.DatadogUserConfig `json:"datadog,omitempty"`

	// ExternalAwsCloudwatchLogs configuration values
	ExternalAwsCloudwatchLogs *externalawscloudwatchlogsuserconfig.ExternalAwsCloudwatchLogsUserConfig `json:"externalAWSCloudwatchLogs,omitempty"`

	// ExternalAwsCloudwatchMetrics configuration values
	ExternalAwsCloudwatchMetrics *externalawscloudwatchmetricsuserconfig.ExternalAwsCloudwatchMetricsUserConfig `json:"externalAWSCloudwatchMetrics,omitempty"`

	// ExternalElasticsearchLogs configuration values
	ExternalElasticsearchLogs *externalelasticsearchlogsuserconfig.ExternalElasticsearchLogsUserConfig `json:"externalElasticsearchLogs,omitempty"`

	// ExternalGoogleCloudBigquery configuration values
	ExternalGoogleCloudBigquery *externalgooglecloudbigqueryuserconfig.ExternalGoogleCloudBigqueryUserConfig `json:"externalGoogleCloudBigquery,omitempty"`

	// ExternalGoogleCloudLogging configuration values
	ExternalGoogleCloudLogging *externalgooglecloudlogginguserconfig.ExternalGoogleCloudLoggingUserConfig `json:"externalGoogleCloudLogging,omitempty"`

	// ExternalKafka configuration values
	ExternalKafka *externalkafkauserconfig.ExternalKafkaUserConfig `json:"externalKafka,omitempty"`

	// ExternalOpensearchLogs configuration values
	ExternalOpensearchLogs *externalopensearchlogsuserconfig.ExternalOpensearchLogsUserConfig `json:"externalOpensearchLogs,omitempty"`

	// ExternalPostgresql configuration values
	ExternalPostgresql *externalpostgresqluserconfig.ExternalPostgresqlUserConfig `json:"externalPostgresql,omitempty"`

	// ExternalSchemaRegistry configuration values
	ExternalSchemaRegistry *externalschemaregistryuserconfig.ExternalSchemaRegistryUserConfig `json:"externalSchemaRegistry,omitempty"`

	// Jolokia configuration values
	Jolokia *jolokiauserconfig.JolokiaUserConfig `json:"jolokia,omitempty"`

	// Prometheus configuration values
	Prometheus *prometheususerconfig.PrometheusUserConfig `json:"prometheus,omitempty"`

	// Rsyslog configuration values
	Rsyslog *rsysloguserconfig.RsyslogUserConfig `json:"rsyslog,omitempty"`
}

// ServiceIntegrationEndpointStatus defines the observed state of ServiceIntegrationEndpoint
type ServiceIntegrationEndpointStatus struct {
	// Conditions represent the latest available observations of an ServiceIntegrationEndpoint state
	Conditions []metav1.Condition `json:"conditions"`

	// Service integration ID
	ID string `json:"id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceIntegrationEndpoint is the Schema for the serviceintegrationendpoints API
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Endpoint Name",type="string",JSONPath=".spec.endpointName"
// +kubebuilder:printcolumn:name="Endpoint Type",type="string",JSONPath=".spec.endpointType"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.id"
type ServiceIntegrationEndpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceIntegrationEndpointSpec   `json:"spec,omitempty"`
	Status ServiceIntegrationEndpointStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &ServiceIntegrationEndpoint{}

func (*ServiceIntegrationEndpoint) NoSecret() bool {
	return true
}

func (in *ServiceIntegrationEndpoint) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *ServiceIntegrationEndpoint) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *ServiceIntegrationEndpoint) getUserConfigFields() map[string]any {
	return map[string]any{
		"autoscaler":                      in.Spec.Autoscaler,
		"datadog":                         in.Spec.Datadog,
		"external_aws_cloudwatch_logs":    in.Spec.ExternalAwsCloudwatchLogs,
		"external_aws_cloudwatch_metrics": in.Spec.ExternalAwsCloudwatchMetrics,
		"external_elasticsearch_logs":     in.Spec.ExternalElasticsearchLogs,
		"external_google_cloud_bigquery":  in.Spec.ExternalGoogleCloudBigquery,
		"external_google_cloud_logging":   in.Spec.ExternalGoogleCloudLogging,
		"external_kafka":                  in.Spec.ExternalKafka,
		"external_opensearch_logs":        in.Spec.ExternalOpensearchLogs,
		"external_postgresql":             in.Spec.ExternalPostgresql,
		"external_schema_registry":        in.Spec.ExternalSchemaRegistry,
		"jolokia":                         in.Spec.Jolokia,
		"prometheus":                      in.Spec.Prometheus,
		"rsyslog":                         in.Spec.Rsyslog,
	}
}

func (in *ServiceIntegrationEndpoint) HasUserConfig() bool {
	_, ok := in.getUserConfigFields()[in.Spec.EndpointType]
	return ok
}

func (in *ServiceIntegrationEndpoint) GetUserConfig() (any, error) {
	configs := in.getUserConfigFields()
	thisType := in.Spec.EndpointType

	// Checks if it is the only configuration set
	for k, v := range configs {
		if k != thisType && !reflect.ValueOf(v).IsNil() {
			return nil, fmt.Errorf("got additional configuration for integration endpoint type %q", k)
		}
	}

	return configs[thisType], nil
}

// +kubebuilder:object:root=true

// ServiceIntegrationEndpointList contains a list of ServiceIntegrationEndpoint
type ServiceIntegrationEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceIntegrationEndpoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceIntegrationEndpoint{}, &ServiceIntegrationEndpointList{})
}
