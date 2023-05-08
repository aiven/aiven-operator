package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
)

func getKafkaYaml(project, name string) string {
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
  disk_space: 600Gib

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

`, project, name)
}

func TestKafka(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	name := randName("kafka")
	yml := getKafkaYaml(testProject, name)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ks := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(ks, name))

	// THEN
	ksAvn, err := avnClient.Services.Get(testProject, name)
	require.NoError(t, err)
	assert.Equal(t, ksAvn.Name, ks.GetName())
	assert.Equal(t, "RUNNING", ks.Status.State)
	assert.Equal(t, ksAvn.State, ks.Status.State)
	assert.Equal(t, ksAvn.Plan, ks.Spec.Plan)
	assert.Equal(t, ksAvn.CloudName, ks.Spec.CloudName)
	assert.Equal(t, "600Gib", ks.Spec.DiskSpace)
	assert.Equal(t, 614400, ksAvn.DiskSpaceMB)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, ks.Spec.Tags)
	ksResp, err := avnClient.ServiceTags.Get(testProject, name)
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

	// New secrets
	assert.NotEmpty(t, secret.Data["KAFKA_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_PORT"])
	assert.NotEmpty(t, secret.Data["KAFKA_USERNAME"])
	assert.NotEmpty(t, secret.Data["KAFKA_PASSWORD"])
	assert.NotEmpty(t, secret.Data["KAFKA_ACCESS_CERT"])
	assert.NotEmpty(t, secret.Data["KAFKA_ACCESS_KEY"])
}
