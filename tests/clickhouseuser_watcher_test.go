//go:build clickhouse

package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// TestClickhouseUserSecretWatch tests ClickhouseUser password changes via connInfoSecretSource
func TestClickhouseUserSecretWatch(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	ch, release, err := sharedResources.AcquireClickhouse(ctx, WithClickhouseTags(map[string]string{"test": "TestClickhouseUserSecretWatch"}))
	require.NoError(t, err)
	defer release()

	serviceName := ch.GetName()
	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	t.Run("BasicSecretUpdate", func(t *testing.T) {
		userName := randName("secret-watch-ch-user")
		secretName := randName("ch-password-secret")
		yml := getClickhouseUserWithSecretSourceYaml(cfg.Project, serviceName, userName, secretName)

		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ClickhouseUser)
		require.NoError(t, s.GetRunning(user, userName))

		// THEN
		userAvn, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, serviceName, user.Status.UUID)
		require.NoError(t, err)
		assert.Equal(t, userName, userAvn.Name)

		secret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      user.Spec.ConnInfoSecretTarget.Name,
			Namespace: user.Namespace,
		}, secret)
		require.NoError(t, err)
		assert.Equal(t, "clickhouse-password-12345", string(secret.Data["PASSWORD"]), "Initial password should match the secret")

		require.NoError(t, pingClickhouse(
			ctx,
			secret.Data["HOST"],
			secret.Data["PORT"],
			secret.Data["USERNAME"],
			secret.Data["PASSWORD"],
		), "initial password should allow ClickHouse connection")

		passwordSecret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      secretName,
			Namespace: user.Namespace,
		}, passwordSecret)
		require.NoError(t, err)

		passwordSecret.Data["PASSWORD"] = []byte("updated-clickhouse-password-67890")
		err = k8sClient.Update(ctx, passwordSecret)
		require.NoError(t, err)

		// verify the password was updated in the target secret
		require.Eventually(t, func() bool {
			finalSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      user.Spec.ConnInfoSecretTarget.Name,
				Namespace: user.Namespace,
			}, finalSecret)
			if err != nil {
				return false
			}

			actualPassword := string(finalSecret.Data["PASSWORD"])
			return actualPassword == "updated-clickhouse-password-67890"
		}, 1*time.Minute, 5*time.Second, "password should be updated to the new value from the secret")

		// get the final secret
		finalSecret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      user.Spec.ConnInfoSecretTarget.Name,
			Namespace: user.Namespace,
		}, finalSecret)
		require.NoError(t, err)

		assert.NoError(t, pingClickhouse(
			ctx,
			finalSecret.Data["HOST"],
			finalSecret.Data["PORT"],
			finalSecret.Data["USERNAME"],
			finalSecret.Data["PASSWORD"],
		), "updated password should allow ClickHouse connection")

		// test that the OLD password no longer works
		assert.Error(t, pingClickhouse(
			ctx,
			finalSecret.Data["HOST"],
			finalSecret.Data["PORT"],
			finalSecret.Data["USERNAME"],
			[]byte("clickhouse-password-12345"),
		), "old password should no longer work after update")
	})

	t.Run("PasswordKeyAndPasswordUpdate", func(t *testing.T) {
		userName := randName("ch-pwd-key-user")
		secretName := randName("ch-pwd-key-secret")
		secretTargetName := randName("ch-pwd-key-target-secret") // Generate once for consistency

		yml := getClickhouseUserWithPasswordKeyYamlWithTarget(cfg.Project, serviceName, userName, secretName, cfg.PrimaryCloudName, "PASSWORD", secretTargetName)
		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ClickhouseUser)
		require.NoError(t, s.GetRunning(user, userName))

		userAvn, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, serviceName, user.Status.UUID)
		require.NoError(t, err)
		assert.Equal(t, userName, userAvn.Name)

		initialSecret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      user.Spec.ConnInfoSecretTarget.Name,
			Namespace: user.Namespace,
		}, initialSecret)
		require.NoError(t, err)
		assert.Equal(t, "initial-ch-password-12345", string(initialSecret.Data["PASSWORD"]))

		// Apply simultaneous changes - change passwordKey from PASSWORD to SECRET_PASSWORD
		// AND update the password value in the secret
		updatedYml := getClickhouseUserWithPasswordKeyYamlWithTarget(cfg.Project, serviceName, userName, secretName, cfg.PrimaryCloudName, "SECRET_PASSWORD", secretTargetName)
		require.NoError(t, s.Apply(updatedYml))

		require.Eventually(t, func() bool {
			updatedUser := &v1alpha1.ClickhouseUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      userName,
				Namespace: user.Namespace,
			}, updatedUser)
			if err != nil {
				return false
			}

			if updatedUser.Spec.ConnInfoSecretSource == nil {
				return false
			}

			passwordKeyUpdated := updatedUser.Spec.ConnInfoSecretSource.PasswordKey == "SECRET_PASSWORD"

			// check that user is running with processed generation
			annotations := updatedUser.GetAnnotations()
			isRunning := annotations != nil && annotations["controllers.aiven.io/instance-is-running"] == "true"
			hasProcessedGeneration := annotations != nil && annotations["controllers.aiven.io/generation-was-processed"] != ""

			return passwordKeyUpdated && isRunning && hasProcessedGeneration
		}, 2*time.Minute, 5*time.Second, "ClickhouseUser should be updated with new passwordKey and running")

		require.Eventually(t, func() bool {
			finalSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      user.Spec.ConnInfoSecretTarget.Name,
				Namespace: user.Namespace,
			}, finalSecret)
			if err != nil {
				return false
			}

			actualPassword := string(finalSecret.Data["PASSWORD"])
			expectedPassword := "updated-secret-password-67890"
			return actualPassword == expectedPassword
		}, 2*time.Minute, 5*time.Second, "target secret should contain password from new SECRET_PASSWORD key")

		// verify the connection works with the new password
		finalSecret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      user.Spec.ConnInfoSecretTarget.Name,
			Namespace: user.Namespace,
		}, finalSecret)
		require.NoError(t, err)

		require.NoError(t, pingClickhouse(
			ctx,
			finalSecret.Data["HOST"],
			finalSecret.Data["PORT"],
			finalSecret.Data["USERNAME"],
			finalSecret.Data["PASSWORD"],
		), "updated password from SECRET_PASSWORD key should allow ClickHouse connection")
	})
}

func getClickhouseUserWithSecretSourceYaml(project, serviceName, userName, secretName string) string {
	secretTargetName := randName("clickhouse-user-secret")
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  PASSWORD: Y2xpY2tob3VzZS1wYXNzd29yZC0xMjM0NQ== # clickhouse-password-12345 # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
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
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, serviceName, userName, secretName, secretTargetName)
}

// getClickhouseUserWithPasswordKeyYaml creates YAML for ClickhouseUser with specified passwordKey
func getClickhouseUserWithPasswordKeyYaml(project, serviceName, userName, secretName, cloudName, passwordKey string) string {
	secretTargetName := randName("ch-pwd-key-secret")
	var passwordValue string
	switch passwordKey {
	case "PASSWORD":
		passwordValue = "aW5pdGlhbC1jaC1wYXNzd29yZC0xMjM0NQ==" // initial-ch-password-12345
	case "SECRET_PASSWORD":
		passwordValue = "dXBkYXRlZC1zZWNyZXQtcGFzc3dvcmQtNjc4OTA=" // updated-secret-password-67890
	default:
		passwordValue = "ZGVmYXVsdC1wYXNzd29yZA==" // default-password # gitleaks:allow
	}

	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  PASSWORD: aW5pdGlhbC1jaC1wYXNzd29yZC0xMjM0NQ== # initial-ch-password-12345 # gitleaks:allow
  SECRET_PASSWORD: %[6]s # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: %[8]s

  connInfoSecretSource:
    name: %[4]s
    passwordKey: %[7]s

  project: %[1]s
  serviceName: %[2]s
`, project, serviceName, userName, secretName, cloudName, passwordValue, passwordKey, secretTargetName)
}

// getClickhouseUserWithPasswordKeyYamlWithTarget creates YAML for ClickhouseUser with specified passwordKey and explicit target name
func getClickhouseUserWithPasswordKeyYamlWithTarget(project, serviceName, userName, secretName, cloudName, passwordKey, secretTargetName string) string {
	var passwordValue string
	switch passwordKey {
	case "PASSWORD":
		passwordValue = "aW5pdGlhbC1jaC1wYXNzd29yZC0xMjM0NQ==" // initial-ch-password-12345
	case "SECRET_PASSWORD":
		passwordValue = "dXBkYXRlZC1zZWNyZXQtcGFzc3dvcmQtNjc4OTA=" // updated-secret-password-67890
	default:
		passwordValue = "ZGVmYXVsdC1wYXNzd29yZA==" // default-password # gitleaks:allow
	}

	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  PASSWORD: aW5pdGlhbC1jaC1wYXNzd29yZC0xMjM0NQ== # initial-ch-password-12345 # gitleaks:allow
  SECRET_PASSWORD: %[6]s # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: %[8]s

  connInfoSecretSource:
    name: %[4]s
    passwordKey: %[7]s

  project: %[1]s
  serviceName: %[2]s
`, project, serviceName, userName, secretName, cloudName, passwordValue, passwordKey, secretTargetName)
}
