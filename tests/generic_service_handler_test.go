package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getCreateServiceYaml(project, pgName string) string {
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

  tags:
    env: prod
    instance: master
`, project, pgName)
}

func getUpdateServiceYaml(project, pgName string) string {
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
`, project, pgName)
}

// TestCreateUpdateService tests create and update flow
func TestCreateUpdateService(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx := context.Background()
	pgName := randName("generic-handler")
	ymlCreate := getCreateServiceYaml(testProject, pgName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(ymlCreate))

	// Waits kube objects
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	// THEN
	// Validates tags
	tagsCreatedAvn, err := avnClient.ServiceTags.Get(ctx, testProject, pgName)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"env": "prod", "instance": "master"}, pg.Spec.Tags)
	assert.Equal(t, tagsCreatedAvn.Tags, pg.Spec.Tags)

	// Updates tags
	ymlUpdate := getUpdateServiceYaml(testProject, pgName)
	require.NoError(t, err)
	require.NoError(t, s.Apply(ymlUpdate))

	pgUpdated := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pgUpdated, pgName))
	tagsUpdatedAvn, err := avnClient.ServiceTags.Get(ctx, testProject, pgName)
	require.NoError(t, err)
	assert.Empty(t, tagsUpdatedAvn.Tags) // cleared tags
}
