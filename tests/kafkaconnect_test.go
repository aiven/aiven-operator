package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	kafkaconnectuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka_connect"
)

func getKafkaConnectYaml(project, name, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaConnect
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  tags:
    env: test
    instance: foo

  project: %[1]s
  cloudName: %[3]s
  plan: business-4
  
  userConfig:
    kafka_connect:
      consumer_isolation_level: read_committed
    public_access:
      kafka_connect: true
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16
`, project, name, cloudName)
}

func TestKafkaConnect(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("kafka-connect")
	yml := getKafkaConnectYaml(cfg.Project, name, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	kc := new(v1alpha1.KafkaConnect)
	require.NoError(t, s.GetRunning(kc, name))

	// THEN
	kcAvn, err := avnClient.Services.Get(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, kcAvn.Name, kc.GetName())
	assert.Equal(t, "RUNNING", kc.Status.State)
	assert.Equal(t, kcAvn.State, kc.Status.State)
	assert.Equal(t, kcAvn.Plan, kc.Spec.Plan)
	assert.Equal(t, kcAvn.CloudName, kc.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, kc.Spec.Tags)

	// UserConfig test
	require.NotNil(t, kc.Spec.UserConfig)
	require.NotNil(t, kc.Spec.UserConfig.KafkaConnect)
	assert.Equal(t, anyPointer("read_committed"), kc.Spec.UserConfig.KafkaConnect.ConsumerIsolationLevel)
	require.NotNil(t, kc.Spec.UserConfig.PublicAccess)
	assert.Equal(t, anyPointer(true), kc.Spec.UserConfig.PublicAccess.KafkaConnect)

	// Validates ip filters
	require.Len(t, kc.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", kc.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *kc.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", kc.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, kc.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*kafkaconnectuserconfig.IpFilter
	require.NoError(t, castInterface(kcAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, kc.Spec.UserConfig.IpFilter)
}
