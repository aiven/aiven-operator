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
	s := NewSession(ctx, k8sClient, cfg.Project)

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
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USERNAME"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["CA_CERT"])
	assert.Contains(t, secret.Data, "ACCESS_CERT")
	assert.Contains(t, secret.Data, "ACCESS_KEY")
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
	s := NewSession(ctx, k8sClient, cfg.Project)

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
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USERNAME"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["CA_CERT"])
	assert.Contains(t, secret.Data, "ACCESS_CERT")
	assert.Contains(t, secret.Data, "ACCESS_KEY")
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
	assert.Equal(t, pgAvn.ServiceUriParams["port"], string(secret.Data["PORT"]))

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, pool is destroyed in Aiven. No service — no pool. No pool — no pool.
	// And we make sure that the controller can delete db itself
	assert.NoError(t, s.Delete(user, func() error {
		_, err = avnGen.ServiceUserGet(ctx, cfg.Project, pgName, userName)
		return err
	}))
}

// TestServiceUserWithPredefinedPassword tests ServiceUser with connInfoSecretSource
func TestServiceUserWithPredefinedPassword(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()
	pgName := randName("su-predefined-pass")
	userName := randName("su-predefined-pass")
	yml := getServiceUserWithSourceSecretYaml(cfg.Project, pgName, userName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Equal(t, serviceRunningState, pg.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)

	// validates ServiceUser
	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, pgName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, user.GetName())
	assert.Equal(t, userName, userAvn.Username)
	assert.Equal(t, pgName, user.Spec.ServiceName)

	// validates Secret with predefined password
	secret, err := s.GetSecret("my-service-user-secret")
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USERNAME"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["CA_CERT"])

	// verify the password matches our predefined one
	actualPassword := string(secret.Data["PASSWORD"])
	assert.Equal(t, "MyCustomPassword123!", actualPassword, "Password should match predefined value")

	// verify custom annotations and labels
	assert.Equal(t, map[string]string{"test": "predefined-password"}, secret.Annotations)
	assert.Equal(t, map[string]string{"type": "custom-password"}, secret.Labels)

	// validates deletion
	assert.NoError(t, s.Delete(user, func() error {
		_, err = avnGen.ServiceUserGet(ctx, cfg.Project, pgName, userName)
		return err
	}))
}

// TestServiceUserWithCustomUsername tests ServiceUser with custom username (resource name = username)
func TestServiceUserWithCustomUsername(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()
	pgName := randName("su-custom-user")
	customResourceName := "my_custom_username_" + randName("")
	yml := getServiceUserWithCustomUsernameYaml(cfg.Project, pgName, customResourceName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, customResourceName))

	// THEN
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Equal(t, serviceRunningState, pg.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)

	// validates ServiceUser - resource name should equal username
	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, pgName, customResourceName)
	require.NoError(t, err)
	assert.Equal(t, customResourceName, user.GetName())
	assert.Equal(t, customResourceName, userAvn.Username)
	assert.Equal(t, pgName, user.Spec.ServiceName)

	// validates Secret with custom username and predefined password
	secret, err := s.GetSecret("my-custom-user-secret")
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USERNAME"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["CA_CERT"])

	// verify the username matches our custom resource name
	actualUsernameInSecret := string(secret.Data["USERNAME"])
	assert.Equal(t, customResourceName, actualUsernameInSecret, "Username should match custom resource name")

	// verify the password matches our predefined one
	actualPassword := string(secret.Data["PASSWORD"])
	assert.Equal(t, "CustomUserPass789!", actualPassword, "Password should match predefined value")

	// verify custom annotations and labels
	assert.Equal(t, map[string]string{"test": "custom-username"}, secret.Annotations)
	assert.Equal(t, map[string]string{"type": "custom-username"}, secret.Labels)

	assert.NoError(t, s.Delete(user, func() error {
		_, err = avnGen.ServiceUserGet(ctx, cfg.Project, pgName, customResourceName)
		return err
	}))
}

// TestServiceUserAvnadminPasswordReset tests reassigning password for default avnadmin user
func TestServiceUserAvnadminPasswordReset(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()
	pgName := randName("su-avnadmin-reset")
	yml := getServiceUserAvnadminResetYaml(cfg.Project, pgName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, "avnadmin"))

	// THEN
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Equal(t, serviceRunningState, pg.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)

	// validates ServiceUser - resource name should be avnadmin
	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, pgName, "avnadmin")
	require.NoError(t, err)
	assert.Equal(t, "avnadmin", user.GetName())   // Kubernetes resource name
	assert.Equal(t, "avnadmin", userAvn.Username) // Aiven username should be avnadmin
	assert.Equal(t, pgName, user.Spec.ServiceName)

	// validates Secret with avnadmin username and predefined password
	secret, err := s.GetSecret("my-avnadmin-secret")
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USERNAME"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["CA_CERT"])

	// verify the username is avnadmin
	actualUsernameInSecret := string(secret.Data["USERNAME"])
	assert.Equal(t, "avnadmin", actualUsernameInSecret, "Username should be avnadmin")

	// verify the password matches our predefined one (not the auto-generated one)
	actualPassword := string(secret.Data["PASSWORD"])
	assert.Equal(t, "NewAvnadminPassword999!", actualPassword, "Password should match our custom avnadmin password")

	// verify custom annotations and labels
	assert.Equal(t, map[string]string{"test": "avnadmin-reset"}, secret.Annotations)
	assert.Equal(t, map[string]string{"type": "admin-password"}, secret.Labels)

	// NOTE: We don't delete the avnadmin user since it's a built-in user that cannot be deleted
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

  project: %[1]s
  serviceName: %[2]s
`, project, pgName, userName, cloudName)
}

func getServiceUserWithCustomUsernameYaml(project, pgName, resourceName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: custom-user-credentials
data:
  PASSWORD: Q3VzdG9tVXNlclBhc3M3ODkh # CustomUserPass789! base64 encoded # gitleaks:allow
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
    name: my-custom-user-secret
    annotations:
      test: custom-username
    labels:
      type: custom-username

  connInfoSecretSource:
    name: custom-user-credentials

  project: %[1]s
  serviceName: %[2]s
`, project, pgName, resourceName, cloudName)
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

  project: %[1]s
  serviceName: %[2]s
`, project, pgName, cloudName)
}
