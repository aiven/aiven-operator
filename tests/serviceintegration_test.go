//go:build serviceintegration

package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aiven/go-client-codegen/handler/kafkatopic"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	datadoguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/datadog"
)

func TestServiceIntegrationClickhousePostgreSQL(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	chName := randName("clickhouse")
	pgName := randName("postgresql")
	siName := randName("clickhouse-postgresql")

	yml, err := loadExampleYaml("serviceintegration.clickhouse_postgresql.yaml", map[string]string{
		"doc[0].metadata.name":               siName,
		"doc[0].spec.project":                cfg.Project,
		"doc[0].spec.sourceServiceName":      pgName,
		"doc[0].spec.destinationServiceName": chName,

		"doc[1].metadata.name":  chName,
		"doc[1].spec.project":   cfg.Project,
		"doc[1].spec.cloudName": cfg.PrimaryCloudName,

		"doc[2].metadata.name":  pgName,
		"doc[2].spec.project":   cfg.Project,
		"doc[2].spec.cloudName": cfg.PrimaryCloudName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ch := new(v1alpha1.Clickhouse)
	require.NoError(t, s.GetRunning(ch, chName))

	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	si := new(v1alpha1.ServiceIntegration)
	require.NoError(t, s.GetRunning(si, siName))

	// THEN
	// Validates Clickhouse
	chAvn, err := avnGen.ServiceGet(ctx, cfg.Project, chName)
	require.NoError(t, err)
	assert.Equal(t, chAvn.ServiceName, ch.GetName())
	assert.Contains(t, serviceRunningStatesAiven, chAvn.State)
	assert.Equal(t, chAvn.Plan, ch.Spec.Plan)
	assert.Equal(t, chAvn.CloudName, ch.Spec.CloudName)
	assert.EqualValues(t, chAvn.Maintenance.Dow, ch.Spec.MaintenanceWindowDow)
	assert.Equal(t, chAvn.Maintenance.Time, ch.Spec.MaintenanceWindowTime)

	// Validates PostgreSQL
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)
	assert.Equal(t, pgAvn.Plan, pg.Spec.Plan)
	assert.Equal(t, pgAvn.CloudName, pg.Spec.CloudName)
	assert.EqualValues(t, pgAvn.Maintenance.Dow, pg.Spec.MaintenanceWindowDow) // pgAvn.Maintenance.Dow has different type
	assert.Equal(t, pgAvn.Maintenance.Time, pg.Spec.MaintenanceWindowTime)

	// Validates ServiceIntegration
	siAvn, err := avnGen.ServiceIntegrationGet(ctx, cfg.Project, si.Status.ID)
	require.NoError(t, err)
	assert.EqualValues(t, "clickhouse_postgresql", siAvn.IntegrationType)
	assert.Equal(t, siAvn.IntegrationType, si.Spec.IntegrationType)
	assert.Equal(t, pgName, siAvn.SourceService)
	assert.Equal(t, chName, *siAvn.DestService)
	assert.True(t, siAvn.Active)
	assert.True(t, siAvn.Enabled)
}

func TestServiceIntegrationKafkaLogs(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	ksName := randName("kafka-logs")
	ktName := randName("kafka-logs")
	siName := randName("kafka-logs")

	yml, err := loadExampleYaml("serviceintegration.kafka_logs.yaml", map[string]string{
		"doc[0].metadata.name":               siName,
		"doc[0].spec.project":                cfg.Project,
		"doc[0].spec.sourceServiceName":      ksName,
		"doc[0].spec.destinationServiceName": ksName,
		"doc[0].spec.kafkaLogs.kafka_topic":  ktName,

		"doc[1].metadata.name":  ksName,
		"doc[1].spec.project":   cfg.Project,
		"doc[1].spec.cloudName": cfg.PrimaryCloudName,

		"doc[2].metadata.name":    ktName,
		"doc[2].spec.project":     cfg.Project,
		"doc[2].spec.serviceName": ksName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ks := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(ks, ksName))

	kt := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(kt, ktName))

	si := new(v1alpha1.ServiceIntegration)
	require.NoError(t, s.GetRunning(si, siName))

	// THEN
	// Validates Kafka
	ksAvn, err := avnGen.ServiceGet(ctx, cfg.Project, ksName)
	require.NoError(t, err)
	assert.Equal(t, ksAvn.ServiceName, ks.GetName())
	assert.Contains(t, serviceRunningStatesAiven, ksAvn.State)
	assert.Equal(t, ksAvn.Plan, ks.Spec.Plan)
	assert.Equal(t, ksAvn.CloudName, ks.Spec.CloudName)

	// Validates KafkaTopic
	var ktAvn *kafkatopic.ServiceKafkaTopicGetOut
	// Kafka topics are eventually consistent in Aiven API, so we poll until they become readable
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		var getErr error
		ktAvn, getErr = avnGen.ServiceKafkaTopicGet(ctx, cfg.Project, ksName, ktName)
		assert.NoError(collect, getErr)
	}, 2*time.Minute, 10*time.Second)

	assert.Equal(t, ktAvn.TopicName, kt.GetName())
	assert.Equal(t, ktAvn.State, kt.Status.State)
	assert.Equal(t, ktAvn.Replication, kt.Spec.Replication)
	assert.Len(t, ktAvn.Partitions, kt.Spec.Partitions)

	// Validates ServiceIntegration
	siAvn, err := avnGen.ServiceIntegrationGet(ctx, cfg.Project, si.Status.ID)
	require.NoError(t, err)
	assert.EqualValues(t, "kafka_logs", siAvn.IntegrationType)
	assert.Equal(t, siAvn.IntegrationType, si.Spec.IntegrationType)
	assert.Equal(t, ksName, siAvn.SourceService)
	assert.Equal(t, ksName, *siAvn.DestService)
	assert.True(t, siAvn.Active)
	assert.True(t, siAvn.Enabled)
	require.NotNil(t, si.Spec.KafkaLogsUserConfig)
	assert.Equal(t, ktName, si.Spec.KafkaLogsUserConfig.KafkaTopic)
}

func TestServiceIntegrationKafkaConnect(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	ksName := randName("kafka-connect")
	kcName := randName("kafka-connect")
	siName := randName("kafka-connect")

	yml, err := loadExampleYaml("serviceintegration.kafka_connect.yaml", map[string]string{
		"doc[0].metadata.name":               siName,
		"doc[0].spec.project":                cfg.Project,
		"doc[0].spec.sourceServiceName":      ksName,
		"doc[0].spec.destinationServiceName": kcName,

		"doc[1].metadata.name":  ksName,
		"doc[1].spec.project":   cfg.Project,
		"doc[1].spec.cloudName": cfg.PrimaryCloudName,

		"doc[2].metadata.name":  kcName,
		"doc[2].spec.project":   cfg.Project,
		"doc[2].spec.cloudName": cfg.PrimaryCloudName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ks := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(ks, ksName))

	kc := new(v1alpha1.KafkaConnect)
	require.NoError(t, s.GetRunning(kc, kcName))

	si := new(v1alpha1.ServiceIntegration)
	require.NoError(t, s.GetRunning(si, siName))

	// THEN
	// Validates Kafka
	ksAvn, err := avnGen.ServiceGet(ctx, cfg.Project, ksName)
	require.NoError(t, err)
	assert.Equal(t, ksAvn.ServiceName, ks.GetName())
	assert.Contains(t, serviceRunningStatesAiven, ksAvn.State)
	assert.Equal(t, ksAvn.Plan, ks.Spec.Plan)
	assert.Equal(t, ksAvn.CloudName, ks.Spec.CloudName)

	// Validates KafkaConnect
	kcAvn, err := avnGen.ServiceGet(ctx, cfg.Project, kcName)
	require.NoError(t, err)
	assert.Equal(t, kcAvn.ServiceName, kc.GetName())
	assert.Contains(t, serviceRunningStatesAiven, kcAvn.State)
	assert.Equal(t, kcAvn.Plan, kc.Spec.Plan)
	assert.Equal(t, kcAvn.CloudName, kc.Spec.CloudName)
	assert.Equal(t, "read_committed", *kc.Spec.UserConfig.KafkaConnect.ConsumerIsolationLevel)
	assert.True(t, *kc.Spec.UserConfig.PublicAccess.KafkaConnect)

	// Validates ServiceIntegration
	siAvn, err := avnGen.ServiceIntegrationGet(ctx, cfg.Project, si.Status.ID)
	require.NoError(t, err)
	assert.EqualValues(t, "kafka_connect", siAvn.IntegrationType)
	assert.Equal(t, siAvn.IntegrationType, si.Spec.IntegrationType)
	assert.Equal(t, ksName, siAvn.SourceService)
	assert.Equal(t, kcName, *siAvn.DestService)
	assert.True(t, siAvn.Active)
	assert.True(t, siAvn.Enabled)
	require.NotNil(t, si.Spec.KafkaConnectUserConfig)
	assert.Equal(t, "connect", *si.Spec.KafkaConnectUserConfig.KafkaConnect.GroupId)
	assert.Equal(t, "__connect_status", *si.Spec.KafkaConnectUserConfig.KafkaConnect.StatusStorageTopic)
	assert.Equal(t, "__connect_offsets", *si.Spec.KafkaConnectUserConfig.KafkaConnect.OffsetStorageTopic)
}

func TestServiceIntegrationAutoscaler(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	pg, releasePostgreSQL, err := sharedResources.AcquirePostgreSQL(ctx)
	require.NoError(t, err)
	defer releasePostgreSQL()

	pgName := pg.Name

	endpointName := randName("autoscaler-endpoint")
	endpointDisplayName := randName("autoscaler")

	endpointYml, err := loadExampleYaml("serviceintegrationendpoint.autoscaler.yaml", map[string]string{
		"metadata.name":     endpointName,
		"spec.project":      cfg.Project,
		"spec.endpointName": endpointDisplayName,
	})
	require.NoError(t, err)

	siName := randName("autoscaler")

	sEndpoint := NewSession(ctx, k8sClient)
	defer sEndpoint.Destroy(t)

	require.NoError(t, sEndpoint.Apply(endpointYml))

	endpoint := new(v1alpha1.ServiceIntegrationEndpoint)
	require.NoError(t, sEndpoint.GetRunning(endpoint, endpointName))
	require.NotEmpty(t, endpoint.Status.ID)
	endpointID := endpoint.Status.ID

	yml, err := loadExampleYaml("serviceintegration.autoscaler.yaml", map[string]string{
		"doc[0].metadata.name":              siName,
		"doc[0].spec.project":               cfg.Project,
		"doc[0].spec.sourceServiceName":     pgName,
		"doc[0].spec.destinationEndpointId": endpointID,
		"doc[1]":                            "REMOVE",
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	si := new(v1alpha1.ServiceIntegration)
	require.NoError(t, s.GetRunning(si, siName))

	// THEN
	// Validates PostgreSQL
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pgName)
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)

	// Validates ServiceIntegration
	siAvn, err := avnGen.ServiceIntegrationGet(ctx, cfg.Project, si.Status.ID)
	require.NoError(t, err)
	assert.EqualValues(t, "autoscaler", siAvn.IntegrationType)
	assert.Equal(t, siAvn.IntegrationType, si.Spec.IntegrationType)
	assert.Equal(t, pgName, siAvn.SourceService)
	assert.Equal(t, endpointID, *siAvn.DestEndpointId)
	assert.True(t, siAvn.Active)
	assert.True(t, siAvn.Enabled)
}

// todo: refactor when ServiceIntegrationEndpoint released
func TestServiceIntegrationDatadog(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	endpointID := os.Getenv("DATADOG_ENDPOINT_ID")
	if endpointID == "" {
		t.Skip("Provide DATADOG_ENDPOINT_ID for this test")
	}

	// GIVEN
	ctx, cancel := testCtx()

	defer cancel()
	pgName := randName("datadog")
	siName := randName("datadog")

	yml, err := loadExampleYaml("serviceintegration.datadog.yaml", map[string]string{
		"doc[0].metadata.name":              siName,
		"doc[0].spec.project":               cfg.Project,
		"doc[0].spec.sourceServiceName":     pgName,
		"doc[0].spec.destinationEndpointId": endpointID,

		"doc[1].metadata.name":  pgName,
		"doc[1].spec.project":   cfg.Project,
		"doc[1].spec.cloudName": cfg.PrimaryCloudName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	si := new(v1alpha1.ServiceIntegration)
	require.NoError(t, s.GetRunning(si, siName))

	// THEN
	// Validates PostgreSQL
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)
	assert.Equal(t, pgAvn.Plan, pg.Spec.Plan)

	// Validates Datadog
	siAvn, err := avnGen.ServiceIntegrationGet(ctx, cfg.Project, si.Status.ID)
	require.NoError(t, err)
	assert.EqualValues(t, "datadog", siAvn.IntegrationType)
	assert.Equal(t, siAvn.IntegrationType, si.Spec.IntegrationType)
	assert.True(t, siAvn.Active)
	assert.True(t, siAvn.Enabled)

	// Tests user config
	require.NotNil(t, si.Spec.DatadogUserConfig)
	assert.Equal(t, anyPointer(true), si.Spec.DatadogUserConfig.DatadogDbmEnabled)
	expectedTags := []*datadoguserconfig.DatadogTags{
		{Tag: "env", Comment: anyPointer("test")},
	}
	assert.Equal(t, expectedTags, si.Spec.DatadogUserConfig.DatadogTags)
}

func getWebhookMultipleUserConfigsDeniedYaml(project, siName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  integrationType: datadog
  sourceServiceName: whatever

  datadog:
    datadog_dbm_enabled: True

  clickhousePostgresql:
    databases:
      - database: defaultdb
        schema: public
`, project, siName)
}

// TestWebhookMultipleUserConfigsDenied tests v1alpha1.ServiceIntegration.ValidateCreate()
func TestWebhookMultipleUserConfigsDenied(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	// GIVEN
	siName := randName("datadog")
	yml := getWebhookMultipleUserConfigsDeniedYaml(cfg.Project, siName)

	// WHEN
	s := NewSession(ctx, k8sClient)

	// THEN
	err := s.Apply(yml)
	errStringExpected := `admission webhook ` +
		`"vserviceintegration.kb.io" denied the request: ` +
		`got additional configuration for integration type "clickhouse_postgresql"`
	assert.EqualError(t, err, errStringExpected)
}

// TestServiceIntegrationAdoptExisting tests that the operator adopts an existing integration
// when a K8s resource is created with matching configuration.
func TestServiceIntegrationAdoptExisting(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx := context.Background()

	ch, releaseClickhouse, err := sharedResources.AcquireClickhouse(ctx)
	require.NoError(t, err)
	defer releaseClickhouse()

	pg, releasePostgreSQL, err := sharedResources.AcquirePostgreSQL(ctx)
	require.NoError(t, err)
	defer releasePostgreSQL()

	chName := ch.Name
	pgName := pg.Name

	s := NewSession(context.Background(), k8sClient)
	defer s.Destroy(t)

	t.Run("adopts matching integration", func(t *testing.T) {
		subCtx, subCancel := testCtx()
		defer subCancel()

		siName := randName("si-adopt")

		// create the integration via API first
		integrationOut, err := avnGen.ServiceIntegrationCreate(subCtx, cfg.Project, &service.ServiceIntegrationCreateIn{
			IntegrationType: "clickhouse_postgresql",
			SourceService:   &pgName,
			DestService:     &chName,
		})
		require.NoError(t, err)
		require.NotEmpty(t, integrationOut.ServiceIntegrationId)
		existingIntegrationID := integrationOut.ServiceIntegrationId

		// cleanup integration
		defer func() {
			_ = avnGen.ServiceIntegrationDelete(subCtx, cfg.Project, existingIntegrationID)
		}()

		// create the K8s ServiceIntegration resource with the same configuration
		integrationYml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  integrationType: clickhouse_postgresql
  sourceServiceName: %s
  destinationServiceName: %s
  clickhousePostgresql:
    databases:
      - database: defaultdb
        schema: public
`, siName, cfg.Project, pgName, chName)

		require.NoError(t, s.Apply(integrationYml))

		si := new(v1alpha1.ServiceIntegration)
		require.NoError(t, s.GetRunning(si, siName))

		assert.Equal(t, existingIntegrationID, si.Status.ID, "operator should adopt the existing integration")

		siAvn, err := avnGen.ServiceIntegrationGet(subCtx, cfg.Project, si.Status.ID)
		require.NoError(t, err)
		assert.EqualValues(t, "clickhouse_postgresql", siAvn.IntegrationType)
		assert.Equal(t, pgName, siAvn.SourceService)
		assert.Equal(t, chName, *siAvn.DestService)
		assert.True(t, siAvn.Active)
		assert.True(t, siAvn.Enabled)

		// check that no duplicate integration was created
		pgService, err := avnGen.ServiceGet(subCtx, cfg.Project, pgName)
		require.NoError(t, err)

		clickhousePostgresqlCount := 0
		for _, integration := range pgService.ServiceIntegrations {
			if integration.IntegrationType == "clickhouse_postgresql" &&
				integration.DestService != nil &&
				*integration.DestService == chName {
				clickhousePostgresqlCount++
			}
		}
		assert.Equal(t, 1, clickhousePostgresqlCount, "should have exactly one integration, not duplicates")
	})

	t.Run("adopts with explicit project fields", func(t *testing.T) {
		subCtx, subCancel := testCtx()
		defer subCancel()

		siName := randName("si-explicit")

		integrationOut, err := avnGen.ServiceIntegrationCreate(subCtx, cfg.Project, &service.ServiceIntegrationCreateIn{
			IntegrationType: "clickhouse_postgresql",
			SourceService:   &pgName,
			SourceProject:   &cfg.Project,
			DestService:     &chName,
			DestProject:     &cfg.Project,
		})
		require.NoError(t, err)
		existingIntegrationID := integrationOut.ServiceIntegrationId

		defer func() {
			_ = avnGen.ServiceIntegrationDelete(subCtx, cfg.Project, existingIntegrationID)
		}()

		integrationYml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  integrationType: clickhouse_postgresql
  sourceServiceName: %s
  sourceProjectName: %s
  destinationServiceName: %s
  destinationProjectName: %s
  clickhousePostgresql:
    databases:
      - database: defaultdb
        schema: public
`, siName, cfg.Project, pgName, cfg.Project, chName, cfg.Project)

		require.NoError(t, s.Apply(integrationYml))

		si := new(v1alpha1.ServiceIntegration)
		require.NoError(t, s.GetRunning(si, siName))

		assert.Equal(t, existingIntegrationID, si.Status.ID, "should adopt with explicit project fields")
	})

	t.Run("updates userConfig after adoption", func(t *testing.T) {
		subCtx, subCancel := testCtx()
		defer subCancel()

		siName := randName("si-userconfig")

		// integration WITHOUT userConfig
		integrationOut, err := avnGen.ServiceIntegrationCreate(subCtx, cfg.Project, &service.ServiceIntegrationCreateIn{
			IntegrationType: "clickhouse_postgresql",
			SourceService:   &pgName,
			DestService:     &chName,
		})
		require.NoError(t, err)
		existingIntegrationID := integrationOut.ServiceIntegrationId

		defer func() {
			_ = avnGen.ServiceIntegrationDelete(subCtx, cfg.Project, existingIntegrationID)
		}()

		// K8s resource WITH userConfig
		integrationYml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  integrationType: clickhouse_postgresql
  sourceServiceName: %s
  destinationServiceName: %s
  clickhousePostgresql:
    databases:
      - database: defaultdb
        schema: public
`, siName, cfg.Project, pgName, chName)

		require.NoError(t, s.Apply(integrationYml))

		si := new(v1alpha1.ServiceIntegration)
		require.NoError(t, s.GetRunning(si, siName))

		assert.Equal(t, existingIntegrationID, si.Status.ID, "should adopt existing integration")

		siAvn, err := avnGen.ServiceIntegrationGet(subCtx, cfg.Project, si.Status.ID)
		require.NoError(t, err)
		assert.True(t, siAvn.Active, "integration should be active")
		assert.True(t, siAvn.Enabled, "integration should be enabled")
	})
}
