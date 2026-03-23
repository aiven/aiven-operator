//go:build valkey

package tests

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestServiceUserValkeyAccessControl(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	serviceName := randName("valkey-su")
	userName := randName("valkey-user")
	sourceSecretName := randName("valkey-su-source")
	targetSecretName := randName("valkey-su-target")

	const (
		initialPassword = "InitialValkeyPassword123!"
		updatedPassword = "UpdatedValkeyPassword67890!"
	)

	initialAccessControl := &v1alpha1.ServiceUserAccessControl{
		ValkeyACLKeys:       []string{"cache:*"},
		ValkeyACLCommands:   []string{"-acl"},
		ValkeyACLCategories: []string{"+@all"},
		ValkeyACLChannels:   []string{"initial*"},
	}

	updatedAccessControl := &v1alpha1.ServiceUserAccessControl{
		ValkeyACLKeys:       []string{"cache:updated:*"},
		ValkeyACLCommands:   []string{"-slowlog"},
		ValkeyACLCategories: []string{"+@read"},
		ValkeyACLChannels:   []string{"updated*"},
	}

	yml, err := getServiceUserValkeyCreateYAML(
		cfg.Project,
		cfg.PrimaryCloudName,
		serviceName,
		userName,
		sourceSecretName,
		targetSecretName,
		initialPassword,
		initialAccessControl,
	)
	require.NoError(t, err)

	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	valkey := new(v1alpha1.Valkey)
	require.NoError(t, s.GetRunning(valkey, serviceName))

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	require.Eventually(t, func() bool {
		userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		if err != nil {
			return false
		}

		return userAvn.Password == initialPassword &&
			serviceUserValkeyAccessControlEquals(userAvn.AccessControl, initialAccessControl)
	}, 3*time.Minute, 10*time.Second, "ServiceUser should be created with managed Valkey ACL and custom password")

	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, userAvn.Username)
	assert.Equal(t, initialPassword, userAvn.Password)
	assert.True(t, serviceUserValkeyAccessControlEquals(userAvn.AccessControl, initialAccessControl))

	secret, err := s.GetSecret(targetSecretName)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["SERVICEUSER_HOST"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_PORT"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_USERNAME"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_PASSWORD"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_CA_CERT"])
	assert.Contains(t, secret.Data, "SERVICEUSER_ACCESS_CERT")
	assert.Contains(t, secret.Data, "SERVICEUSER_ACCESS_KEY")
	assert.Equal(t, userName, string(secret.Data["SERVICEUSER_USERNAME"]))
	assert.Equal(t, initialPassword, string(secret.Data["SERVICEUSER_PASSWORD"]))

	updatedYml := getServiceUserValkeyUpdateYAML(
		cfg.Project,
		serviceName,
		userName,
		sourceSecretName,
		targetSecretName,
		updatedPassword,
		updatedAccessControl,
	)

	require.NoError(t, s.Apply(updatedYml))

	updatedUser := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(updatedUser, userName))

	require.Eventually(t, func() bool {
		userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		if err != nil {
			return false
		}

		secret, err := s.GetSecret(targetSecretName)
		if err != nil {
			return false
		}

		return userAvn.Password == updatedPassword &&
			serviceUserValkeyAccessControlEquals(userAvn.AccessControl, updatedAccessControl) &&
			string(secret.Data["SERVICEUSER_PASSWORD"]) == updatedPassword
	}, 3*time.Minute, 10*time.Second, "ServiceUser should reconcile Valkey ACL and password changes together")

	updatedUserAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
	require.NoError(t, err)
	assert.Equal(t, updatedPassword, updatedUserAvn.Password)
	assert.True(t, serviceUserValkeyAccessControlEquals(updatedUserAvn.AccessControl, updatedAccessControl))

	updatedSecret, err := s.GetSecret(targetSecretName)
	require.NoError(t, err)
	assert.NotEmpty(t, updatedSecret.Data["SERVICEUSER_HOST"])
	assert.NotEmpty(t, updatedSecret.Data["SERVICEUSER_PORT"])
	assert.NotEmpty(t, updatedSecret.Data["SERVICEUSER_CA_CERT"])
	assert.Contains(t, updatedSecret.Data, "SERVICEUSER_ACCESS_CERT")
	assert.Contains(t, updatedSecret.Data, "SERVICEUSER_ACCESS_KEY")
	assert.Equal(t, userName, string(updatedSecret.Data["SERVICEUSER_USERNAME"]))
	assert.Equal(t, updatedPassword, string(updatedSecret.Data["SERVICEUSER_PASSWORD"]))

	assert.NoError(t, s.Delete(updatedUser, func() error {
		_, err = avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		return err
	}))
}

func getServiceUserValkeyCreateYAML(
	project, cloudName, serviceName, userName, sourceSecretName, targetSecretName, password string,
	accessControl *v1alpha1.ServiceUserAccessControl,
) (string, error) {
	valkeyYml, err := loadExampleYaml("valkey.yaml", map[string]string{
		"metadata.name":                  serviceName,
		"spec.project":                   project,
		"spec.cloudName":                 cloudName,
		"spec.connInfoSecretTarget.name": serviceName,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s\n---\n%s", valkeyYml, getServiceUserValkeyUpdateYAML(
		project,
		serviceName,
		userName,
		sourceSecretName,
		targetSecretName,
		password,
		accessControl,
	)), nil
}

func getServiceUserValkeyUpdateYAML(
	project, serviceName, userName, sourceSecretName, targetSecretName, password string,
	accessControl *v1alpha1.ServiceUserAccessControl,
) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  password: %[6]s
---
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: %[5]s

  connInfoSecretSource:
    name: %[4]s
    passwordKey: password

  project: %[1]s
  serviceName: %[2]s
%[7]s
`, project, serviceName, userName, sourceSecretName, targetSecretName, base64.StdEncoding.EncodeToString([]byte(password)), serviceUserValkeyAccessControlBlock(accessControl))
}

func serviceUserValkeyAccessControlBlock(accessControl *v1alpha1.ServiceUserAccessControl) string {
	if accessControl == nil {
		return ""
	}

	return fmt.Sprintf(`  accessControl:
    valkeyAclKeys:%s
    valkeyAclCommands:%s
    valkeyAclCategories:%s
    valkeyAclChannels:%s
`, yamlStringList(accessControl.ValkeyACLKeys), yamlStringList(accessControl.ValkeyACLCommands), yamlStringList(accessControl.ValkeyACLCategories), yamlStringList(accessControl.ValkeyACLChannels))
}

func yamlStringList(items []string) string {
	if len(items) == 0 {
		return " []"
	}

	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "\n      - %q", item)
	}

	return b.String()
}

func serviceUserValkeyAccessControlEquals(
	actual *service.AccessControlOut,
	expected *v1alpha1.ServiceUserAccessControl,
) bool {
	if expected == nil {
		return actual == nil
	}

	if actual == nil {
		return false
	}

	return slices.Equal(actual.ValkeyAclKeys, expected.ValkeyACLKeys) &&
		slices.Equal(actual.ValkeyAclCommands, expected.ValkeyACLCommands) &&
		slices.Equal(actual.ValkeyAclCategories, expected.ValkeyACLCategories) &&
		slices.Equal(actual.ValkeyAclChannels, expected.ValkeyACLChannels)
}
