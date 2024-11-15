package v1alpha1

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWebhooks(mgr ctrl.Manager) error {
	if err := (&Project{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook Project: %w", err)
	}
	if err := (&AlloyDBOmni{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook AlloyDBOmni: %w", err)
	}
	if err := (&PostgreSQL{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook PostgreSQL: %w", err)
	}
	if err := (&Database{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook Database: %w", err)
	}
	if err := (&Flink{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook Flink: %w", err)
	}
	if err := (&ConnectionPool{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook ConnectionPool: %w", err)
	}
	if err := (&ServiceUser{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook ServiceUser: %w", err)
	}
	if err := (&Kafka{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook Kafka: %w", err)
	}
	if err := (&KafkaConnect{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook KafkaConnect: %w", err)
	}
	if err := (&KafkaTopic{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook KafkaTopic: %w", err)
	}
	if err := (&KafkaACL{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook KafkaACL: %w", err)
	}
	if err := (&KafkaSchema{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook KafkaSchema: %w", err)
	}
	if err := (&ServiceIntegration{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook ServiceIntegration: %w", err)
	}
	if err := (&ServiceIntegrationEndpoint{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook ServiceIntegrationEndpoint: %w", err)
	}
	if err := (&KafkaConnector{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook KafkaConnector: %w", err)
	}
	if err := (&Redis{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook Redis: %w", err)
	}
	if err := (&OpenSearch{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook OpenSearch: %w", err)
	}
	if err := (&Clickhouse{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook Clickhouse: %w", err)
	}
	if err := (&MySQL{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook MySQL: %w", err)
	}
	if err := (&Cassandra{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook Cassandra: %w", err)
	}
	if err := (&Grafana{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook Grafana: %w", err)
	}
	if err := (&Valkey{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("webhook Valkey: %w", err)
	}

	//+kubebuilder:scaffold:builder
	return nil
}
