package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
)

func getKafkaYaml(project, name, cloudName string) string {
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
  cloudName: %[3]s
  plan: business-4
  disk_space: 600GiB

  tags:
    env: test
    instance: foo

  userConfig:
    kafka_rest: true
    kafka_connect: true
    schema_registry: true
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16
    kafka_authentication_methods:
      sasl: true
`, project, name, cloudName)
}

func TestKafka(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("kafka")
	yml := getKafkaYaml(cfg.Project, name, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ks := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(ks, name))

	// THEN
	ksAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, ksAvn.ServiceName, ks.GetName())
	assert.Equal(t, serviceRunningState, ks.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, ksAvn.State)
	assert.Equal(t, ksAvn.Plan, ks.Spec.Plan)
	assert.Equal(t, ksAvn.CloudName, ks.Spec.CloudName)
	assert.Equal(t, "600GiB", ks.Spec.DiskSpace)
	assert.Equal(t, float64(614400), *ksAvn.DiskSpaceMb)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, ks.Spec.Tags)
	ksResp, err := avnClient.ServiceTags.Get(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, ksResp.Tags, ks.Spec.Tags)

	// UserConfig test
	require.NotNil(t, ks.Spec.UserConfig)
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.KafkaRest)
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.KafkaConnect)
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.SchemaRegistry)

	// Validates ip filters
	require.Len(t, ks.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", ks.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *ks.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", ks.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, ks.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*kafkauserconfig.IpFilter
	require.NoError(t, castInterface(ksAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, ks.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(ks.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USERNAME"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["ACCESS_CERT"])
	assert.NotEmpty(t, secret.Data["ACCESS_KEY"])
	assert.NotEmpty(t, secret.Data["CA_CERT"])

	// New secrets
	assert.NotEmpty(t, secret.Data["KAFKA_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_PORT"])
	assert.NotEmpty(t, secret.Data["KAFKA_USERNAME"])
	assert.NotEmpty(t, secret.Data["KAFKA_PASSWORD"])
	assert.NotEmpty(t, secret.Data["KAFKA_ACCESS_CERT"])
	assert.NotEmpty(t, secret.Data["KAFKA_ACCESS_KEY"])
	assert.NotEmpty(t, secret.Data["KAFKA_CA_CERT"])

	// SASL test
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.KafkaAuthenticationMethods.Sasl)
	assert.NotEmpty(t, secret.Data["KAFKA_SASL_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_SASL_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_SASL_PORT"], secret.Data["KAFKA_PORT"])

	// Schema registry test
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.SchemaRegistry)
	assert.NotEmpty(t, secret.Data["KAFKA_SCHEMA_REGISTRY_URI"])
	assert.NotEmpty(t, secret.Data["KAFKA_SCHEMA_REGISTRY_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_SCHEMA_REGISTRY_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_SCHEMA_REGISTRY_PORT"], secret.Data["KAFKA_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_SCHEMA_REGISTRY_PORT"], secret.Data["KAFKA_SASL_PORT"])

	// Kafka Connect test
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.KafkaConnect)
	assert.NotEmpty(t, secret.Data["KAFKA_CONNECT_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_CONNECT_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_CONNECT_PORT"], secret.Data["KAFKA_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_CONNECT_PORT"], secret.Data["KAFKA_SASL_PORT"])

	// Kafka REST test
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.KafkaRest)
	assert.NotEmpty(t, secret.Data["KAFKA_REST_URI"])
	assert.NotEmpty(t, secret.Data["KAFKA_REST_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_REST_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_REST_PORT"], secret.Data["KAFKA_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_REST_PORT"], secret.Data["KAFKA_SASL_PORT"])
}
