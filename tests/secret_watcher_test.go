package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// TestServiceUserSecretWatch tests ServiceUser password changes via connInfoSecretSource
func TestServiceUserSecretWatch(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	serviceName := randName("secret-watch-pg")
	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	// create PG service
	pgYaml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  cloudName: %s
  plan: startup-16
`, serviceName, cfg.Project, cfg.PrimaryCloudName)

	require.NoError(t, s.Apply(pgYaml))

	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, serviceName))

	t.Run("BasicSecretUpdate", func(t *testing.T) {
		userName := randName("secret-watch-user")
		secretName := randName("password-secret")
		yml := getServiceUserWithSecretSourceYaml(cfg.Project, serviceName, userName, secretName, cfg.PrimaryCloudName)

		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(user, userName))

		// THEN
		userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		require.NoError(t, err)
		assert.Equal(t, userName, userAvn.Username)

		initialPassword := userAvn.Password

		secret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      secretName,
			Namespace: user.Namespace,
		}, secret)
		require.NoError(t, err)

		// update password to new value
		secret.Data["password"] = []byte("updated-password-67890")
		err = k8sClient.Update(ctx, secret)
		require.NoError(t, err)

		// wait for the ServiceUser controller to process the password change
		require.Eventually(t, func() bool {
			finalUserAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
			if err != nil {
				return false
			}
			return finalUserAvn.Password == "updated-password-67890"
		}, 1*time.Minute, 10*time.Second, "ServiceUser should be reconciled with new password")

		// verify the password actually changed in Aiven
		finalUserAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		require.NoError(t, err)
		assert.Equal(t, userName, finalUserAvn.Username)
		assert.Equal(t, "updated-password-67890", finalUserAvn.Password, "password should be updated to the new value from the secret")
		assert.NotEqual(t, initialPassword, finalUserAvn.Password, "password should have changed from initial value")
	})

	t.Run("RaceCondition", func(t *testing.T) {
		userName := randName("race-user")
		secretName := randName("race-secret")

		yml := getServiceUserWithSecretSourceYaml(cfg.Project, serviceName, userName, secretName, cfg.PrimaryCloudName)
		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(user, userName))

		// apply changes to both secret and ServiceUser simultaneously
		t.Logf("TEST: Applying simultaneous changes to secret and ServiceUser")
		updatedYml := getUpdatedServiceUserAndSecretYaml(cfg.Project, serviceName, userName, secretName, cfg.PrimaryCloudName)

		// Get the current state before applying changes
		beforeUpdate := &v1alpha1.ServiceUser{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      userName,
			Namespace: user.Namespace,
		}, beforeUpdate)
		if err == nil {
			t.Logf("TEST: Before Apply - ResourceVersion: %s, Generation: %d",
				beforeUpdate.ResourceVersion, beforeUpdate.Generation)
		}

		applyErr := s.Apply(updatedYml)
		if applyErr != nil {
			t.Logf("TEST: Apply failed with error: %v", applyErr)
		} else {
			t.Logf("TEST: Apply succeeded")
		}
		require.NoError(t, applyErr)

		// verify secret watcher handles the race condition properly
		t.Logf("TEST: Starting to wait for race condition handling")

		var lastState string
		checkCount := 0

		require.Eventually(t, func() bool {
			checkCount++
			updated := &v1alpha1.ServiceUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      userName,
				Namespace: user.Namespace,
			}, updated)
			if err != nil {
				t.Logf("TEST: [Check %d] Failed to get ServiceUser: %v", checkCount, err)
				return false
			}

			// check user labels
			labels := updated.GetLabels()
			hasUserLabels := labels != nil &&
				labels["app.kubernetes.io/version"] == "v2.0" &&
				labels["environment"] == "production"

			// check that user's spec changed (authentication is mutable)
			hasUpdatedAuth := updated.Spec.Authentication == "caching_sha2_password"

			// check annotations for debugging
			annotations := updated.GetAnnotations()
			hasProcessedGeneration := annotations != nil && annotations["controllers.aiven.io/generation-was-processed"] != ""
			isRunning := annotations != nil && annotations["controllers.aiven.io/instance-is-running"] == "true"

			// Build current state string
			currentState := fmt.Sprintf("labels=%v auth=%s processedGen=%v running=%v gen=%d rv=%s",
				hasUserLabels, updated.Spec.Authentication, hasProcessedGeneration, isRunning,
				updated.Generation, updated.ResourceVersion)

			// Only log if state changed to reduce noise
			if currentState != lastState {
				t.Logf("TEST: [Check %d] State change: %s", checkCount, currentState)

				if labels != nil {
					t.Logf("TEST: [Check %d] Labels: version=%s env=%s", checkCount,
						labels["app.kubernetes.io/version"], labels["environment"])
				} else {
					t.Logf("TEST: [Check %d] Labels: nil", checkCount)
				}

				if annotations != nil {
					t.Logf("TEST: [Check %d] Annotations: processedGen=%s secretSourceUpdated=%s isRunning=%s",
						checkCount,
						annotations["controllers.aiven.io/generation-was-processed"],
						annotations["controllers.aiven.io/secret-source-updated"],
						annotations["controllers.aiven.io/instance-is-running"])
				} else {
					t.Logf("TEST: [Check %d] Annotations: nil", checkCount)
				}

				lastState = currentState
			}

			success := hasUserLabels && hasUpdatedAuth
			if success {
				t.Logf("TEST: [Check %d] SUCCESS! Both conditions met", checkCount)
			}

			return success
		}, 2*time.Minute, 5*time.Second, "secret watcher should handle race condition gracefully")

		// verify the password was actually updated
		require.Eventually(t, func() bool {
			finalUserAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
			if err != nil {
				return false
			}
			return finalUserAvn.Password == "updated-race-password-67890"
		}, 3*time.Minute, 10*time.Second, "password should be updated despite race condition")
	})

	t.Run("NameChangeRaceCondition", func(t *testing.T) {
		initialUserName := randName("initial-user")
		newUserName := randName("renamed-user")
		secretName := randName("name-change-secret")

		yml := getServiceUserWithSecretSourceYaml(cfg.Project, serviceName, initialUserName, secretName, cfg.PrimaryCloudName)
		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(user, initialUserName))

		// apply changes that rename ServiceUser AND update secret simultaneously
		t.Logf("TEST: Applying name change and secret update simultaneously")
		updatedYml := getServiceUserWithNameChangeAndSecretYaml(cfg.Project, serviceName, initialUserName, newUserName, secretName, cfg.PrimaryCloudName)
		require.NoError(t, s.Apply(updatedYml))

		t.Logf("TEST: Waiting for new ServiceUser to be running")
		newUser := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(newUser, newUserName))

		t.Logf("TEST: Starting to wait for stable state with all conditions")
		require.Eventually(t, func() bool {
			updated := &v1alpha1.ServiceUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      newUserName,
				Namespace: newUser.Namespace,
			}, updated)
			if err != nil {
				t.Logf("TEST: Failed to get renamed ServiceUser: %v", err)
				return false
			}

			labels := updated.GetLabels()
			hasUserLabels := labels != nil &&
				labels["app.kubernetes.io/name"] == "renamed-service-user" &&
				labels["version"] == "v3.0"

			annotations := updated.GetAnnotations()
			isRunning := annotations != nil && annotations["controllers.aiven.io/instance-is-running"] == "true"
			hasProcessedGeneration := annotations != nil && annotations["controllers.aiven.io/generation-was-processed"] != ""

			t.Logf("TEST: NameChange state - labels: %v, isRunning: %v, hasProcessedGen: %v, generation: %d, resourceVersion: %s",
				hasUserLabels, isRunning, hasProcessedGeneration,
				updated.Generation, updated.ResourceVersion)

			if annotations != nil {
				t.Logf("TEST: NameChange annotations: processedGen=%s, secretSourceUpdated=%s, isRunning=%s",
					annotations["controllers.aiven.io/generation-was-processed"],
					annotations["controllers.aiven.io/secret-source-updated"],
					annotations["controllers.aiven.io/instance-is-running"])
			}

			return hasUserLabels && isRunning && hasProcessedGeneration
		}, 2*time.Minute, 5*time.Second, "renamed ServiceUser should be running with correct labels")

		require.Eventually(t, func() bool {
			newUserAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, newUserName)
			if err != nil {
				return false
			}
			return newUserAvn.Password == "renamed-user-password-67890"
		}, 5*time.Minute, 10*time.Second, "renamed ServiceUser should have updated password")
	})
}

func getServiceUserWithSecretSourceYaml(project, serviceName, userName, secretName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  password: aW5pdGlhbFBhc3N3b3JkMTIzNDU= # initialPassword12345 # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[5]s
  plan: startup-16
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
    name: my-service-user-secret

  connInfoSecretSource:
    name: %[4]s
    passwordKey: password

  project: %[1]s
  serviceName: %[2]s
`, project, serviceName, userName, secretName, cloudName)
}

// TestClickhouseUserSecretWatch tests ClickhouseUser password changes via connInfoSecretSource
func TestClickhouseUserSecretWatch(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	serviceName := randName("secret-watch-ch")
	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	// create ClickHouse
	chYaml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  cloudName: %s
  plan: startup-16
  userConfig:
    public_access:
      clickhouse: true
`, serviceName, cfg.Project, cfg.PrimaryCloudName)

	require.NoError(t, s.Apply(chYaml))

	ch := new(v1alpha1.Clickhouse)
	require.NoError(t, s.GetRunning(ch, serviceName))

	t.Run("BasicSecretUpdate", func(t *testing.T) {
		userName := randName("secret-watch-ch-user")
		secretName := randName("ch-password-secret")
		yml := getClickhouseUserWithSecretSourceYaml(cfg.Project, serviceName, userName, secretName, cfg.PrimaryCloudName)

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

		yml := getClickhouseUserWithPasswordKeyYaml(cfg.Project, serviceName, userName, secretName, cfg.PrimaryCloudName, "PASSWORD")
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
		updatedYml := getClickhouseUserWithPasswordKeyYaml(cfg.Project, serviceName, userName, secretName, cfg.PrimaryCloudName, "SECRET_PASSWORD")
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

func getClickhouseUserWithSecretSourceYaml(project, serviceName, userName, secretName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  PASSWORD: Y2xpY2tob3VzZS1wYXNzd29yZC0xMjM0NQ== # clickhouse-password-12345 # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[5]s
  plan: startup-16

  userConfig:
    public_access:
      clickhouse: true
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
    name: my-clickhouse-user-secret

  connInfoSecretSource:
    name: %[4]s
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, serviceName, userName, secretName, cloudName)
}

// TestCrossNamespaceSecretWatch tests secret watching across namespaces
func TestCrossNamespaceSecretWatch(t *testing.T) {
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	serviceName := randName("cross-ns-pg")
	userName := randName("cross-ns-user")
	secretName := randName("cross-ns-secret")
	secretNamespace := "test-secrets"

	testNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: secretNamespace},
	}
	err := k8sClient.Create(ctx, testNs)
	require.NoError(t, err)

	defer func() {
		deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer deleteCancel()
		_ = k8sClient.Delete(deleteCtx, testNs)
	}()

	yml := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
  namespace: %[5]s
data:
  password: Y3Jvc3MtbnMtcGFzc3dvcmQtMTIzNDU= # cross-ns-password-12345 # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[6]s
  plan: startup-16
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
    name: cross-ns-service-user-secret

  connInfoSecretSource:
    name: %[4]s
    namespace: %[5]s
    passwordKey: password

  project: %[1]s
  serviceName: %[2]s
`, cfg.Project, serviceName, userName, secretName, secretNamespace, cfg.PrimaryCloudName)

	s := NewSession(ctx, k8sClient)

	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	// wait for PostgreSQL service to be running
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, serviceName))

	// wait for ServiceUser to be running
	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, userAvn.Username)

	initialPassword := userAvn.Password

	secret := &corev1.Secret{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: secretNamespace,
	}, secret)
	require.NoError(t, err)

	secret.Data["password"] = []byte("updated-cross-ns-password-67890")
	err = k8sClient.Update(ctx, secret)
	require.NoError(t, err)

	// wait for the password to be updated in Aiven
	require.Eventually(t, func() bool {
		finalUserAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		if err != nil {
			return false
		}
		return finalUserAvn.Password == "updated-cross-ns-password-67890"
	}, 1*time.Minute, 5*time.Second, "Cross-namespace secret watcher should trigger password update")

	finalUserAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, finalUserAvn.Username)

	assert.Equal(t, "updated-cross-ns-password-67890", finalUserAvn.Password, "Cross-namespace password should be updated to the new value from the secret")
	assert.NotEqual(t, initialPassword, finalUserAvn.Password, "Cross-namespace password should have changed from initial value")
}

// getUpdatedServiceUserAndSecretYaml returns YAML with both secret and ServiceUser changes
// This simulates a user doing "kubectl apply -f config.yaml" with multiple resource changes
func getUpdatedServiceUserAndSecretYaml(project, serviceName, userName, secretName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  password: dXBkYXRlZC1yYWNlLXBhc3N3b3JkLTY3ODkw # updated-race-password-67890 # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[5]s
  plan: startup-16
---
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: %[3]s
  labels:
    app.kubernetes.io/version: "v2.0"  # User adds this label
    environment: "production"         # User adds this label
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: my-service-user-secret

  connInfoSecretSource:
    name: %[4]s
    passwordKey: password

  project: %[1]s
  serviceName: %[2]s

  authentication: caching_sha2_password  # User changes authentication method
`, project, serviceName, userName, secretName, cloudName)
}

// getServiceUserWithNameChangeAndSecretYaml returns YAML that renames ServiceUser and updates secret
// This simulates the most complex race condition scenario
func getServiceUserWithNameChangeAndSecretYaml(project, serviceName, oldUserName, newUserName, secretName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[5]s
data:
  password: cmVuYW1lZC11c2VyLXBhc3N3b3JkLTY3ODkw # renamed-user-password-67890 # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[6]s
  plan: startup-16
---
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: %[4]s  # Changed from %[3]s to %[4]s
  labels:
    app.kubernetes.io/name: "renamed-service-user"  # User adds labels during rename
    version: "v3.0"
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: my-renamed-service-user-secret

  connInfoSecretSource:
    name: %[5]s
    passwordKey: password

  project: %[1]s
  serviceName: %[2]s
`, project, serviceName, oldUserName, newUserName, secretName, cloudName)
}

// getClickhouseUserWithPasswordKeyYaml creates YAML for ClickhouseUser with specified passwordKey
func getClickhouseUserWithPasswordKeyYaml(project, serviceName, userName, secretName, cloudName, passwordKey string) string {
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
    name: my-ch-pwd-key-secret

  connInfoSecretSource:
    name: %[4]s
    passwordKey: %[7]s

  project: %[1]s
  serviceName: %[2]s
`, project, serviceName, userName, secretName, cloudName, passwordValue, passwordKey)
}
