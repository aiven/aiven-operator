package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getDatabaseYaml(project, pgName, dbName, cloudName string) string {
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
kind: Database
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s

  lcCtype: en_US.UTF-8
  lcCollate: en_US.UTF-8

`, project, pgName, dbName, cloudName)
}

func TestDatabase(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	pgName := randName("database")
	dbName := randName("database")
	yml := getDatabaseYaml(testProject, pgName, dbName, testPrimaryCloudName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	db := new(v1alpha1.Database)
	require.NoError(t, s.GetRunning(db, dbName))

	// THEN
	// Validates PostgreSQL
	pgAvn, err := avnClient.Services.Get(testProject, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.Name, pg.GetName())
	assert.Equal(t, "RUNNING", pg.Status.State)
	assert.Equal(t, pgAvn.State, pg.Status.State)
	assert.Equal(t, pgAvn.Plan, pg.Spec.Plan)
	assert.Equal(t, pgAvn.CloudName, pg.Spec.CloudName)

	// Validates Database
	dbAvn, err := avnClient.Databases.Get(testProject, pgName, dbName)
	require.NoError(t, err)
	assert.Equal(t, dbName, db.GetName())
	assert.Equal(t, dbAvn.DatabaseName, db.GetName())
	assert.Equal(t, "en_US.UTF-8", db.Spec.LcCtype)
	assert.Equal(t, dbAvn.LcType, db.Spec.LcCtype)
	assert.Equal(t, "en_US.UTF-8", db.Spec.LcCollate)
	assert.Equal(t, dbAvn.LcCollate, db.Spec.LcCollate)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, db is destroyed in Aiven. No service — no db. No db — no db.
	// And we make sure that controller can delete db itself
	assert.NoError(t, s.Delete(db, func() error {
		_, err = avnClient.Databases.Get(testProject, pgName, dbName)
		return err
	}))
}
