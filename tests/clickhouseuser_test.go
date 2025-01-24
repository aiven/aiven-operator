package tests

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
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
		"doc[0].metadata.name":                  userName,
		"doc[0].spec.project":                   cfg.Project,
		"doc[0].spec.serviceName":               chName,
		"doc[0].spec.connInfoSecretTarget.name": userName,
		// Remove 'username' from the initial yaml
		"doc[0].spec.username": "REMOVE",

		"doc[1].metadata.name":  chName,
		"doc[1].spec.project":   cfg.Project,
		"doc[1].spec.cloudName": cfg.PrimaryCloudName,
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
	chAvn, err := avnGen.ServiceGet(ctx, cfg.Project, chName)
	require.NoError(t, err)
	assert.Equal(t, chAvn.ServiceName, ch.GetName())
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
		_, err = avnClient.ClickhouseUser.Get(ctx, cfg.Project, chName, user.Status.UUID)
		return err
	}))

	// User has been deleted, no access
	assert.ErrorContains(t, pinger(), "Authentication failed: password is incorrect, or there is no user with such name.")

	// GIVEN
	// New manifest with 'username' field set
	updatedUserName := randName("clickhouse-user")
	ymlUsernameSet, err := loadExampleYaml("clickhouseuser.yaml", map[string]string{
		"doc[0].metadata.name":                  "metadata-name",
		"doc[0].spec.project":                   cfg.Project,
		"doc[0].spec.connInfoSecretTarget.name": userName,
		"doc[0].spec.serviceName":               chName,
		"doc[0].spec.username":                  updatedUserName,

		"doc[1].metadata.name":  chName,
		"doc[1].spec.project":   cfg.Project,
		"doc[1].spec.cloudName": cfg.PrimaryCloudName,
	})
	require.NoError(t, err)

	// WHEN
	// Applies updated manifest
	updatedUser := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.Apply(ymlUsernameSet))
	require.NoError(t, s.GetRunning(updatedUser, "metadata-name")) // GetRunning must be called with the metadata name

	updatedUserAvn, err := avnClient.ClickhouseUser.Get(ctx, cfg.Project, chName, updatedUser.Status.UUID)
	require.NoError(t, err)

	// THEN
	// 'username' field is preferred over 'metadata.name'
	assert.NotEqual(t, updatedUserName, updatedUser.ObjectMeta.Name)
	assert.NotEqual(t, updatedUserAvn.Name, updatedUser.ObjectMeta.Name)
	assert.Equal(t, updatedUserName, updatedUser.Spec.Username)
	assert.Equal(t, updatedUserAvn.Name, updatedUser.Spec.Username)
}

func pingClickhouse[T string | []byte](ctx context.Context, host, port, username, password T) error {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Protocol: clickhouse.Native,
		Addr:     []string{fmt.Sprintf("%s:%s", host, port)},
		Auth:     clickhouse.Auth{Username: string(username), Password: string(password)},
		TLS:      &tls.Config{InsecureSkipVerify: true},
	})
	if err != nil {
		return err
	}
	return conn.Ping(ctx)
}
