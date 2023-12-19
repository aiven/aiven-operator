package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	mysqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/mysql"
)

func getMySQLYaml(project, name, cloudName string, includeTechnicalEmails bool) string {
	baseYaml := `
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
`

	if includeTechnicalEmails {
		baseYaml += `
  technicalEmails:
    - email: "test@example.com"
`
	}

	return fmt.Sprintf(baseYaml, project, name, cloudName)
}

func TestMySQL(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx := context.Background()
	name := randName("mysql")
	yml := getMySQLYaml(testProject, name, testPrimaryCloudName, false)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ms := new(v1alpha1.MySQL)
	require.NoError(t, s.GetRunning(ms, name))

	// THEN
	msAvn, err := avnClient.Services.Get(ctx, testProject, name)
	require.NoError(t, err)
	assert.Equal(t, msAvn.Name, ms.GetName())
	assert.Equal(t, "RUNNING", ms.Status.State)
	assert.Equal(t, msAvn.State, ms.Status.State)
	assert.Equal(t, msAvn.Plan, ms.Spec.Plan)
	assert.Equal(t, msAvn.CloudName, ms.Spec.CloudName)
	assert.Equal(t, "100Gib", ms.Spec.DiskSpace)
	assert.Equal(t, 102400, msAvn.DiskSpaceMB)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, ms.Spec.Tags)
	msResp, err := avnClient.ServiceTags.Get(ctx, testProject, name)
	require.NoError(t, err)
	assert.Equal(t, msResp.Tags, ms.Spec.Tags)

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
}

func TestMySQLTechnicalEmails(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx := context.Background()
	name := randName("mysql")
	yml := getMySQLYaml(testProject, name, testPrimaryCloudName, true)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ms := new(v1alpha1.MySQL)
	require.NoError(t, s.GetRunning(ms, name))

	// THEN
	// Technical emails are set
	msAvn, err := avnClient.Services.Get(ctx, testProject, name)
	require.NoError(t, err)
	assert.Len(t, ms.Spec.TechnicalEmails, 1)
	assert.Equal(t, "test@example.com", msAvn.TechnicalEmails[0].Email)

	// WHEN
	// Technical emails are removed from manifest
	updatedYml := getMySQLYaml(testProject, name, testPrimaryCloudName, false)

	// Applies updated manifest
	require.NoError(t, s.Apply(updatedYml))

	// Waits kube objects
	require.NoError(t, s.GetRunning(ms, name))

	// THEN
	// Technical emails are removed from service
	msAvnUpdated, err := avnClient.Services.Get(ctx, testProject, name)
	require.NoError(t, err)
	assert.Empty(t, msAvnUpdated.TechnicalEmails)
}
