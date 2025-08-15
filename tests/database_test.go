//go:build database

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

	pg, release, err := sharedResources.AcquirePostgreSQL(ctx)
	require.NoError(t, err)
	defer release()

	// Cleans test afterward
	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	pgName := pg.GetName()
	dbName := randName("database-db")
	yml, err := loadExampleYaml("database.yaml", map[string]string{
		"metadata.name":    dbName,
		"spec.project":     cfg.Project,
		"spec.serviceName": pgName,

		// Remove 'databaseName' from the initial yaml
		"spec.databaseName": "REMOVE",
	})
	require.NoError(t, err)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	db := new(v1alpha1.Database)
	require.NoError(t, s.GetRunning(db, dbName))

	// THEN
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

	ctx, cancel := testCtx()
	defer cancel()

	// GIVEN
	pg, release, err := sharedResources.AcquirePostgreSQL(ctx)
	require.NoError(t, err)
	defer release()

	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	pgName := pg.GetName()
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
			yml, err := loadExampleYaml("database.yaml", map[string]string{
				"metadata.name":     tc.metaDBName,
				"spec.project":      cfg.Project,
				"spec.serviceName":  pgName,
				"spec.databaseName": tc.specDatabaseName, // Set databaseName from test case
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
