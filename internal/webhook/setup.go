package webhook

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhooks(mgr ctrl.Manager) error {
	if err := SetupProjectWebhook(mgr); err != nil {
		return fmt.Errorf("webhook Project: %w", err)
	}
	if err := SetupPostgreSQLWebhook(mgr); err != nil {
		return fmt.Errorf("webhook PostgreSQL: %w", err)
	}
	if err := SetupDatabaseWebhook(mgr); err != nil {
		return fmt.Errorf("webhook Database: %w", err)
	}
	if err := SetupFlinkWebhook(mgr); err != nil {
		return fmt.Errorf("webhook Flink: %w", err)
	}
	if err := SetupConnectionPoolWebhook(mgr); err != nil {
		return fmt.Errorf("webhook ConnectionPool: %w", err)
	}
	if err := SetupServiceUserWebhook(mgr); err != nil {
		return fmt.Errorf("webhook ServiceUser: %w", err)
	}
	if err := SetupKafkaWebhook(mgr); err != nil {
		return fmt.Errorf("webhook Kafka: %w", err)
	}
	if err := SetupKafkaConnectWebhook(mgr); err != nil {
		return fmt.Errorf("webhook KafkaConnect: %w", err)
	}
	if err := SetupKafkaTopicWebhook(mgr); err != nil {
		return fmt.Errorf("webhook KafkaTopic: %w", err)
	}
	if err := SetupKafkaACLWebhook(mgr); err != nil {
		return fmt.Errorf("webhook KafkaACL: %w", err)
	}
	if err := SetupKafkaSchemaWebhook(mgr); err != nil {
		return fmt.Errorf("webhook KafkaSchema: %w", err)
	}
	if err := SetupServiceIntegrationWebhook(mgr); err != nil {
		return fmt.Errorf("webhook ServiceIntegration: %w", err)
	}
	if err := SetupServiceIntegrationEndpointWebhook(mgr); err != nil {
		return fmt.Errorf("webhook ServiceIntegrationEndpoint: %w", err)
	}
	if err := SetupKafkaConnectorWebhook(mgr); err != nil {
		return fmt.Errorf("webhook KafkaConnector: %w", err)
	}
	if err := SetupOpenSearchWebhook(mgr); err != nil {
		return fmt.Errorf("webhook OpenSearch: %w", err)
	}
	if err := SetupClickhouseWebhook(mgr); err != nil {
		return fmt.Errorf("webhook Clickhouse: %w", err)
	}
	if err := SetupMySQLWebhook(mgr); err != nil {
		return fmt.Errorf("webhook MySQL: %w", err)
	}
	if err := SetupGrafanaWebhook(mgr); err != nil {
		return fmt.Errorf("webhook Grafana: %w", err)
	}
	if err := SetupValkeyWebhook(mgr); err != nil {
		return fmt.Errorf("webhook Valkey: %w", err)
	}

	//+kubebuilder:scaffold:builder
	return nil
}
