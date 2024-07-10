package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestClickhouseUser(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	chName := randName("clickhouse")
	userName := randName("clickhouse-user")
	yml, err := loadExampleYaml("clickhouseuser.yaml", map[string]string{
		"google-europe-west1":    cfg.PrimaryCloudName,
		"my-aiven-project":       cfg.Project,
		"clickhouse-user-secret": userName,
		"my-clickhouse-user":     userName,
		"my-clickhouse":          chName,
		// Remove 'username' from the initial yaml
		"username: example-username": "",
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ch := new(v1alpha1.Clickhouse)
	require.NoError(t, s.GetRunning(ch, chName))

	// THEN
	chAvn, err := avnClient.Services.Get(ctx, cfg.Project, chName)
	require.NoError(t, err)
	assert.Equal(t, chAvn.Name, ch.GetName())
	assert.Equal(t, serviceRunningState, ch.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, chAvn.State)
	assert.Equal(t, chAvn.Plan, ch.Spec.Plan)
	assert.Equal(t, chAvn.CloudName, ch.Spec.CloudName)

	user := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.GetRunning(user, userName))

	userAvn, err := avnClient.ClickhouseUser.Get(ctx, cfg.Project, chName, user.Status.UUID)
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

	// GIVEN
	// New manifest with 'username' field set
	updatedUserName := randName("clickhouse-user")
	ymlUsernameSet, err := loadExampleYaml("clickhouseuser.yaml", map[string]string{
		"google-europe-west1":        cfg.PrimaryCloudName,
		"my-aiven-project":           cfg.Project,
		"clickhouse-user-secret":     updatedUserName,
		"name: my-clickhouse-user":   "name: metadata-name",
		"my-clickhouse":              chName,
		"username: example-username": fmt.Sprintf("username: %s", updatedUserName),
	})
	require.NoError(t, err)

	// WHEN
	// Applies updated manifest
	updatedUser := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.Apply(ymlUsernameSet))
	require.NoError(t, s.GetRunning(updatedUser, "metadata-name")) // GetRunning must be called with the metadata name

	userAvn, err = avnClient.ClickhouseUser.Get(ctx, cfg.Project, chName, updatedUser.Status.UUID)
	require.NoError(t, err)

	// THEN
	// 'username' field is preferred over 'metadata.name'
	assert.NotEqual(t, updatedUserName, updatedUser.ObjectMeta.Name)
	assert.NotEqual(t, userAvn.Name, updatedUser.ObjectMeta.Name)
	assert.Equal(t, updatedUserName, updatedUser.Spec.Username)
	assert.Equal(t, userAvn.Name, updatedUser.Spec.Username)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, user is destroyed in Aiven. No service — no user. No user — no user.
	// And we make sure that controller can delete user itself
	assert.NoError(t, s.Delete(user, func() error {
		_, err = avnClient.ClickhouseUser.Get(ctx, cfg.Project, chName, user.Status.UUID)
		return err
	}))
}
