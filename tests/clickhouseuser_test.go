package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getClickhouseUserYaml(project, chName, userName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[4]s
  plan: startup-16

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
    name: my-ch-user-secret
    annotations:
      foo: bar
    labels:
      baz: egg

  project: %[1]s
  serviceName: %[2]s

`, project, chName, userName, cloudName)
}

func TestClickhouseUser(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	chName := randName("clickhouse-user")
	userName := randName("clickhouse-user")
	yml := getClickhouseUserYaml(testProject, chName, userName, testCloudName)
	s, err := NewSession(k8sClient, avnClient, testProject, yml)
	require.NoError(t, err)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply())

	// Waits kube objects
	ch := new(v1alpha1.Clickhouse)
	require.NoError(t, s.GetRunning(ch, chName))

	// THEN
	chAvn, err := avnClient.Services.Get(testProject, chName)
	require.NoError(t, err)
	assert.Equal(t, chAvn.Name, ch.GetName())
	assert.Equal(t, "RUNNING", ch.Status.State)
	assert.Equal(t, chAvn.State, ch.Status.State)
	assert.Equal(t, chAvn.Plan, ch.Spec.Plan)
	assert.Equal(t, chAvn.CloudName, ch.Spec.CloudName)

	user := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.GetRunning(user, userName))

	userAvn, err := avnClient.ClickhouseUser.Get(testProject, chName, user.Status.UUID)
	require.NoError(t, err)
	assert.Equal(t, userName, user.GetName())
	assert.Equal(t, userAvn.Name, user.GetName())

	secret, err := s.GetSecret(user.Spec.ConnInfoSecretTarget.Name)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["USERNAME"])
	assert.Equal(t, map[string]string{"foo": "bar"}, secret.Annotations)
	assert.Equal(t, map[string]string{"baz": "egg"}, secret.Labels)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, user is destroyed in Aiven. No service — no user. No user — no user.
	// And we make sure that controller can delete user itself
	assert.NoError(t, s.Delete(user, func() error {
		_, err = avnClient.ClickhouseUser.Get(testProject, chName, user.Status.UUID)
		return err
	}))
}
