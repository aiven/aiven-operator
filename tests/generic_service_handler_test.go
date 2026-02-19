//go:build genericservicehandler

package tests

import (
	"fmt"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
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

  maintenanceWindowDow: friday
  maintenanceWindowTime: 09:00:00

  tags:
    env: prod
    instance: master
`, project, pgName)
}

func getUpdateServiceYaml(project, pgName string, powered bool) string {
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
  powered: %[3]t

  maintenanceWindowDow: sunday
  maintenanceWindowTime: 11:00:00
`, project, pgName, powered)
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
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(ymlCreate))

	// Waits kube objects
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))
	assert.Equal(t, "true", controllers.GetIsRunningAnnotation(pg))

	// THEN
	// Validates tags
	tagsCreatedAvnTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"env": "prod", "instance": "master"}, pg.Spec.Tags)
	assert.Equal(t, tagsCreatedAvnTags, pg.Spec.Tags)

	// Validates the maintenance window
	avnPg, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypeRunning, avnPg.State)
	assert.Equal(t, service.MaintenanceDowTypeFriday, avnPg.Maintenance.Dow)
	assert.Equal(t, "09:00:00", avnPg.Maintenance.Time)

	// Removes the tags and updates the maintenance window
	ymlUpdate := getUpdateServiceYaml(cfg.Project, pgName, true)
	require.NoError(t, err)
	require.NoError(t, s.Apply(ymlUpdate))

	pgUpdated := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pgUpdated, pgName))
	tagsUpdatedAvnTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Empty(t, tagsUpdatedAvnTags) // cleared tags

	// Validates the maintenance window is updated
	avnPgUpdated, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypeRunning, avnPgUpdated.State)
	assert.Equal(t, service.MaintenanceDowTypeSunday, avnPgUpdated.Maintenance.Dow)
	assert.Equal(t, "11:00:00", avnPgUpdated.Maintenance.Time)

	// Validates service is powered off
	ymlPowerOff := getUpdateServiceYaml(cfg.Project, pgName, false)
	require.NoError(t, s.Apply(ymlPowerOff))

	pgPoweredOff := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pgPoweredOff, pgName))

	avnPgPoweredOff, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypePoweroff, avnPgPoweredOff.State)
	assert.Equal(t, "false", controllers.GetIsRunningAnnotation(pgPoweredOff))

	// Validates the service is powered on
	ymlPowerOn := getUpdateServiceYaml(cfg.Project, pgName, true)
	require.NoError(t, s.Apply(ymlPowerOn))

	pgPoweredOn := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pgPoweredOn, pgName))

	avnPgPoweredOn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypeRunning, avnPgPoweredOn.State)
	assert.Equal(t, "true", controllers.GetIsRunningAnnotation(pgPoweredOn))
}

// TestTerminationProtectionDeletion verifies that the controller can delete a service
// when termination protection is disabled in the K8s spec but still enabled on the Aiven side.
func TestTerminationProtectionDeletion(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	pgName := randName("tp-delete")
	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	// Create a PostgreSQL service with termination protection disabled.
	yml := fmt.Sprintf(`
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
  terminationProtection: false
`, cfg.Project, pgName)
	require.NoError(t, s.Apply(yml))

	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	avnPg, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	require.False(t, avnPg.TerminationProtection)

	// Simulate the race condition: enable termination protection directly on Aiven, bypassing the controller.
	// This creates the exact inconsistent state: K8s: terminationProtection=false / Aiven: terminationProtection=true
	tp := true
	_, err = avnGen.ServiceUpdate(ctx, cfg.Project, pgName, &service.ServiceUpdateIn{
		TerminationProtection: &tp,
	})
	require.NoError(t, err)

	// WHEN: delete the K8s resource.
	// If the deadlock occurs, this times out.
	require.NoError(t, s.Delete(pg, func() error {
		_, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
		return err
	}))

	// THEN: verify the service is gone from Aiven, not just from K8s.
	assert.Eventually(t, func() bool {
		_, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
		return avngen.IsNotFound(err)
	}, 2*time.Minute, 5*time.Second, "service %s should be deleted from Aiven", pgName)
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
	s := NewSession(ctx, k8sClient)

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
			// check that the error mentions the invalid plan name and service type API may return different messages
			assert.Contains(t, c.Message, "startup-1234")
		}
	}
	assert.True(t, found)
}
