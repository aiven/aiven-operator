package tests

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getServiceUserYaml(project, kafkaName, userName, cloudName string) string {
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

func TestServiceUser(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx := context.Background()
	kafkaName := randName("service-user")
	userName := randName("service-user")
	yml := getServiceUserYaml(testProject, kafkaName, userName, testPrimaryCloudName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterward
	defer s.Destroy()

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
	kafkaAvn, err := avnClient.Services.Get(ctx, testProject, kafkaName)
	require.NoError(t, err)
	assert.Equal(t, kafkaAvn.Name, kafka.GetName())
	assert.Equal(t, "RUNNING", kafka.Status.State)
	assert.Equal(t, kafkaAvn.State, kafka.Status.State)
	assert.Equal(t, kafkaAvn.Plan, kafka.Spec.Plan)
	assert.Equal(t, kafkaAvn.CloudName, kafka.Spec.CloudName)

	// Validates ServiceUser
	userAvn, err := avnClient.ServiceUsers.Get(ctx, testProject, kafkaName, userName)
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
	assert.NotEmpty(t, kafkaAvn.URIParams["port"])
	assert.NotEqual(t, kafkaAvn.URIParams["port"], strPort)

	intPort, err := strconv.ParseInt(strPort, 10, 32)
	assert.NoError(t, err)
	assert.True(t, intPort > 0)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, pool is destroyed in Aiven. No service — no pool. No pool — no pool.
	// And we make sure that the controller can delete db itself
	assert.NoError(t, s.Delete(user, func() error {
		_, err = avnClient.ServiceUsers.Get(ctx, testProject, kafkaName, userName)
		return err
	}))
}
