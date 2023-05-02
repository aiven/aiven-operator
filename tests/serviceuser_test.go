package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getServiceUserYaml(project, pgName, userName string) string {
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
  cloudName: google-europe-west1
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
`, project, pgName, userName)
}

func TestServiceUser(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	pgName := randName("connection-pool")
	userName := randName("connection-pool")
	yml := getServiceUserYaml(testProject, pgName, userName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

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
	pgAvn, err := avnClient.Services.Get(testProject, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.Name, pg.GetName())
	assert.Equal(t, "RUNNING", pg.Status.State)
	assert.Equal(t, pgAvn.State, pg.Status.State)
	assert.Equal(t, pgAvn.Plan, pg.Spec.Plan)
	assert.Equal(t, pgAvn.CloudName, pg.Spec.CloudName)

	// Validates ServiceUser
	userAvn, err := avnClient.ServiceUsers.Get(testProject, pgName, userName)
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
	assert.Equal(t, map[string]string{"foo": "bar"}, secret.Annotations)
	assert.Equal(t, map[string]string{"baz": "egg"}, secret.Labels)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, pool is destroyed in Aiven. No service — no pool. No pool — no pool.
	// And we make sure that controller can delete db itself
	assert.NoError(t, s.Delete(user, func() error {
		_, err = avnClient.ServiceUsers.Get(testProject, pgName, userName)
		return err
	}))
}
