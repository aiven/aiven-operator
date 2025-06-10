package tests

import (
	"testing"

	"github.com/aiven/go-client-codegen/handler/kafkaconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

func TestKafkaConnector(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	kafkaName := randName("kafka-connector")
	osName := randName("kafka-connector")
	topicName := randName("kafka-connector")
	connectorName := randName("kafka-connector")
	yml, err := loadExampleYaml("kafkaconnector.yaml", map[string]string{
		// Kafka
		"doc[0].metadata.name":  kafkaName,
		"doc[0].spec.project":   cfg.Project,
		"doc[0].spec.cloudName": cfg.PrimaryCloudName,

		// Kafka Topic
		"doc[1].metadata.name":    topicName,
		"doc[1].spec.project":     cfg.Project,
		"doc[1].spec.serviceName": kafkaName,

		// OpenSearch
		"doc[2].metadata.name":  osName,
		"doc[2].spec.project":   cfg.Project,
		"doc[2].spec.cloudName": cfg.PrimaryCloudName,

		// Kafka Connector
		"doc[3].metadata.name":    connectorName,
		"doc[3].spec.project":     cfg.Project,
		"doc[3].spec.serviceName": kafkaName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects

	kafkaService := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(kafkaService, kafkaName))

	osService := new(v1alpha1.OpenSearch)
	require.NoError(t, s.GetRunning(osService, osName))

	kafkaTopic := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(kafkaTopic, topicName))

	kafkaConnector := new(v1alpha1.KafkaConnector)
	require.NoError(t, s.GetRunning(kafkaConnector, connectorName))

	// THEN
	kcAvn, err := controllers.GetKafkaConnectorByName(ctx, avnGen, cfg.Project, kafkaName, connectorName)
	require.NoError(t, err)
	assert.Equal(t, kcAvn.Name, kafkaConnector.GetName())
	assert.Equal(t, kcAvn.Config.ConnectorClass, kafkaConnector.Spec.ConnectorClass)

	// Validates Kafka Connector status
	status, err := avnGen.ServiceKafkaConnectGetConnectorStatus(ctx, cfg.Project, kafkaName, connectorName)
	require.NoError(t, err)
	assert.Equal(t, kafkaconnect.ServiceKafkaConnectConnectorStateTypeRunning, status.State)
}
