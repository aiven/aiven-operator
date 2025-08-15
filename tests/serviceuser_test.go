//go:build serviceuser

package tests

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getServiceUserKafkaYaml(project, kafkaName, userName, cloudName string) string {
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
  plan: startup-2

  userConfig:
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
    name: my-service-user-secret
    annotations:
      foo: bar
    labels:
      baz: egg

  project: %[1]s
  serviceName: %[2]s
`, project, kafkaName, userName, cloudName)
}

// TestServiceUserKafka kafka with sasl enabled changes its port to expose
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
	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, kafkaName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, user.GetName())
	assert.Equal(t, userName, userAvn.Username)
	assert.Equal(t, kafkaName, user.Spec.ServiceName)

	// Validates Secret
	secret, err := s.GetSecret("my-service-user-secret")
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
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[4]s
  plan: startup-4

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
    annotations:
      foo: bar
    labels:
      baz: egg

  project: %[1]s
  serviceName: %[2]s
`, project, pgName, userName, cloudName)
}

// TestServiceUserPg same as TestServiceUserKafka but runs with pg and expects the default port to be exposed
func TestServiceUserPg(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()
	pgName := randName("connection-pool")
	userName := randName("connection-pool")
	yml := getServiceUserPgYaml(cfg.Project, pgName, userName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	// Validates PostgreSQL
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Equal(t, serviceRunningState, pg.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)
	assert.Equal(t, pgAvn.Plan, pg.Spec.Plan)
	assert.Equal(t, pgAvn.CloudName, pg.Spec.CloudName)

	// Validates ServiceUser
	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, pgName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, user.GetName())
	assert.Equal(t, userName, userAvn.Username)
	assert.Equal(t, pgName, user.Spec.ServiceName)

	// Validates Secret
	secret, err := s.GetSecret("my-service-user-secret")
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
	pgName := randName("serviceuser-scenarios")
	s := NewSession(ctx, k8sClient)

	defer s.Destroy(t)

	pgYaml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: startup-4
`, cfg.Project, pgName, cfg.PrimaryCloudName)

	require.NoError(t, s.Apply(pgYaml))

	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

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

		userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, pgName, userName)
		require.NoError(t, err)
		assert.Equal(t, userName, user.GetName())
		assert.Equal(t, userName, userAvn.Username)
		assert.Equal(t, pgName, user.Spec.ServiceName)

		secret, err := s.GetSecret("my-service-user-secret")
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

	t.Run("AvnadminPasswordReset", func(t *testing.T) {
		// Tests password reset functionality for the built-in 'avnadmin' user.
		// Verifies that the operator can modify credentials for system user and that
		// built-in users persist after ServiceUser resource deletion.
		yml := getServiceUserAvnadminResetYaml(cfg.Project, pgName, cfg.PrimaryCloudName)

		require.NoError(t, s.Apply(yml))

		user := new(v1alpha1.ServiceUser)
		require.NoError(t, s.GetRunning(user, "avnadmin"))

		userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, pgName, "avnadmin")
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
	})
}

func getServiceUserWithSourceSecretYaml(project, pgName, userName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: predefined-password-secret
data:
  PASSWORD: TXlDdXN0b21QYXNzd29yZDEyMyE= # MyCustomPassword123! base64 encoded # gitleaks:allow
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
  cloudName: %[4]s
  plan: startup-4

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
    annotations:
      test: predefined-password
    labels:
      type: custom-password

  connInfoSecretSource:
    name: predefined-password-secret
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, pgName, userName, cloudName)
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
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: startup-4

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
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: empty-password-secret
data:
  PASSWORD: "" # Empty password - this should trigger validation error
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
  cloudName: %[4]s
  plan: startup-4

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
    name: empty-password-secret-target
    annotations:
      test: empty-password-validation
    labels:
      type: validation-test

  connInfoSecretSource:
    name: empty-password-secret
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, pgName, userName, cloudName)
}
