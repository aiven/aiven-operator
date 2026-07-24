//go:build postgresql

package tests

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/avast/retry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getServiceUserKafkaYaml(project, kafkaName, userName, cloudName string) string {
	// secret name based on userName to avoid conflicts
	secretName := userName + "-secret"
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[4]s
  plan: startup-4

  userConfig:
    schema_registry: true
    kafka_authentication_methods:
      sasl: true
      certificate: false
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
    annotations:
      foo: bar
    labels:
      baz: egg

  project: %[1]s
  serviceName: %[2]s
`, project, kafkaName, userName, cloudName, secretName)
}

// TestServiceUserKafka verifies ServiceUser behavior with Kafka's SASL authentication.
// It creates its own Kafka instance with SASL enabled and certificate
// authentication disabled to test SASL-specific port behavior.
func TestServiceUserKafka(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()
	kafkaName := randName("service-user")
	userName := randName("service-user")
	yml := getServiceUserKafkaYaml(cfg.Project, kafkaName, userName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	kafka := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(kafka, kafkaName))

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	// Validates Kafka
	kafkaAvn, err := avnGen.ServiceGet(ctx, cfg.Project, kafkaName)
	require.NoError(t, err)
	assert.Equal(t, kafkaAvn.ServiceName, kafka.GetName())
	assert.Equal(t, serviceRunningState, kafka.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, kafkaAvn.State)
	assert.Equal(t, kafkaAvn.Plan, kafka.Spec.Plan)
	assert.Equal(t, kafkaAvn.CloudName, kafka.Spec.CloudName)

	// Validates ServiceUser
	userAvn, err := getServiceUserWithRetry(ctx, avnGen, cfg.Project, kafkaName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, user.GetName())
	assert.Equal(t, userName, userAvn.Username)
	assert.Equal(t, kafkaName, user.Spec.ServiceName)

	// Validates Secret
	secret, err := s.GetSecret(userName + "-secret")
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["SERVICEUSER_HOST"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_PORT"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_USERNAME"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_PASSWORD"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_CA_CERT"])
	assert.Contains(t, secret.Data, "SERVICEUSER_ACCESS_CERT")
	assert.Contains(t, secret.Data, "SERVICEUSER_ACCESS_KEY")
	assert.Equal(t, map[string]string{"foo": "bar"}, secret.Annotations)
	assert.Equal(t, map[string]string{"baz": "egg"}, secret.Labels)

	var kafkaSaslComponent *service.ComponentOut
	var schemaRegistryComponent *service.ComponentOut
	for i := range kafkaAvn.Components {
		c := &kafkaAvn.Components[i]
		switch {
		case c.Component == "kafka" && c.KafkaAuthenticationMethod == service.KafkaAuthenticationMethodTypeSasl:
			kafkaSaslComponent = c
		case c.Component == "schema_registry":
			schemaRegistryComponent = c
		}
	}

	require.NotNil(t, kafkaSaslComponent)
	assert.NotEmpty(t, secret.Data["SERVICEUSER_SASL_HOST"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_SASL_PORT"])
	assert.Equal(t, kafkaSaslComponent.Host, string(secret.Data["SERVICEUSER_SASL_HOST"]))
	assert.Equal(t, strconv.Itoa(kafkaSaslComponent.Port), string(secret.Data["SERVICEUSER_SASL_PORT"]))

	require.NotNil(t, schemaRegistryComponent)
	assert.NotEmpty(t, secret.Data["SERVICEUSER_SCHEMA_REGISTRY_HOST"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_SCHEMA_REGISTRY_PORT"])
	assert.Equal(t, schemaRegistryComponent.Host, string(secret.Data["SERVICEUSER_SCHEMA_REGISTRY_HOST"]))
	assert.Equal(t, strconv.Itoa(schemaRegistryComponent.Port), string(secret.Data["SERVICEUSER_SCHEMA_REGISTRY_PORT"]))

	kafkaForUpdate := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(kafkaForUpdate, kafkaName))
	kafkaForUpdate.Spec.UserConfig.SchemaRegistry = anyPointer(false)
	require.NoError(t, k8sClient.Update(ctx, kafkaForUpdate))

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		updatedKafkaAvn, getErr := avnGen.ServiceGet(ctx, cfg.Project, kafkaName)
		require.NoError(collect, getErr)

		enabled, ok := updatedKafkaAvn.UserConfig["schema_registry"].(bool)
		require.True(collect, ok)
		assert.False(collect, enabled)
	}, 10*time.Minute, 10*time.Second, "Schema Registry should be disabled in Aiven")

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		updatedSecret, getErr := s.GetSecret(userName + "-secret")
		require.NoError(collect, getErr)
		assert.Contains(collect, updatedSecret.Data, "SERVICEUSER_SCHEMA_REGISTRY_HOST")
		assert.Contains(collect, updatedSecret.Data, "SERVICEUSER_SCHEMA_REGISTRY_PORT")
		assert.Empty(collect, updatedSecret.Data["SERVICEUSER_SCHEMA_REGISTRY_HOST"])
		assert.Empty(collect, updatedSecret.Data["SERVICEUSER_SCHEMA_REGISTRY_PORT"])
	}, 10*time.Minute, 10*time.Second, "ServiceUser secret should clear disabled Schema Registry endpoint keys")

	// This kafka has sasl enabled and cert auth disabled.
	// Which means that the port is not the same as in uri params.
	strPort := string(secret.Data["SERVICEUSER_PORT"])
	assert.NotEmpty(t, kafkaAvn.ServiceUriParams["port"])
	assert.NotEqual(t, kafkaAvn.ServiceUriParams["port"], strPort)

	intPort, err := strconv.ParseInt(strPort, 10, 32)
	require.NoError(t, err)
	assert.Positive(t, intPort)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, pool is destroyed in Aiven. No service — no pool. No pool — no pool.
	// And we make sure that the controller can delete db itself
	assert.NoError(t, s.Delete(user, func() error {
		_, err = avnGen.ServiceUserGet(ctx, cfg.Project, kafkaName, userName)
		return err
	}))
}

func getServiceUserPgYaml(project, pgName, userName, cloudName string) string {
	// secret name based on userName to avoid conflicts
	secretName := userName + "-secret"
	return fmt.Sprintf(`
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
    annotations:
      foo: bar
    labels:
      baz: egg

  project: %[1]s
  serviceName: %[2]s
`, project, pgName, userName, cloudName, secretName)
}

// TestServiceUserPg same as TestServiceUserKafka but runs with pg and expects the default port to be exposed
func TestServiceUserPg(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	pg, releasePG, err := sharedResources.AcquirePostgreSQL(ctx)
	require.NoError(t, err)
	defer releasePG()

	pgName := pg.GetName()
	userName := randName("connection-pool")

	yml := getServiceUserPgYaml(cfg.Project, pgName, userName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Equal(t, serviceRunningState, pg.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)

	// Validates ServiceUser
	userAvn, err := getServiceUserWithRetry(ctx, avnGen, cfg.Project, pgName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, user.GetName())
	assert.Equal(t, userName, userAvn.Username)
	assert.Equal(t, pgName, user.Spec.ServiceName)

	// Validates Secret
	secret, err := s.GetSecret(userName + "-secret")
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["SERVICEUSER_HOST"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_PORT"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_USERNAME"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_PASSWORD"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_CA_CERT"])
	assert.Contains(t, secret.Data, "SERVICEUSER_ACCESS_CERT")
	assert.Contains(t, secret.Data, "SERVICEUSER_ACCESS_KEY")
	assert.Equal(t, map[string]string{"foo": "bar"}, secret.Annotations)
	assert.Equal(t, map[string]string{"baz": "egg"}, secret.Labels)

	// Default port is exposed
	assert.NotEmpty(t, pgAvn.ServiceUriParams["port"])
	assert.Equal(t, pgAvn.ServiceUriParams["port"], string(secret.Data["SERVICEUSER_PORT"]))

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, pool is destroyed in Aiven. No service — no pool. No pool — no pool.
	// And we make sure that the controller can delete db itself
	assert.NoError(t, s.Delete(user, func() error {
		_, err = avnGen.ServiceUserGet(ctx, cfg.Project, pgName, userName)
		return err
	}))
}

// TestServiceUserCustomCredentials tests ServiceUser credential management scenarios using a shared PG instance.
func TestServiceUserCustomCredentials(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	pg, releasePG, err := sharedResources.AcquirePostgreSQL(ctx)
	require.NoError(t, err)
	defer releasePG()

	pgName := pg.GetName()
	s := NewSession(ctx, k8sClient)

	defer s.Destroy(t)

	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Equal(t, serviceRunningState, pg.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)

	t.Run("CustomPasswordSource", func(t *testing.T) {
		// Tests ServiceUser creation with a predefined password from connInfoSecretSource.
		// Verifies that the operator correctly reads a password from a source secret
		// and applies it to the ServiceUser.
		userName := randName("su-custom-pass")
		yml := getServiceUserWithSourceSecretYaml(cfg.Project, pgName, userName, cfg.PrimaryCloudName)

		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(user, userName))

		userAvn, err := getServiceUserWithRetry(ctx, avnGen, cfg.Project, pgName, userName)
		require.NoError(t, err)
		assert.Equal(t, userName, user.GetName())
		assert.Equal(t, userName, userAvn.Username)
		assert.Equal(t, pgName, user.Spec.ServiceName)

		secret, err := s.GetSecret(userName + "-secret")
		require.NoError(t, err)
		assert.NotEmpty(t, secret.Data["SERVICEUSER_HOST"])
		assert.NotEmpty(t, secret.Data["SERVICEUSER_PORT"])
		assert.NotEmpty(t, secret.Data["SERVICEUSER_USERNAME"])
		assert.NotEmpty(t, secret.Data["SERVICEUSER_PASSWORD"])
		assert.NotEmpty(t, secret.Data["SERVICEUSER_CA_CERT"])

		// verify the password matches predefined value from source secret
		actualPassword := string(secret.Data["SERVICEUSER_PASSWORD"])
		assert.Equal(t, "MyCustomPassword123!", actualPassword, "Password should match predefined value")

		assert.Equal(t, map[string]string{"test": "predefined-password"}, secret.Annotations)
		assert.Equal(t, map[string]string{"type": "custom-password"}, secret.Labels)

		assert.NoError(t, s.Delete(user, func() error {
			_, err = avnGen.ServiceUserGet(ctx, cfg.Project, pgName, userName)
			return err
		}))
	})

	t.Run("EmptyPasswordValidation", func(t *testing.T) {
		// Tests validation of empty passwords in connInfoSecretSource.
		// Verifies that the operator properly validates password requirements
		// and fails ServiceUser creation when an empty password is provided.
		userName := randName("su-empty-pass")
		yml := getServiceUserWithEmptyPasswordYaml(cfg.Project, pgName, userName, cfg.PrimaryCloudName)

		require.NoError(t, s.Apply(yml))

		// attempt to wait for ServiceUser to become ready (should fail)
		user := new(v1alpha1.ServiceUser)
		err := s.GetRunning(user, userName)

		// Verify that ServiceUser creation failed
		require.Error(t, err, "ServiceUser should fail to be created with empty password")
		require.Contains(t, err.Error(), "password length must be between 8 and 256 characters")
	})
}

// TestServiceUserSpecUsername tests spec.username decoupling the Aiven username from the resource name.
func TestServiceUserSpecUsername(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	pg, releasePG, err := sharedResources.AcquirePostgreSQL(ctx)
	require.NoError(t, err)
	defer releasePG()

	pgName := pg.GetName()
	s := NewSession(ctx, k8sClient)

	defer s.Destroy(t)

	t.Run("UsernameOverride", func(t *testing.T) {
		// Manages an Aiven user whose name contains underscores, which is not a valid Kubernetes object name.
		resourceName := randName("su-override")
		username := strings.ReplaceAll(resourceName, "-", "_")

		require.NoError(t, s.Apply(getServiceUserSpecUsernameYaml(cfg.Project, pgName, resourceName, username)))

		user := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(user, resourceName))

		// The user exists at Aiven under the override, not the resource name
		userAvn, err := getServiceUserWithRetry(ctx, avnGen, cfg.Project, pgName, username)
		require.NoError(t, err)
		assert.Equal(t, username, userAvn.Username)
		assert.Equal(t, resourceName, user.GetName())

		// The secret keeps the resource name; SERVICEUSER_USERNAME carries the override
		secret, err := s.GetSecret(resourceName + "-secret")
		require.NoError(t, err)
		assert.Equal(t, username, string(secret.Data["SERVICEUSER_USERNAME"]))
		assert.NotEmpty(t, secret.Data["SERVICEUSER_PASSWORD"])

		// Deletion removes the user at Aiven under the override name
		assert.NoError(t, s.Delete(user, func() error {
			_, err := avnGen.ServiceUserGet(ctx, cfg.Project, pgName, username)
			return err
		}))
	})

	t.Run("UsernameImmutability", func(t *testing.T) {
		resourceName := randName("su-immutable")
		username := strings.ReplaceAll(resourceName, "-", "_")

		require.NoError(t, s.Apply(getServiceUserSpecUsernameYaml(cfg.Project, pgName, resourceName, username)))

		user := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(user, resourceName))

		// Changing the value is rejected by the field-level rule
		err := s.Apply(getServiceUserSpecUsernameYaml(cfg.Project, pgName, resourceName, username+"_new"))
		require.ErrorContains(t, err, "username is immutable")

		// Unsetting the field is rejected by the spec-level presence rule.
		err = s.Apply(getServiceUserSpecUsernameYaml(cfg.Project, pgName, resourceName, ""))
		require.ErrorContains(t, err, "username can only be set during resource creation")

		// Setting the field on a resource created without it is rejected by the same presence rule.
		plainName := randName("su-plain")
		require.NoError(t, s.Apply(getServiceUserSpecUsernameYaml(cfg.Project, pgName, plainName, "")))

		plain := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(plain, plainName))

		err = s.Apply(getServiceUserSpecUsernameYaml(cfg.Project, pgName, plainName, plainName+"_override"))
		require.ErrorContains(t, err, "username can only be set during resource creation")
	})
}

func getServiceUserSpecUsernameYaml(project, pgName, resourceName, username string) string {
	usernameField := ""
	if username != "" {
		usernameField = "\n  username: " + username
	}
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: %[2]s-secret

  project: %[1]s
  serviceName: %[3]s%[4]s
`, project, resourceName, pgName, usernameField)
}

func TestServiceUserAvnadminPasswordReset(t *testing.T) {
	// Tests password reset functionality for the built-in 'avnadmin' user.
	// Verifies that the operator can modify credentials for system user and that
	// built-in users persist after ServiceUser resource deletion.
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	pg, releasePG, err := sharedResources.AcquirePostgreSQL(ctx)
	require.NoError(t, err)
	defer releasePG()

	pgName := pg.GetName()
	s := NewSession(ctx, k8sClient)

	defer s.Destroy(t)

	yml := getServiceUserAvnadminResetYaml(cfg.Project, pgName, cfg.PrimaryCloudName)

	require.NoError(t, s.Apply(yml))

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, "avnadmin"))

	userAvn, err := getServiceUserWithRetry(ctx, avnGen, cfg.Project, pgName, "avnadmin")
	require.NoError(t, err)
	assert.Equal(t, "avnadmin", user.GetName())
	assert.Equal(t, "avnadmin", userAvn.Username)
	assert.Equal(t, pgName, user.Spec.ServiceName)

	secret, err := s.GetSecret("my-avnadmin-secret")
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["SERVICEUSER_HOST"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_PORT"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_USERNAME"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_PASSWORD"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_CA_CERT"])

	actualUsernameInSecret := string(secret.Data["SERVICEUSER_USERNAME"])
	assert.Equal(t, "avnadmin", actualUsernameInSecret, "Username should be avnadmin")

	// verify the password was reset to custom value
	actualPassword := string(secret.Data["SERVICEUSER_PASSWORD"])
	assert.Equal(t, "NewAvnadminPassword999!", actualPassword, "Password should match our custom avnadmin password")

	assert.Equal(t, map[string]string{"test": "avnadmin-reset"}, secret.Annotations)
	assert.Equal(t, map[string]string{"type": "admin-password"}, secret.Labels)

	// validate that built-in users persist after ServiceUser deletion
	assert.NoError(t, s.Delete(user, func() error {
		// avnadmin user should still exist after ServiceUser deletion since it's a built-in user
		_, err = avnGen.ServiceUserGet(ctx, cfg.Project, pgName, "avnadmin")
		if err != nil {
			return fmt.Errorf("avnadmin user should still exist after ServiceUser deletion: %w", err)
		}
		return nil
	}))
}

func getServiceUserWithSourceSecretYaml(project, pgName, userName, cloudName string) string {
	// secret name based on userName to avoid conflicts
	secretName := userName + "-secret"
	sourceSecretName := userName + "-source-secret"
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[6]s
data:
  PASSWORD: TXlDdXN0b21QYXNzd29yZDEyMyE= # MyCustomPassword123! base64 encoded # gitleaks:allow
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
    annotations:
      test: predefined-password
    labels:
      type: custom-password

  connInfoSecretSource:
    name: %[6]s
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, pgName, userName, cloudName, secretName, sourceSecretName)
}

func getServiceUserAvnadminResetYaml(project, pgName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: avnadmin-new-password
data:
  PASSWORD: TmV3QXZuYWRtaW5QYXNzd29yZDk5OSE= # NewAvnadminPassword999! base64 encoded # gitleaks:allow
---

apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: avnadmin
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: my-avnadmin-secret
    annotations:
      test: avnadmin-reset
    labels:
      type: admin-password

  connInfoSecretSource:
    name: avnadmin-new-password
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, pgName, cloudName)
}

func getServiceUserWithEmptyPasswordYaml(project, pgName, userName, cloudName string) string {
	// secret name based on userName to avoid conflicts
	secretName := userName + "-secret"
	sourceSecretName := userName + "-source-secret"
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[6]s
data:
  PASSWORD: "" # Empty password - this should trigger validation error
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
    annotations:
      test: empty-password-validation
    labels:
      type: validation-test

  connInfoSecretSource:
    name: %[6]s
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, pgName, userName, cloudName, secretName, sourceSecretName)
}

func getServiceUserWithRetry(
	ctx context.Context,
	avnGen avngen.Client,
	project, serviceName, username string,
) (*service.ServiceUserGetOut, error) {
	var user *service.ServiceUserGetOut
	err := retry.Do(
		func() error {
			var retryErr error
			user, retryErr = avnGen.ServiceUserGet(ctx, project, serviceName, username)
			return retryErr
		},
		retry.RetryIf(isNotFound),
		retry.Attempts(3),
		retry.Delay(200*time.Millisecond),
	)

	return user, err
}
