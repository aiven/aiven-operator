package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

func TestDatabase(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()
	s := NewSession(ctx, k8sClient, cfg.Project)

	pgName := randName("database-pg")
	dbName := randName("database-db")
	yml, err := loadExampleYaml("database.yaml", map[string]string{
		"doc[0].metadata.name":  pgName,
		"doc[0].spec.project":   cfg.Project,
		"doc[0].spec.cloudName": cfg.PrimaryCloudName,

		"doc[1].metadata.name":    dbName,
		"doc[1].spec.project":     cfg.Project,
		"doc[1].spec.serviceName": pgName,

		// Remove 'databaseName' from the initial yaml
		"doc[1].spec.databaseName": "REMOVE",
	})
	require.NoError(t, err)
	// Cleans test afterward
	defer s.Destroy(t)

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
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Equal(t, serviceRunningState, pg.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)
	assert.Equal(t, pgAvn.Plan, pg.Spec.Plan)
	assert.Equal(t, pgAvn.CloudName, pg.Spec.CloudName)

	// Validates Database
	dbAvn, err := controllers.GetDatabaseByName(ctx, avnGen, cfg.Project, pgName, dbName)
	require.NoError(t, err)
	assert.Equal(t, dbName, db.GetName())
	assert.Equal(t, dbAvn.DatabaseName, db.GetName())
	assert.Equal(t, "en_US.UTF-8", db.Spec.LcCtype) // the default value
	assert.Equal(t, fromPtr(dbAvn.LcCtype), db.Spec.LcCtype)
	assert.Equal(t, "en_US.UTF-8", db.Spec.LcCollate) // the default value
	assert.Equal(t, fromPtr(dbAvn.LcCollate), db.Spec.LcCollate)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, db is destroyed in Aiven. No service — no db. No db — no db.
	// And we make sure that controller can delete db itself
	assert.NoError(t, s.Delete(db, func() error {
		_, err = controllers.GetDatabaseByName(ctx, avnGen, cfg.Project, pgName, dbName)
		return err
	}))
}

func TestDatabase_databaseName(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	pgName := randName("databasename-pg")

	// Create a PostgreSQL to reuse with database test cases
	yml, err := loadExampleYaml("database.yaml", map[string]string{
		"doc[0].metadata.name":  pgName,
		"doc[0].spec.project":   cfg.Project,
		"doc[0].spec.cloudName": cfg.PrimaryCloudName,

		"doc[1]": "REMOVE", // remove database from yaml
	})
	require.NoError(t, err)

	// WHEN
	// Applies given manifest
	err = s.Apply(yml)
	require.NoError(t, err, "Failed to apply YAML")

	// Waits kube objects
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	testCases := []struct {
		name                      string
		metaDBName                string
		specDatabaseName          string
		expectedAivenDatabaseName string
		expectError               bool
		expectUpdateError         bool
		expectErrorMsgContains    string
	}{
		{
			name:                   "invalid - metadata.name contains underscores",
			metaDBName:             "invalid_db_name",
			specDatabaseName:       "REMOVE",
			expectError:            true,
			expectErrorMsgContains: "Invalid value: \"invalid_db_name\": a lowercase RFC 1123 subdomain must",
		},
		{
			name:                   "invalid - databaseName exceeds maxLength",
			metaDBName:             "metadata-db-name",
			specDatabaseName:       "db-name-very-long-name-exceeding-maxlength", // exceed maxLength of 40
			expectError:            true,
			expectErrorMsgContains: "Too long: may not be longer than 40",
		},
		{
			name:                   "invalid - spec.databaseName contains invalid characters",
			metaDBName:             "valid-db-name",
			specDatabaseName:       "invalid-db-name!",
			expectError:            true,
			expectErrorMsgContains: "Invalid value: \"invalid-db-name!\": spec.databaseName in body should match",
		},
		{
			name:                      "valid - default to metadata.name",
			metaDBName:                "metadata-db-name",
			specDatabaseName:          "REMOVE", // remove from yaml, should default to metadata.name
			expectedAivenDatabaseName: "metadata-db-name",
		},
		{
			name:                      "valid - uses spec.databaseName if specified",
			metaDBName:                "metadata-db-name", // both metadata.name and spec.databaseName are specified, should use spec.databaseName
			specDatabaseName:          "spec_valid_db_name",
			expectedAivenDatabaseName: "spec_valid_db_name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := testCtx()
			defer cancel()

			yml, err := loadExampleYaml("database.yaml", map[string]string{
				"doc[0].metadata.name":  pgName,
				"doc[0].spec.project":   cfg.Project,
				"doc[0].spec.cloudName": cfg.PrimaryCloudName,

				"doc[1].metadata.name":     tc.metaDBName,
				"doc[1].spec.project":      cfg.Project,
				"doc[1].spec.serviceName":  pgName,
				"doc[1].spec.databaseName": tc.specDatabaseName, // Set databaseName from test case
			})
			require.NoError(t, err)

			// WHEN
			// Applies given manifest
			err = s.Apply(yml)

			// IF expectError
			if tc.expectError {
				// THEN expect error during apply
				assert.ErrorContains(t, err, tc.expectErrorMsgContains)
				return // Skip further assertions for error cases
			}

			require.NoError(t, err, "Failed to apply YAML")

			db := new(v1alpha1.Database)
			require.NoError(t, s.GetRunning(db, tc.metaDBName))

			// THEN
			// Validates Database
			dbAvn, err := controllers.GetDatabaseByName(ctx, avnGen, cfg.Project, pgName, tc.expectedAivenDatabaseName)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedAivenDatabaseName, dbAvn.DatabaseName)

			// Deletion validation
			assert.NoError(t, s.Delete(db, func() error {
				_, err = controllers.GetDatabaseByName(ctx, avnGen, cfg.Project, pgName, tc.expectedAivenDatabaseName)
				return err
			}))
		})
	}
}
