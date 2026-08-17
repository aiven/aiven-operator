//go:build mysql

package tests

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

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
	s := NewSession(ctx, k8sClient)

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
	assertServiceVersion(t, msAvn.Metadata["mysql_version"], ms.Status.Version)
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

	newServiceUser := func(t *testing.T) *v1alpha1.ServiceUser {
		t.Helper()
		userName := randName("mysql-user")
		yml, err := loadExampleYaml("serviceuser.yaml", map[string]string{
			"metadata.name":                    userName,
			"spec.project":                     cfg.Project,
			"spec.serviceName":                 name,
			"spec.connInfoSecretTarget.name":   userName,
			"spec.connInfoSecretTarget.prefix": "SERVICEUSER_",
		})
		require.NoError(t, err)
		user := new(v1alpha1.ServiceUser)
		require.NoError(t, yaml.Unmarshal([]byte(yml), user))
		user.Spec.Authentication = service.AuthenticationTypeMysqlNativePassword
		return user
	}

	waitForCredentials := func(t *testing.T, user *v1alpha1.ServiceUser, authentication service.AuthenticationType, password string) {
		t.Helper()
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, name, user.GetUsername())
			require.NoError(collect, err)
			assert.Equal(collect, authentication, userAvn.Authentication)
			assert.Equal(collect, password, userAvn.Password)

			secret, err := s.GetSecret(user.Spec.ConnInfoSecretTarget.Name)
			require.NoError(collect, err)
			assert.Equal(collect, user.GetUsername(), string(secret.Data["SERVICEUSER_USERNAME"]))
			assert.Equal(collect, password, string(secret.Data["SERVICEUSER_PASSWORD"]))
		}, 3*time.Minute, 10*time.Second, "authentication and published credentials should match")
	}

	t.Run("creates ServiceUser with requested authentication", func(t *testing.T) {
		user := newServiceUser(t)
		require.NoError(t, s.ApplyObjects(user))
		require.NoError(t, s.GetRunning(user, user.Name))

		userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, name, user.GetUsername())
		require.NoError(t, err)
		assert.Equal(t, service.AuthenticationTypeMysqlNativePassword, userAvn.Authentication)
		require.NotEmpty(t, userAvn.Password)
		waitForCredentials(t, user, service.AuthenticationTypeMysqlNativePassword, userAvn.Password)
	})

	t.Run("preserves the password when authentication changes in the spec", func(t *testing.T) {
		user := newServiceUser(t)
		require.NoError(t, s.ApplyObjects(user))
		require.NoError(t, s.GetRunning(user, user.Name))

		secret, err := s.GetSecret(user.Spec.ConnInfoSecretTarget.Name)
		require.NoError(t, err)
		password := string(secret.Data["SERVICEUSER_PASSWORD"])
		require.NotEmpty(t, password)
		waitForCredentials(t, user, service.AuthenticationTypeMysqlNativePassword, password)

		orig := user.DeepCopy()
		user.Spec.Authentication = service.AuthenticationTypeCachingSha2Password
		require.NoError(t, k8sClient.Patch(ctx, user, client.MergeFrom(orig)))

		waitForCredentials(t, user, service.AuthenticationTypeCachingSha2Password, password)
	})

	t.Run("restores authentication after an external change without changing the password", func(t *testing.T) {
		user := newServiceUser(t)
		require.NoError(t, s.ApplyObjects(user))
		require.NoError(t, s.GetRunning(user, user.Name))

		secret, err := s.GetSecret(user.Spec.ConnInfoSecretTarget.Name)
		require.NoError(t, err)
		password := string(secret.Data["SERVICEUSER_PASSWORD"])
		require.NotEmpty(t, password)
		waitForCredentials(t, user, service.AuthenticationTypeMysqlNativePassword, password)

		changed, err := avnGen.ServiceUserCredentialsModify(ctx, cfg.Project, name, user.GetUsername(), &service.ServiceUserCredentialsModifyIn{
			Authentication: service.AuthenticationTypeCachingSha2Password,
			NewPassword:    &password,
			Operation:      service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
		})
		require.NoError(t, err)
		idx := slices.IndexFunc(changed.Users, func(u service.UserOut) bool { return u.Username == user.GetUsername() })
		require.NotEqual(t, -1, idx)
		require.Equal(t, service.AuthenticationTypeCachingSha2Password, changed.Users[idx].Authentication)
		require.Equal(t, password, changed.Users[idx].Password)

		// Leave the CR untouched so the operator has to discover the drift.
		waitForCredentials(t, user, service.AuthenticationTypeMysqlNativePassword, password)
	})

	t.Run("restores authentication and the source password after an external change", func(t *testing.T) {
		user := newServiceUser(t)
		const sourcePassword = "SourceMySQLPassword123!"
		source := &corev1.Secret{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
			ObjectMeta: metav1.ObjectMeta{Name: user.Name + "-source"},
			Data:       map[string][]byte{"PASSWORD": []byte(sourcePassword)},
		}
		require.NoError(t, s.ApplyObjects(source))
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{Name: source.Name, PasswordKey: "PASSWORD"}
		require.NoError(t, s.ApplyObjects(user))
		require.NoError(t, s.GetRunning(user, user.Name))
		waitForCredentials(t, user, service.AuthenticationTypeMysqlNativePassword, sourcePassword)

		externalPassword := "ExternalMySQLPassword456!"
		changed, err := avnGen.ServiceUserCredentialsModify(ctx, cfg.Project, name, user.GetUsername(), &service.ServiceUserCredentialsModifyIn{
			Authentication: service.AuthenticationTypeCachingSha2Password,
			NewPassword:    &externalPassword,
			Operation:      service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
		})
		require.NoError(t, err)
		idx := slices.IndexFunc(changed.Users, func(u service.UserOut) bool { return u.Username == user.GetUsername() })
		require.NotEqual(t, -1, idx)
		require.Equal(t, service.AuthenticationTypeCachingSha2Password, changed.Users[idx].Authentication)
		require.Equal(t, externalPassword, changed.Users[idx].Password)

		waitForCredentials(t, user, service.AuthenticationTypeMysqlNativePassword, sourcePassword)
	})

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
