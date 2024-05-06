package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestServiceIntegrationEndpointExternalPostgres(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	endpointPgName := randName("postgresql")

	yml, err := loadExampleYaml("serviceintegrationendpoint.external_postgresql.yaml", map[string]string{
		"aiven-project-name":              cfg.Project,
		"my-service-integration-endpoint": endpointPgName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// THEN

	// Validates ServiceIntegrationEndpoint externalPostgresql
	endpointPg := new(v1alpha1.ServiceIntegrationEndpoint)
	require.NoError(t, s.GetRunning(endpointPg, endpointPgName))
	endpointPgAvn, err := avnGen.ServiceIntegrationEndpointGet(ctx, cfg.Project, endpointPg.Status.ID)
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
}

func TestServiceIntegrationEndpoint(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	endpointRegistryName := randName("schema-registry")

	yml, err := loadExampleYaml("serviceintegrationendpoint.external_schema_registry.yaml", map[string]string{
		"aiven-project-name":              cfg.Project,
		"my-service-integration-endpoint": endpointRegistryName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// THEN

	// Validates ServiceIntegrationEndpoint externalSchemaRegistry
	endpointRegistry := new(v1alpha1.ServiceIntegrationEndpoint)
	require.NoError(t, s.GetRunning(endpointRegistry, endpointRegistryName))
	endpointRegistryAvn, err := avnGen.ServiceIntegrationEndpointGet(ctx, cfg.Project, endpointRegistry.Status.ID)
	require.NoError(t, err)
	assert.Equal(t, "external_schema_registry", string(endpointRegistryAvn.EndpointType))
	assert.Equal(t, "https://schema-registry.example.com:8081", endpointRegistry.Spec.ExternalSchemaRegistry.Url)
	assert.Equal(t, "basic", endpointRegistry.Spec.ExternalSchemaRegistry.Authentication)
	assert.EqualValues(t, "username", *endpointRegistry.Spec.ExternalSchemaRegistry.BasicAuthUsername)
	assert.EqualValues(t, "password", *endpointRegistry.Spec.ExternalSchemaRegistry.BasicAuthPassword)
	assert.EqualValues(t, endpointRegistryAvn.EndpointType, endpointRegistry.Spec.EndpointType)
	assert.EqualValues(t, endpointRegistryAvn.UserConfig["url"], endpointRegistry.Spec.ExternalSchemaRegistry.Url)
	assert.EqualValues(t, endpointRegistryAvn.UserConfig["authentication"], endpointRegistry.Spec.ExternalSchemaRegistry.Authentication)
	assert.EqualValues(t, endpointRegistryAvn.UserConfig["basic_auth_username"], *endpointRegistry.Spec.ExternalSchemaRegistry.BasicAuthUsername)
	assert.EqualValues(t, endpointRegistryAvn.UserConfig["basic_auth_password"], *endpointRegistry.Spec.ExternalSchemaRegistry.BasicAuthPassword)
}
