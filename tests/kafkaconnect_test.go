package tests

import (
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	kafkaconnectuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka_connect"
)

func TestKafkaConnect(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("kafka-connect")
	yml, err := loadExampleYaml("kafkaconnect.yaml", map[string]string{
		"metadata.name":  name,
		"spec.project":   cfg.Project,
		"spec.cloudName": cfg.PrimaryCloudName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	kc := new(v1alpha1.KafkaConnect)
	require.NoError(t, s.GetRunning(kc, name))

	// THEN
	kcAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, kcAvn.ServiceName, kc.GetName())
	assert.Equal(t, serviceRunningState, kc.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, kcAvn.State)
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

	// Powers off
	powered := false
	kcPoweredOff := kc.DeepCopy()
	kcPoweredOff.Spec.Powered = &powered
	require.NoError(t, k8sClient.Update(ctx, kcPoweredOff))
	require.NoError(t, s.GetRunning(kcPoweredOff, name))

	kcAvnPoweredOff, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypePoweroff, kcAvnPoweredOff.State)

	// Powers on
	powered = true
	kcPoweredOn := kcPoweredOff.DeepCopy()
	kcPoweredOn.Spec.Powered = &powered
	require.NoError(t, k8sClient.Update(ctx, kcPoweredOn))
	require.NoError(t, s.GetRunning(kcPoweredOn, name))

	kcAvnPoweredOn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypeRunning, kcAvnPoweredOn.State)
}
