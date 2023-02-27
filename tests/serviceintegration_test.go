package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getClickhousePostgreSQLYaml(project, chName, pgName, siName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: ch-token

  project: %[1]s
  cloudName: google-europe-west1
  plan: startup-16
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

---

apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: pg-token

  project: %[1]s
  cloudName: google-europe-west1
  plan: startup-4
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

  userConfig:
    pg_version: "15"

---

apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: %[4]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  integrationType: clickhouse_postgresql
  sourceServiceName: %[3]s
  destinationServiceName: %[2]s

  clickhousePostgresql:
    databases:
      - database: defaultdb
        schema: public
`, project, chName, pgName, siName)
}

func TestServiceIntegrationClickhousePostgreSQL(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	chName := randName("clickhouse-postgresql")
	pgName := randName("clickhouse-postgresql")
	siName := randName("clickhouse-postgresql")

	yml := getClickhousePostgreSQLYaml(testProject, chName, pgName, siName)
	s, err := NewSession(k8sClient, avnClient, testProject, yml)
	require.NoError(t, err)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply())

	// Waits kube objects
	ch := new(v1alpha1.Clickhouse)
	require.NoError(t, s.GetRunning(ch, chName))

	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	si := new(v1alpha1.ServiceIntegration)
	require.NoError(t, s.GetRunning(si, siName))

	// THEN
	// Validates Clickhouse
	chAvn, err := avnClient.Services.Get(testProject, chName)
	require.NoError(t, err)
	assert.Equal(t, chAvn.Name, ch.GetName())
	assert.Equal(t, chAvn.State, ch.Status.State)
	assert.Equal(t, chAvn.Plan, ch.Spec.Plan)
	assert.Equal(t, chAvn.CloudName, ch.Spec.CloudName)
	assert.Equal(t, chAvn.MaintenanceWindow.DayOfWeek, ch.Spec.MaintenanceWindowDow)
	assert.Equal(t, chAvn.MaintenanceWindow.TimeOfDay, ch.Spec.MaintenanceWindowTime)

	// Validates PostgreSQL
	pgAvn, err := avnClient.Services.Get(testProject, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.Name, pg.GetName())
	assert.Equal(t, pgAvn.State, pg.Status.State)
	assert.Equal(t, pgAvn.Plan, pg.Spec.Plan)
	assert.Equal(t, pgAvn.CloudName, pg.Spec.CloudName)
	assert.Equal(t, pgAvn.MaintenanceWindow.DayOfWeek, pg.Spec.MaintenanceWindowDow)
	assert.Equal(t, pgAvn.MaintenanceWindow.TimeOfDay, pg.Spec.MaintenanceWindowTime)
	assert.Equal(t, pgAvn.UserConfig["pg_version"].(string), *pg.Spec.UserConfig.PgVersion)

	// Validates ServiceIntegration
	siAvn, err := avnClient.ServiceIntegrations.Get(testProject, si.Status.ID)
	require.NoError(t, err)
	assert.Equal(t, "clickhouse_postgresql", siAvn.IntegrationType)
	assert.Equal(t, siAvn.IntegrationType, si.Spec.IntegrationType)
	assert.Equal(t, pgName, *siAvn.SourceService)
	assert.Equal(t, chName, *siAvn.DestinationService)
	assert.True(t, siAvn.Active)
	assert.True(t, siAvn.Enabled)
}

func getKafkaLogsYaml(project, ksName, ktName, siName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: google-europe-west1
  plan: business-4

---

apiVersion: aiven.io/v1alpha1
kind: KafkaTopic
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  replication: 2
  partitions: 1

---

apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: %[4]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  integrationType: kafka_logs
  sourceServiceName: %[2]s
  destinationServiceName: %[2]s

  kafkaLogs:
    kafka_topic: %[3]s
`, project, ksName, ktName, siName)
}

func TestServiceIntegrationKafkaLogs(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ksName := randName("kafka-logs")
	ktName := randName("kafka-logs")
	siName := randName("kafka-logs")

	yml := getKafkaLogsYaml(testProject, ksName, ktName, siName)
	s, err := NewSession(k8sClient, avnClient, testProject, yml)
	require.NoError(t, err)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply())

	// Waits kube objects
	ks := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(ks, ksName))

	kt := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(kt, ktName))

	si := new(v1alpha1.ServiceIntegration)
	require.NoError(t, s.GetRunning(si, siName))

	// THEN
	// Validates Kafka
	ksAvn, err := avnClient.Services.Get(testProject, ksName)
	require.NoError(t, err)
	assert.Equal(t, ksAvn.Name, ks.GetName())
	assert.Equal(t, ksAvn.State, ks.Status.State)
	assert.Equal(t, ksAvn.Plan, ks.Spec.Plan)
	assert.Equal(t, ksAvn.CloudName, ks.Spec.CloudName)

	// Validates KafkaTopic
	ktAvn, err := avnClient.KafkaTopics.Get(testProject, ksName, ktName)
	require.NoError(t, err)
	assert.Equal(t, ktAvn.TopicName, kt.GetName())
	assert.Equal(t, ktAvn.State, kt.Status.State)
	assert.Equal(t, ktAvn.Replication, kt.Spec.Replication)
	assert.Len(t, ktAvn.Partitions, kt.Spec.Partitions)

	// Validates ServiceIntegration
	siAvn, err := avnClient.ServiceIntegrations.Get(testProject, si.Status.ID)
	require.NoError(t, err)
	assert.Equal(t, "kafka_logs", siAvn.IntegrationType)
	assert.Equal(t, siAvn.IntegrationType, si.Spec.IntegrationType)
	assert.Equal(t, ksName, *siAvn.SourceService)
	assert.Equal(t, ksName, *siAvn.DestinationService)
	assert.True(t, siAvn.Active)
	assert.True(t, siAvn.Enabled)
	require.NotNil(t, si.Spec.KafkaLogsUserConfig)
	assert.Equal(t, ktName, si.Spec.KafkaLogsUserConfig.KafkaTopic)
}

func getKafkaConnectYaml(project, ksName, kcName, siName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: google-europe-west1
  plan: business-4

---

apiVersion: aiven.io/v1alpha1
kind: KafkaConnect
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: google-europe-west1
  plan: business-4

  userConfig:
    kafka_connect:
      consumer_isolation_level: read_committed
    public_access:
      kafka_connect: true

---

apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: %[4]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  integrationType: kafka_connect
  sourceServiceName: %[2]s
  destinationServiceName: %[3]s

  kafkaConnect:
    kafka_connect:
      group_id: "connect"
      status_storage_topi: "__connect_status"
      offset_storage_topic: "__connect_offsets"
`, project, ksName, kcName, siName)
}

func TestServiceIntegrationKafkaConnect(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ksName := randName("kafka-connect")
	kcName := randName("kafka-connect")
	siName := randName("kafka-connect")

	yml := getKafkaConnectYaml(testProject, ksName, kcName, siName)
	s, err := NewSession(k8sClient, avnClient, testProject, yml)
	require.NoError(t, err)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply())

	// Waits kube objects
	ks := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(ks, ksName))

	kc := new(v1alpha1.KafkaConnect)
	require.NoError(t, s.GetRunning(kc, kcName))

	si := new(v1alpha1.ServiceIntegration)
	require.NoError(t, s.GetRunning(si, siName))

	// THEN
	// Validates Kafka
	ksAvn, err := avnClient.Services.Get(testProject, ksName)
	require.NoError(t, err)
	assert.Equal(t, ksAvn.Name, ks.GetName())
	assert.Equal(t, ksAvn.State, ks.Status.State)
	assert.Equal(t, ksAvn.Plan, ks.Spec.Plan)
	assert.Equal(t, ksAvn.CloudName, ks.Spec.CloudName)

	// Validates KafkaConnect
	kcAvn, err := avnClient.Services.Get(testProject, kcName)
	require.NoError(t, err)
	assert.Equal(t, kcAvn.Name, kc.GetName())
	assert.Equal(t, kcAvn.State, kc.Status.State)
	assert.Equal(t, kcAvn.Plan, kc.Spec.Plan)
	assert.Equal(t, kcAvn.CloudName, kc.Spec.CloudName)
	assert.Equal(t, "read_committed", *kc.Spec.UserConfig.KafkaConnect.ConsumerIsolationLevel)
	assert.True(t, *kc.Spec.UserConfig.PublicAccess.KafkaConnect)

	// Validates ServiceIntegration
	siAvn, err := avnClient.ServiceIntegrations.Get(testProject, si.Status.ID)
	require.NoError(t, err)
	assert.Equal(t, "kafka_connect", siAvn.IntegrationType)
	assert.Equal(t, siAvn.IntegrationType, si.Spec.IntegrationType)
	assert.Equal(t, ksName, *siAvn.SourceService)
	assert.Equal(t, kcName, *siAvn.DestinationService)
	assert.True(t, siAvn.Active)
	assert.True(t, siAvn.Enabled)
	require.NotNil(t, si.Spec.KafkaConnectUserConfig)
	assert.Equal(t, "connect", *si.Spec.KafkaConnectUserConfig.KafkaConnect.GroupId)
}
