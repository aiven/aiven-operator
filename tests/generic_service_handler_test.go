package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const serviceRunningState = service.ServiceStateTypeRunning

// serviceRunningStatesAiven these Aiven service states match to RUNNING state in kube
var serviceRunningStatesAiven = []service.ServiceStateType{service.ServiceStateTypeRunning, service.ServiceStateTypeRebalancing}

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
	ctx, cancel := testCtx()
	defer cancel()

	pgName := randName("generic-handler")
	ymlCreate := getCreateServiceYaml(cfg.Project, pgName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(ymlCreate))

	// Waits kube objects
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	// THEN
	// Validates tags
	tagsCreatedAvnTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, pgName)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"env": "prod", "instance": "master"}, pg.Spec.Tags)
	assert.Equal(t, tagsCreatedAvnTags, pg.Spec.Tags)

	// Updates tags
	ymlUpdate := getUpdateServiceYaml(cfg.Project, pgName)
	require.NoError(t, err)
	require.NoError(t, s.Apply(ymlUpdate))

	pgUpdated := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pgUpdated, pgName))
	tagsUpdatedAvnTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Empty(t, tagsUpdatedAvnTags) // cleared tags
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
	ctx, cancel := testCtx()
	defer cancel()

	pgName := randName("generic-handler")
	yml := getErrorConditionYaml(cfg.Project, pgName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

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
