package tests

import (
	"fmt"
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	mysqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/mysql"
)

func getMySQLYaml(project, name, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: MySQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: business-4
  disk_space: 100GiB

  tags:
    env: test
    instance: foo

  userConfig:
    backup_hour: 12
    backup_minute: 42
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16

`, project, name, cloudName)
}

func TestMySQL(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("mysql")
	yml := getMySQLYaml(cfg.Project, name, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ms := new(v1alpha1.MySQL)
	require.NoError(t, s.GetRunning(ms, name))

	// THEN
	msAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, msAvn.ServiceName, ms.GetName())
	assert.Equal(t, serviceRunningState, ms.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, msAvn.State)
	assert.Equal(t, msAvn.Plan, ms.Spec.Plan)
	assert.Equal(t, msAvn.CloudName, ms.Spec.CloudName)
	assert.Equal(t, "100GiB", ms.Spec.DiskSpace)
	assert.Equal(t, int(102400), *msAvn.DiskSpaceMb)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, ms.Spec.Tags)
	msTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, msTags, ms.Spec.Tags)

	// UserConfig test
	require.NotNil(t, ms.Spec.UserConfig)
	assert.Equal(t, anyPointer(12), ms.Spec.UserConfig.BackupHour)
	assert.Equal(t, anyPointer(42), ms.Spec.UserConfig.BackupMinute)

	// Validates ip filters
	require.Len(t, ms.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", ms.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *ms.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", ms.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, ms.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*mysqluserconfig.IpFilter
	require.NoError(t, castInterface(msAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, ms.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(ms.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["MYSQL_HOST"])
	assert.NotEmpty(t, secret.Data["MYSQL_PORT"])
	assert.NotEmpty(t, secret.Data["MYSQL_DATABASE"])
	assert.NotEmpty(t, secret.Data["MYSQL_USER"])
	assert.NotEmpty(t, secret.Data["MYSQL_PASSWORD"])
	assert.NotEmpty(t, secret.Data["MYSQL_SSL_MODE"])
	assert.NotEmpty(t, secret.Data["MYSQL_URI"])
	assert.NotEmpty(t, secret.Data["MYSQL_REPLICA_URI"]) // business-4 has replica
	assert.NotEmpty(t, secret.Data["MYSQL_CA_CERT"])

	// Tests service power off functionality
	// Note: Power on testing is handled generically in generic_service_handler_test.go
	// since it's consistent across services. Power off testing is done here since
	// the flow can vary by service type and may require service-specific steps.
	poweredOff := ms.DeepCopy()
	poweredOff.Spec.Powered = anyPointer(false)
	require.NoError(t, k8sClient.Update(ctx, poweredOff))
	require.NoError(t, s.GetRunning(poweredOff, name))

	poweredOffAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypePoweroff, poweredOffAvn.State)
}
