//go:build postgresql

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

	pg, release, err := sharedResources.AcquirePostgreSQL(ctx)
	require.NoError(t, err)
	defer release()

	serviceName := pg.GetName()
	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	t.Run("BasicSecretUpdate", func(t *testing.T) {
		userName := randName("secret-watch-user")
		secretName := randName("password-secret")
		targetSecretName := randName("service-user-secret")
		yml := getServiceUserWithSecretSourceYaml(cfg.Project, serviceName, userName, secretName, targetSecretName)

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
		targetSecretName := randName("race-service-user-secret")

		yml := getServiceUserWithSecretSourceYaml(cfg.Project, serviceName, userName, secretName, targetSecretName)
		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(user, userName))

		// apply changes to both secret and ServiceUser simultaneously
		updatedYml := getUpdatedServiceUserAndSecretYaml(cfg.Project, serviceName, userName, secretName, targetSecretName)
		require.NoError(t, s.Apply(updatedYml))

		// verify secret watcher handles the race condition properly
		require.Eventually(t, func() bool {
			updated := &v1alpha1.ServiceUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      userName,
				Namespace: user.Namespace,
			}, updated)
			if err != nil {
				return false
			}

			// check user labels
			labels := updated.GetLabels()
			hasUserLabels := labels != nil &&
				labels["app.kubernetes.io/version"] == "v2.0" &&
				labels["environment"] == "production"

			// check that user's spec changed (authentication is mutable)
			hasUpdatedAuth := updated.Spec.Authentication == "caching_sha2_password"

			return hasUserLabels && hasUpdatedAuth
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
		oldTargetSecretName := randName("initial-service-user-secret")
		newTargetSecretName := randName("renamed-service-user-secret")

		yml := getServiceUserWithSecretSourceYaml(cfg.Project, serviceName, initialUserName, secretName, oldTargetSecretName)
		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(user, initialUserName))

		// apply changes that rename ServiceUser AND update secret simultaneously
		updatedYml := getServiceUserWithNameChangeAndSecretYaml(cfg.Project, serviceName, initialUserName, newUserName, secretName, newTargetSecretName)
		require.NoError(t, s.Apply(updatedYml))

		newUser := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(newUser, newUserName))

		require.Eventually(t, func() bool {
			updated := &v1alpha1.ServiceUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      newUserName,
				Namespace: newUser.Namespace,
			}, updated)
			if err != nil {
				return false
			}

			labels := updated.GetLabels()
			hasUserLabels := labels != nil &&
				labels["app.kubernetes.io/name"] == "renamed-service-user" &&
				labels["version"] == "v3.0"

			annotations := updated.GetAnnotations()
			isRunning := annotations != nil && annotations["controllers.aiven.io/instance-is-running"] == "true"
			hasProcessedGeneration := annotations != nil && annotations["controllers.aiven.io/generation-was-processed"] != ""

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

func getServiceUserWithSecretSourceYaml(project, serviceName, userName, secretName, targetSecretName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  password: aW5pdGlhbFBhc3N3b3JkMTIzNDU= # initialPassword12345 # gitleaks:allow
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
`, project, serviceName, userName, secretName, targetSecretName)
}

// TestCrossNamespaceSecretWatch tests secret watching across namespaces
func TestCrossNamespaceSecretWatch(t *testing.T) {
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	pg, release, err := sharedResources.AcquirePostgreSQL(ctx)
	require.NoError(t, err)
	defer release()

	serviceName := pg.GetName()
	userName := randName("cross-ns-user")
	secretName := randName("cross-ns-secret")
	secretNamespace := "test-secrets"

	testNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: secretNamespace},
	}
	err = k8sClient.Create(ctx, testNs)
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
`, cfg.Project, serviceName, userName, secretName, secretNamespace)

	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

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
func getUpdatedServiceUserAndSecretYaml(project, serviceName, userName, secretName, targetSecretName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  password: dXBkYXRlZC1yYWNlLXBhc3N3b3JkLTY3ODkw # updated-race-password-67890 # gitleaks:allow
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
    name: %[5]s

  connInfoSecretSource:
    name: %[4]s
    passwordKey: password

  project: %[1]s
  serviceName: %[2]s

  authentication: caching_sha2_password  # User changes authentication method
`, project, serviceName, userName, secretName, targetSecretName)
}

// getServiceUserWithNameChangeAndSecretYaml returns YAML that renames ServiceUser and updates secret
// This simulates the most complex race condition scenario
func getServiceUserWithNameChangeAndSecretYaml(project, serviceName, oldUserName, newUserName, secretName, targetSecretName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[5]s
data:
  password: cmVuYW1lZC11c2VyLXBhc3N3b3JkLTY3ODkw # renamed-user-password-67890 # gitleaks:allow
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
    name: %[6]s

  connInfoSecretSource:
    name: %[5]s
    passwordKey: password

  project: %[1]s
  serviceName: %[2]s
`, project, serviceName, oldUserName, newUserName, secretName, targetSecretName)
}
