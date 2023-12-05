package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

func getErrorConditionYaml(project, pgName string) string {
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
  plan: startup-1234
`, project, pgName)
}

func TestErrorCondition(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx := context.Background()
	pgName := randName("generic-handler")
	yml := getErrorConditionYaml(testProject, pgName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// THEN
	pg := new(v1alpha1.PostgreSQL)
	for *pg.Conditions() == nil {
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: pgName}, pg)
		if apierrors.IsNotFound(err) {
			// Ignore not found, because it takes time to commit a resource to the storage
			err = nil
		}
		require.NoError(t, err)
		time.Sleep(time.Second)
	}

	found := false
	for _, c := range *pg.Conditions() {
		if c.Reason == "CreateOrUpdate" {
			found = true
			assert.Contains(t, c.Message, "Plan 'startup-1234' for service type ServiceType.pg is not available in cloud 'google-europe-west1'")
		}
	}
	assert.True(t, found)
}
