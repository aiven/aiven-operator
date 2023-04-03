package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	mysqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/mysql"
)

func getMySQLYaml(project, name string) string {
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
  cloudName: google-europe-west1
  plan: business-4
  disk_space: 100Gib

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

`, project, name)
}

func TestMySQL(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	name := randName("mysql")
	yml := getMySQLYaml(testProject, name)
	s, err := NewSession(k8sClient, avnClient, testProject, yml)
	require.NoError(t, err)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply())

	// Waits kube objects
	ms := new(v1alpha1.MySQL)
	require.NoError(t, s.GetRunning(ms, name))

	// THEN
	chAvn, err := avnClient.Services.Get(testProject, name)
	require.NoError(t, err)
	assert.Equal(t, chAvn.Name, ms.GetName())
	assert.Equal(t, "RUNNING", ms.Status.State)
	assert.Equal(t, chAvn.State, ms.Status.State)
	assert.Equal(t, chAvn.Plan, ms.Spec.Plan)
	assert.Equal(t, chAvn.CloudName, ms.Spec.CloudName)
	assert.Equal(t, "100Gib", ms.Spec.DiskSpace)
	assert.Equal(t, 102400, chAvn.DiskSpaceMB)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, ms.Spec.Tags)

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
	require.NoError(t, castInterface(chAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, ms.Spec.UserConfig.IpFilter)

	// Secrets test
	ctx := context.Background()
	secret := new(corev1.Secret)
	require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: "default"}, secret))
	assert.NotEmpty(t, secret.Data["MYSQL_HOST"])
	assert.NotEmpty(t, secret.Data["MYSQL_PORT"])
	assert.NotEmpty(t, secret.Data["MYSQL_DATABASE"])
	assert.NotEmpty(t, secret.Data["MYSQL_USER"])
	assert.NotEmpty(t, secret.Data["MYSQL_PASSWORD"])
	assert.NotEmpty(t, secret.Data["MYSQL_SSL_MODE"])
	assert.NotEmpty(t, secret.Data["MYSQL_URI"])
	assert.NotEmpty(t, secret.Data["MYSQL_REPLICA_URI"]) // business-4 has replica
}
