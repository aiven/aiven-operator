//go:build misc

package tests

import (
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestServiceIntegrationEndpointExternalPostgres(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	t.Run("creates and reconciles a new endpoint", func(t *testing.T) {
		t.Parallel()
		defer recoverPanic(t)

		// GIVEN
		ctx, cancel := testCtx()
		defer cancel()

		endpointPgName := randName("postgresql")

		yml, err := loadExampleYaml("serviceintegrationendpoint.external_postgresql.yaml", map[string]string{
			"metadata.name": endpointPgName,
			"spec.project":  cfg.Project,
		})
		require.NoError(t, err)
		s := NewSession(ctx, k8sClient)

		// Cleans test afterward
		defer s.Destroy(t)

		// WHEN
		// Applies given manifest
		require.NoError(t, s.Apply(yml))

		// THEN

		// Validates ServiceIntegrationEndpoint externalPostgresql
		endpointPg := new(v1alpha1.ServiceIntegrationEndpoint)
		require.NoError(t, s.GetRunning(endpointPg, endpointPgName))
		endpointPgAvn, err := avnGen.ServiceIntegrationEndpointGet(ctx, cfg.Project, endpointPg.Status.ID, service.ServiceIntegrationEndpointGetIncludeSecrets(true))
		require.NoError(t, err)
		assert.Equal(t, "external_postgresql", string(endpointPgAvn.EndpointType))
		assert.Equal(t, "username", endpointPg.Spec.ExternalPostgresql.Username)
		assert.Equal(t, "password", *endpointPg.Spec.ExternalPostgresql.Password)
		assert.Equal(t, "example.example", endpointPg.Spec.ExternalPostgresql.Host)
		assert.Equal(t, 5432, endpointPg.Spec.ExternalPostgresql.Port)
		assert.EqualValues(t, endpointPgAvn.EndpointType, endpointPg.Spec.EndpointType)
		assert.EqualValues(t, endpointPgAvn.UserConfig["username"], endpointPg.Spec.ExternalPostgresql.Username)
		assert.EqualValues(t, endpointPgAvn.UserConfig["password"], *endpointPg.Spec.ExternalPostgresql.Password)
		assert.EqualValues(t, endpointPgAvn.UserConfig["host"], endpointPg.Spec.ExternalPostgresql.Host)
		assert.EqualValues(t, endpointPgAvn.UserConfig["port"], endpointPg.Spec.ExternalPostgresql.Port)
	})

	// Verifies that the controller adopts a pre-existing Aiven endpoint that matches the
	// manifest's name+type instead of trying to create a duplicate.
	t.Run("adopts a pre-existing endpoint with the same name and type", func(t *testing.T) {
		t.Parallel()
		defer recoverPanic(t)

		// GIVEN
		ctx, cancel := testCtx()
		defer cancel()

		endpointName := randName("adoption")

		// Pre-create the endpoint directly via the Aiven API, simulating an endpoint that exists.
		preCreated, err := avnGen.ServiceIntegrationEndpointCreate(ctx, cfg.Project, &service.ServiceIntegrationEndpointCreateIn{
			EndpointName: endpointName,
			EndpointType: service.EndpointTypeExternalPostgresql,
			UserConfig: map[string]any{
				"username": "username",
				"password": "password",
				"host":     "example.example",
				"port":     5432,
				"ssl_mode": "require",
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, preCreated.EndpointId)

		yml, err := loadExampleYaml("serviceintegrationendpoint.external_postgresql.yaml", map[string]string{
			"metadata.name":     endpointName,
			"spec.project":      cfg.Project,
			"spec.endpointName": endpointName,
		})
		require.NoError(t, err)
		s := NewSession(ctx, k8sClient)

		defer s.Destroy(t)

		// WHEN
		require.NoError(t, s.Apply(yml))

		// THEN
		// The controller must adopt the pre-existing endpoint.
		endpoint := new(v1alpha1.ServiceIntegrationEndpoint)
		require.NoError(t, s.GetRunning(endpoint, endpointName))
		assert.Equal(t, preCreated.EndpointId, endpoint.Status.ID, "controller should adopt the existing endpoint, not create a new one")

		// And exactly one endpoint with this name+type exists on Aiven.
		list, err := avnGen.ServiceIntegrationEndpointList(ctx, cfg.Project)
		require.NoError(t, err)
		var matching int
		for i := range list {
			if list[i].EndpointName == endpointName && list[i].EndpointType == service.EndpointTypeExternalPostgresql {
				matching++
			}
		}
		assert.Equal(t, 1, matching, "expected exactly one endpoint with this name+type after adoption")
	})
}

func TestServiceIntegrationEndpoint(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	endpointRegistryName := randName("schema-registry")

	yml, err := loadExampleYaml("serviceintegrationendpoint.external_schema_registry.yaml", map[string]string{
		"metadata.name": endpointRegistryName,
		"spec.project":  cfg.Project,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// THEN

	// Validates ServiceIntegrationEndpoint externalSchemaRegistry
	endpointRegistry := new(v1alpha1.ServiceIntegrationEndpoint)
	require.NoError(t, s.GetRunning(endpointRegistry, endpointRegistryName))
	endpointRegistryAvn, err := avnGen.ServiceIntegrationEndpointGet(ctx, cfg.Project, endpointRegistry.Status.ID, service.ServiceIntegrationEndpointGetIncludeSecrets(true))
	require.NoError(t, err)
	assert.Equal(t, "external_schema_registry", string(endpointRegistryAvn.EndpointType))
	assert.Equal(t, "https://schema-registry.example.com:8081", endpointRegistry.Spec.ExternalSchemaRegistry.Url)
	assert.Equal(t, "basic", endpointRegistry.Spec.ExternalSchemaRegistry.Authentication)
	assert.Equal(t, "username", *endpointRegistry.Spec.ExternalSchemaRegistry.BasicAuthUsername)
	assert.Equal(t, "password", *endpointRegistry.Spec.ExternalSchemaRegistry.BasicAuthPassword)
	assert.EqualValues(t, endpointRegistryAvn.EndpointType, endpointRegistry.Spec.EndpointType)
	assert.EqualValues(t, endpointRegistryAvn.UserConfig["url"], endpointRegistry.Spec.ExternalSchemaRegistry.Url)
	assert.EqualValues(t, endpointRegistryAvn.UserConfig["authentication"], endpointRegistry.Spec.ExternalSchemaRegistry.Authentication)
	assert.EqualValues(t, endpointRegistryAvn.UserConfig["basic_auth_username"], *endpointRegistry.Spec.ExternalSchemaRegistry.BasicAuthUsername)
	assert.EqualValues(t, endpointRegistryAvn.UserConfig["basic_auth_password"], *endpointRegistry.Spec.ExternalSchemaRegistry.BasicAuthPassword)
}

func TestServiceIntegrationEndpointAutoscaler(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	endpointName := randName("autoscaler")

	yml, err := loadExampleYaml("serviceintegrationendpoint.autoscaler.yaml", map[string]string{
		"metadata.name": endpointName,
		"spec.project":  cfg.Project,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// THEN

	// Validates autoscaler ServiceIntegrationEndpoint
	endpointAutoscaler := new(v1alpha1.ServiceIntegrationEndpoint)
	require.NoError(t, s.GetRunning(endpointAutoscaler, endpointName))

	endpointAvn, err := avnGen.ServiceIntegrationEndpointGet(ctx, cfg.Project, endpointAutoscaler.Status.ID, service.ServiceIntegrationEndpointGetIncludeSecrets(true))
	require.NoError(t, err)

	assert.EqualValues(t, "autoscaler", endpointAvn.EndpointType)
	assert.EqualValues(t, endpointAvn.EndpointType, endpointAutoscaler.Spec.EndpointType)
	// TODO: remove type assertions once generated client has full user config typing
	assert.Equal(t, "autoscale_disk", endpointAutoscaler.Spec.Autoscaler.Autoscaling[0].Type)
	assert.EqualValues(t, "autoscale_disk", endpointAvn.UserConfig["autoscaling"].([]interface{})[0].(map[string]interface{})["type"])
	assert.Equal(t, 100, endpointAutoscaler.Spec.Autoscaler.Autoscaling[0].CapGb)
	assert.EqualValues(t, 100, endpointAvn.UserConfig["autoscaling"].([]interface{})[0].(map[string]interface{})["cap_gb"])
}
