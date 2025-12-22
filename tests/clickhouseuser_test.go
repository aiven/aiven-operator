//go:build clickhouse

package tests

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	avngen "github.com/aiven/go-client-codegen"
	clickhouse2 "github.com/aiven/go-client-codegen/handler/clickhouse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

func TestClickhouseUser(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	ch, release, err := sharedResources.AcquireClickhouse(ctx)
	require.NoError(t, err)
	defer release()

	chName := ch.GetName()
	userName := randName("clickhouse-user")
	yml, err := loadExampleYaml("clickhouseuser.yaml", map[string]string{
		"metadata.name":                  userName,
		"spec.project":                   cfg.Project,
		"spec.serviceName":               chName,
		"spec.connInfoSecretTarget.name": userName,
		// Remove 'username' from the initial yaml
		"spec.username": "REMOVE",
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects

	// THEN
	chAvn, err := avnGen.ServiceGet(ctx, cfg.Project, chName)
	require.NoError(t, err)
	assert.Equal(t, chAvn.ServiceName, ch.GetName())
	assert.Equal(t, serviceRunningState, ch.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, chAvn.State)
	assert.Equal(t, chAvn.Plan, ch.Spec.Plan)
	assert.Equal(t, chAvn.CloudName, ch.Spec.CloudName)

	user := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.GetRunning(user, userName))

	userAvn, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
	require.NoError(t, err)

	// Gets name from `metadata.name` when `username` is not set
	assert.Equal(t, userName, user.ObjectMeta.Name)
	assert.Equal(t, userAvn.Name, user.ObjectMeta.Name)

	secret, err := s.GetSecret(user.Spec.ConnInfoSecretTarget.Name)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["USERNAME"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_HOST"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_PORT"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_PASSWORD"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_USERNAME"])
	assert.Equal(t, map[string]string{"foo": "bar"}, secret.Annotations)
	assert.Equal(t, map[string]string{"baz": "egg"}, secret.Labels)
	// Secret should use 'metadata.name' as 'username'
	assert.EqualValues(t, secret.Data["USERNAME"], user.ObjectMeta.Name)
	assert.EqualValues(t, secret.Data["CLICKHOUSEUSER_USERNAME"], user.ObjectMeta.Name)

	// Secrets validation
	pinger := func() error {
		return pingClickhouse(
			ctx,
			secret.Data["CLICKHOUSEUSER_HOST"],
			secret.Data["CLICKHOUSEUSER_PORT"],
			secret.Data["CLICKHOUSEUSER_USERNAME"],
			secret.Data["CLICKHOUSEUSER_PASSWORD"],
		)
	}
	assert.NoError(t, pinger())

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, user is destroyed in Aiven. No service — no user. No user — no user.
	// And we make sure that controller can delete user itself
	assert.NoError(t, s.Delete(user, func() error {
		_, err = getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
		return err
	}))

	// User has been deleted, no access
	require.ErrorContains(t, pinger(), "Authentication failed: password is incorrect, or there is no user with such name.")

	// GIVEN
	// New manifest with 'username' field set
	updatedUserName := randName("clickhouse-user")
	updatedSecretName := randName("clickhouse-user-secret")
	ymlUsernameSet, err := loadExampleYaml("clickhouseuser.yaml", map[string]string{
		"metadata.name":                  "metadata-name",
		"spec.project":                   cfg.Project,
		"spec.connInfoSecretTarget.name": updatedSecretName,
		"spec.serviceName":               chName,
		"spec.username":                  updatedUserName,
	})
	require.NoError(t, err)

	// WHEN
	// Applies updated manifest
	updatedUser := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.Apply(ymlUsernameSet))
	require.NoError(t, s.GetRunning(updatedUser, "metadata-name")) // GetRunning must be called with the metadata name

	updatedUserAvn, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, updatedUser.Status.UUID)
	require.NoError(t, err)

	// THEN
	// 'username' field is preferred over 'metadata.name'
	assert.NotEqual(t, updatedUserName, updatedUser.ObjectMeta.Name)
	assert.NotEqual(t, updatedUserAvn.Name, updatedUser.ObjectMeta.Name)
	assert.Equal(t, updatedUserName, updatedUser.Spec.Username)
	assert.Equal(t, updatedUserAvn.Name, updatedUser.Spec.Username)
}

func TestClickhouseUserPreservesSecretPasswordOnUpdate(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	ch, release, err := sharedResources.AcquireClickhouse(ctx)
	require.NoError(t, err)
	defer release()

	chName := ch.GetName()
	userName := randName("chu-secret-compat")
	yml, err := loadExampleYaml("clickhouseuser.yaml", map[string]string{
		"metadata.name":                  userName,
		"spec.project":                   cfg.Project,
		"spec.serviceName":               chName,
		"spec.connInfoSecretTarget.name": userName,
		// Remove 'username' from the initial yaml
		"spec.username": "REMOVE",
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	require.NoError(t, s.Apply(yml))

	user := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.GetRunning(user, userName))

	secret, err := s.GetSecret(userName)
	require.NoError(t, err)
	require.NotEmpty(t, secret.Data["PASSWORD"])
	require.NotEmpty(t, secret.Data["CLICKHOUSEUSER_PASSWORD"])

	origPassword := append([]byte(nil), secret.Data["PASSWORD"]...)
	origPasswordPrefixed := append([]byte(nil), secret.Data["CLICKHOUSEUSER_PASSWORD"]...)

	require.NoError(t, pingClickhouse(
		ctx,
		secret.Data["CLICKHOUSEUSER_HOST"],
		secret.Data["CLICKHOUSEUSER_PORT"],
		secret.Data["CLICKHOUSEUSER_USERNAME"],
		secret.Data["CLICKHOUSEUSER_PASSWORD"],
	))

	// Simulate a "legacy" secret that contains extra keys
	secret.Data["EXTRA"] = []byte("keep-me")
	require.NoError(t, k8sClient.Update(ctx, secret))

	// WHEN
	// Trigger Update in operator-managed mode, where the controller omits password keys from SecretDetails
	ymlUpdated, err := loadExampleYaml("clickhouseuser.yaml", map[string]string{
		"metadata.name":                             userName,
		"spec.project":                              cfg.Project,
		"spec.serviceName":                          chName,
		"spec.connInfoSecretTarget.name":            userName,
		"spec.connInfoSecretTarget.labels.baz":      "egg-updated",
		"spec.connInfoSecretTarget.annotations.foo": "bar-updated",
		// Keep operator-managed mode by leaving connInfoSecretSource unset, and keep username unset
		"spec.username": "REMOVE",
	})
	require.NoError(t, err)

	require.NoError(t, s.Apply(ymlUpdated))
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	// Password keys should still be present and unchanged
	updatedSecret, err := s.GetSecret(userName)
	require.NoError(t, err)
	require.Equal(t, origPassword, updatedSecret.Data["PASSWORD"])
	require.Equal(t, origPasswordPrefixed, updatedSecret.Data["CLICKHOUSEUSER_PASSWORD"])
	require.Equal(t, []byte("keep-me"), updatedSecret.Data["EXTRA"])

	require.NoError(t, pingClickhouse(
		ctx,
		updatedSecret.Data["CLICKHOUSEUSER_HOST"],
		updatedSecret.Data["CLICKHOUSEUSER_PORT"],
		updatedSecret.Data["CLICKHOUSEUSER_USERNAME"],
		updatedSecret.Data["CLICKHOUSEUSER_PASSWORD"],
	))

	require.NoError(t, s.Delete(user, func() error {
		_, err = getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
		return err
	}))
}

// TestClickhouseUserDeletionPolicyOrphan verifies that ClickhouseUser with
// deletion-policy Orphan keeps the Aiven user after the Kubernetes resource is deleted.
func TestClickhouseUserDeletionPolicyOrphan(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	ch, release, err := sharedResources.AcquireClickhouse(ctx)
	require.NoError(t, err)
	defer release()

	chName := ch.GetName()
	userName := randName("chu-orphan")
	secretName := randName("chu-orphan-secret")

	// Orphan deletion policy is set via annotation on the ClickhouseUser.
	yml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: %s
  annotations:
    controllers.aiven.io/deletion-policy: Orphan
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: %s

  project: %s
  serviceName: %s
`, userName, secretName, cfg.Project, chName)

	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	require.NoError(t, s.Apply(yml))

	user := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.GetRunning(user, userName))

	userAvn, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
	require.NoError(t, err)
	assert.Equal(t, userName, userAvn.Name)

	secret, err := s.GetSecret(secretName)
	require.NoError(t, err)
	require.NotEmpty(t, secret.Data["HOST"])
	require.NotEmpty(t, secret.Data["PORT"])
	require.NotEmpty(t, secret.Data["USERNAME"])
	require.NotEmpty(t, secret.Data["PASSWORD"])

	host := append([]byte(nil), secret.Data["HOST"]...)
	port := append([]byte(nil), secret.Data["PORT"]...)
	username := append([]byte(nil), secret.Data["USERNAME"]...)
	password := append([]byte(nil), secret.Data["PASSWORD"]...)

	require.NoError(t, pingClickhouse(ctx, host, port, username, password))

	// WHEN
	// Delete the ClickhouseUser with deletion-policy Orphan.
	require.NoError(t, s.Delete(user, func() error {
		_, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
		return err
	}))

	// THEN
	// The Kubernetes resource should be gone.
	deleted := new(v1alpha1.ClickhouseUser)
	err = k8sClient.Get(ctx, types.NamespacedName{Name: userName, Namespace: defaultNamespace}, deleted)
	require.True(t, isNotFound(err))

	// The Aiven user should still exist because of Orphan policy.
	userAvnAfter, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
	require.NoError(t, err)
	require.Equal(t, userAvn.Uuid, userAvnAfter.Uuid)

	// And the saved credentials should still allow connecting to ClickHouse.
	require.NoError(t, pingClickhouse(ctx, host, port, username, password))
}

func pingClickhouse[T string | []byte](ctx context.Context, host, port, username, password T) error {
	var (
		lastErr     error
		maxAttempts = 3
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := func() error {
			conn, err := clickhouse.Open(&clickhouse.Options{
				Protocol: clickhouse.Native,
				Addr:     []string{fmt.Sprintf("%s:%s", host, port)},
				Auth:     clickhouse.Auth{Username: string(username), Password: string(password)},
				TLS:      &tls.Config{InsecureSkipVerify: true},
			})
			if err != nil {
				return err
			}
			defer conn.Close()

			return conn.Ping(ctx)
		}()

		if err == nil {
			return nil
		}

		lastErr = err

		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

func getClickHouseUserByID(ctx context.Context, avnGen avngen.Client, project, serviceName, userID string) (*clickhouse2.UserOut, error) {
	list, err := avnGen.ServiceClickHouseUserList(ctx, project, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get Clickhouse user by ID %s: %w", userID, err)
	}

	for _, u := range list {
		if u.Uuid == userID {
			return &u, nil
		}
	}
	return nil, controllers.NewNotFound(fmt.Sprintf("ClickHouse user %s not found", userID))
}

// TestClickhouseUserCustomCredentials tests ClickhouseUser credential management scenarios
func TestClickhouseUserCustomCredentials(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	ch, release, err := sharedResources.AcquireClickhouse(ctx)
	require.NoError(t, err)
	defer release()

	chName := ch.GetName()
	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	chAvn, err := avnGen.ServiceGet(ctx, cfg.Project, chName)
	require.NoError(t, err)
	assert.Equal(t, chAvn.ServiceName, ch.GetName())
	assert.Equal(t, serviceRunningState, ch.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, chAvn.State)

	t.Run("CustomPasswordSource", func(t *testing.T) {
		// tests ClickhouseUser creation with a predefined password from connInfoSecretSource
		// verifies that the operator correctly reads a password from a source secret
		// and applies it to the ClickhouseUser
		userName := randName("chu-custom-pass")
		yml := getClickhouseUserWithSourceSecretYaml(cfg.Project, chName, userName, cfg.PrimaryCloudName)

		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ClickhouseUser)
		require.NoError(t, s.GetRunning(user, userName))

		userAvn, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
		require.NoError(t, err)
		assert.Equal(t, userName, user.GetName())
		assert.Equal(t, userName, userAvn.Name)
		assert.Equal(t, chName, user.Spec.ServiceName)

		secretName := fmt.Sprintf("my-clickhouse-user-secret-%s", userName)
		secret, err := s.GetSecret(secretName)
		require.NoError(t, err)
		assert.NotEmpty(t, secret.Data["HOST"])
		assert.NotEmpty(t, secret.Data["PORT"])
		assert.NotEmpty(t, secret.Data["USERNAME"])
		assert.NotEmpty(t, secret.Data["PASSWORD"])
		assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_HOST"])
		assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_PORT"])
		assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_USERNAME"])
		assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_PASSWORD"])

		// verify the password matches predefined value from source secret
		actualPassword := string(secret.Data["PASSWORD"])
		assert.Equal(t, "MyCustomClickhousePassword123!", actualPassword, "Password should match predefined value")

		assert.Equal(t, map[string]string{"test": "predefined-password"}, secret.Annotations)
		assert.Equal(t, map[string]string{"type": "custom-password"}, secret.Labels)

		assert.NoError(t, pingClickhouse(
			ctx,
			secret.Data["CLICKHOUSEUSER_HOST"],
			secret.Data["CLICKHOUSEUSER_PORT"],
			secret.Data["CLICKHOUSEUSER_USERNAME"],
			secret.Data["CLICKHOUSEUSER_PASSWORD"],
		))

		assert.NoError(t, s.Delete(user, func() error {
			_, err = getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
			return err
		}))
	})

	t.Run("CrossNamespacePasswordSource", func(t *testing.T) {
		// tests ClickhouseUser creation with a password from a different namespace
		userName := randName("chu-cross-ns")
		yml := getClickhouseUserWithCrossNamespaceSecretYaml(cfg.Project, chName, userName, cfg.PrimaryCloudName)

		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ClickhouseUser)
		require.NoError(t, s.GetRunning(user, userName))

		userAvn, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
		require.NoError(t, err)
		assert.Equal(t, userName, user.GetName())
		assert.Equal(t, userName, userAvn.Name)
		assert.Equal(t, chName, user.Spec.ServiceName)

		secretName := fmt.Sprintf("my-clickhouse-user-secret-%s", userName)
		secret, err := s.GetSecret(secretName)
		require.NoError(t, err)
		assert.NotEmpty(t, secret.Data["HOST"])
		assert.NotEmpty(t, secret.Data["PORT"])
		assert.NotEmpty(t, secret.Data["USERNAME"])
		assert.NotEmpty(t, secret.Data["PASSWORD"])

		// verify the password matches cross-namespace secret value
		actualPassword := string(secret.Data["PASSWORD"])
		assert.Equal(t, "CrossNamespacePassword456!", actualPassword, "Password should match cross-namespace secret value")

		assert.Equal(t, map[string]string{"test": "cross-namespace-password"}, secret.Annotations)
		assert.Equal(t, map[string]string{"type": "cross-namespace"}, secret.Labels)

		// test ClickHouse connection with cross-namespace password
		assert.NoError(t, pingClickhouse(
			ctx,
			secret.Data["CLICKHOUSEUSER_HOST"],
			secret.Data["CLICKHOUSEUSER_PORT"],
			secret.Data["CLICKHOUSEUSER_USERNAME"],
			secret.Data["CLICKHOUSEUSER_PASSWORD"],
		))

		assert.NoError(t, s.Delete(user, func() error {
			_, err = getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
			return err
		}))
	})

	t.Run("InvalidPasswordValidation", func(t *testing.T) {
		// tests validation of invalid passwords in connInfoSecretSource
		userName := randName("chu-invalid-pass")
		yml := getClickhouseUserWithInvalidPasswordYaml(cfg.Project, chName, userName, cfg.PrimaryCloudName)

		require.NoError(t, s.Apply(yml))

		// attempt to wait for ClickhouseUser to become ready (should fail)
		user := new(v1alpha1.ClickhouseUser)
		err := s.GetRunning(user, userName)

		require.Error(t, err, "ClickhouseUser should fail to be created with invalid password")
	})

	t.Run("MissingSecretValidation", func(t *testing.T) {
		// tests validation when connInfoSecretSource references a non-existent secret
		userName := randName("chu-missing-secret")
		yml := getClickhouseUserWithMissingSecretYaml(cfg.Project, chName, userName, cfg.PrimaryCloudName)

		require.NoError(t, s.Apply(yml))

		// attempt to wait for ClickhouseUser to become ready (should fail)
		user := new(v1alpha1.ClickhouseUser)
		err := s.GetRunning(user, userName)

		require.Error(t, err, "ClickhouseUser should fail to be created with missing secret")
	})

	t.Run("MissingPasswordKeyValidation", func(t *testing.T) {
		// tests validation when connInfoSecretSource references a secret without the specified password key
		userName := randName("chu-missing-key")
		yml := getClickhouseUserWithMissingPasswordKeyYaml(cfg.Project, chName, userName, cfg.PrimaryCloudName)

		require.NoError(t, s.Apply(yml))

		// attempt to wait for ClickhouseUser to become ready (should fail)
		user := new(v1alpha1.ClickhouseUser)
		err = s.GetRunning(user, userName)
		require.Error(t, err, "ClickhouseUser should fail to be created with missing password key")
	})

	t.Run("DocumentationExample", func(t *testing.T) {
		// tests the exact ClickhouseUser configuration from the example
		userName := "my-clickhouse-user"

		yml, err := loadExampleYaml("clickhouseuser.custom_credentials.yaml", map[string]string{
			"doc[1].metadata.name":    userName,
			"doc[1].spec.project":     cfg.Project,
			"doc[1].spec.serviceName": chName,
		})
		require.NoError(t, err)

		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ClickhouseUser)
		require.NoError(t, s.GetRunning(user, userName))

		userAvn, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
		require.NoError(t, err)
		assert.Equal(t, "example-username", userAvn.Name) // username field from the example
		assert.Equal(t, chName, user.Spec.ServiceName)

		secret, err := s.GetSecret("clickhouse-user-secret")
		require.NoError(t, err)
		assert.NotEmpty(t, secret.Data["MY_CLICKHOUSE_PREFIX_HOST"])
		assert.NotEmpty(t, secret.Data["MY_CLICKHOUSE_PREFIX_PORT"])
		assert.NotEmpty(t, secret.Data["MY_CLICKHOUSE_PREFIX_USERNAME"])
		assert.NotEmpty(t, secret.Data["MY_CLICKHOUSE_PREFIX_PASSWORD"])

		// verify the password matches the exact value from the documentation example
		actualPassword := string(secret.Data["MY_CLICKHOUSE_PREFIX_PASSWORD"])
		assert.Equal(t, "MyCustomPassword123!", actualPassword, "Password should match documentation example")

		// verify annotations and labels from the documentation example
		assert.Equal(t, map[string]string{"foo": "bar"}, secret.Annotations)
		assert.Equal(t, map[string]string{"baz": "egg"}, secret.Labels)

		// test ClickHouse connection with the password from documentation example
		assert.NoError(t, pingClickhouse(
			ctx,
			secret.Data["MY_CLICKHOUSE_PREFIX_HOST"],
			secret.Data["MY_CLICKHOUSE_PREFIX_PORT"],
			secret.Data["MY_CLICKHOUSE_PREFIX_USERNAME"],
			secret.Data["MY_CLICKHOUSE_PREFIX_PASSWORD"],
		))

		assert.NoError(t, s.Delete(user, func() error {
			_, err = getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
			return err
		}))
	})
}

// TestClickhouseUserBuiltInAvnadmin tests the built-in 'avnadmin' user with custom password.
// This test creates its own Clickhouse instance because it modifies the avnadmin password,
// which would interfere with other tests using shared Clickhouse.
func TestClickhouseUserBuiltInAvnadmin(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	chName := randName("clickhouse-avnadmin")

	yml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: %s

  project: %s
  cloudName: %s
  plan: startup-v2-16
`, chName, chName, cfg.Project, cfg.PrimaryCloudName)

	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	require.NoError(t, s.Apply(yml))

	ch := new(v1alpha1.Clickhouse)
	require.NoError(t, s.GetRunning(ch, chName))

	// WHEN
	userName := "avnadmin" // built-in username
	userYml := getClickhouseUserWithBuiltInUserYaml(cfg.Project, chName, userName, cfg.PrimaryCloudName)
	require.NoError(t, s.Apply(userYml))

	user := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	userAvn, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
	require.NoError(t, err)
	assert.Equal(t, userName, user.GetName())
	assert.Equal(t, userName, userAvn.Name)
	assert.Equal(t, chName, user.Spec.ServiceName)

	secretName := fmt.Sprintf("my-clickhouse-builtin-user-secret-%s", userName)
	secret, err := s.GetSecret(secretName)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USERNAME"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_HOST"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_PORT"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_USERNAME"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSEUSER_PASSWORD"])

	// verify the password matches predefined value for built-in user
	actualPassword := string(secret.Data["PASSWORD"])
	assert.Equal(t, "BuiltInUserPassword123!", actualPassword, "Password should match predefined value for built-in user")

	assert.Equal(t, map[string]string{"test": "builtin-user"}, secret.Annotations)
	assert.Equal(t, map[string]string{"type": "built-in-user"}, secret.Labels)

	// verify that the built-in user works the same as regular users
	assert.NoError(t, pingClickhouse(
		ctx,
		secret.Data["CLICKHOUSEUSER_HOST"],
		secret.Data["CLICKHOUSEUSER_PORT"],
		secret.Data["CLICKHOUSEUSER_USERNAME"],
		secret.Data["CLICKHOUSEUSER_PASSWORD"],
	))

	// the user should be successfully deleted from Kubernetes, but the isBuiltInUser logic
	// should prevent actual deletion from Aiven
	assert.NoError(t, s.Delete(user, func() error {
		_, err = getClickHouseUserByID(ctx, avnGen, cfg.Project, chName, user.Status.UUID)
		// for built-in users, the user would still exist in Aiven after "deletion"
		if isNotFound(err) {
			return nil
		}
		return err
	}))
}

func getClickhouseUserWithBuiltInUserYaml(project, chName, userName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: builtin-user-password-secret-%[3]s
data:
  PASSWORD: QnVpbHRJblVzZXJQYXNzd29yZDEyMyE= # BuiltInUserPassword123! base64 encoded # gitleaks:allow
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
    name: my-clickhouse-builtin-user-secret-%[3]s
    annotations:
      test: builtin-user
    labels:
      type: built-in-user

  connInfoSecretSource:
    name: builtin-user-password-secret-%[3]s
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, chName, userName, cloudName)
}

func getClickhouseUserWithSourceSecretYaml(project, chName, userName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: predefined-clickhouse-password-secret-%[3]s
data:
  PASSWORD: TXlDdXN0b21DbGlja2hvdXNlUGFzc3dvcmQxMjMh # MyCustomClickhousePassword123! base64 encoded # gitleaks:allow
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
    name: my-clickhouse-user-secret-%[3]s
    annotations:
      test: predefined-password
    labels:
      type: custom-password

  connInfoSecretSource:
    name: predefined-clickhouse-password-secret-%[3]s
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, chName, userName, cloudName)
}

func getClickhouseUserWithCrossNamespaceSecretYaml(project, chName, userName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: cross-namespace-password-secret-%[3]s
  namespace: kube-system
data:
  PASSWORD: Q3Jvc3NOYW1lc3BhY2VQYXNzd29yZDQ1NiE= # CrossNamespacePassword456! base64 encoded # gitleaks:allow
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
    name: my-clickhouse-user-secret-%[3]s
    annotations:
      test: cross-namespace-password
    labels:
      type: cross-namespace

  connInfoSecretSource:
    name: cross-namespace-password-secret-%[3]s
    namespace: kube-system
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, chName, userName, cloudName)
}

func getClickhouseUserWithInvalidPasswordYaml(project, chName, userName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: invalid-clickhouse-password-secret-%[3]s
data:
  PASSWORD: c2hvcnQ= # short - invalid password length (5 chars) # gitleaks:allow
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
    name: invalid-password-secret-target
    annotations:
      test: invalid-password-validation
    labels:
      type: validation-test

  connInfoSecretSource:
    name: invalid-clickhouse-password-secret-%[3]s
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, chName, userName, cloudName)
}

func getClickhouseUserWithMissingSecretYaml(project, chName, userName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: missing-secret-target
    annotations:
      test: missing-secret-validation
    labels:
      type: validation-test

  connInfoSecretSource:
    name: nonexistent-secret
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, chName, userName, cloudName)
}

func getClickhouseUserWithMissingPasswordKeyYaml(project, chName, userName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: missing-key-clickhouse-secret-%[3]s
data:
  WRONG_KEY: VmFsaWRQYXNzd29yZDEyMyE= # ValidPassword123! base64 encoded # gitleaks:allow
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
    name: missing-key-secret-target
    annotations:
      test: missing-key-validation
    labels:
      type: validation-test

  connInfoSecretSource:
    name: missing-key-clickhouse-secret-%[3]s
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, chName, userName, cloudName)
}
