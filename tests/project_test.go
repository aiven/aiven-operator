//go:build misc

package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestProject(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	account, err := avnGen.AccountGet(ctx, cfg.AccountID)
	require.NoError(t, err)
	require.NotEmpty(t, account.OrganizationId)

	// Pins an existing billing group so a failed run can't strand an auto-created one.
	billingGroups, err := avnGen.OrganizationBillingGroupList(ctx, account.OrganizationId)
	require.NoError(t, err)
	require.NotEmpty(t, billingGroups, "no billing group found for organization %q", account.OrganizationId)
	billingGroupID := billingGroups[0].BillingGroupId

	name := randName("project")
	yml, err := loadExampleYaml("project.yaml", map[string]string{
		"metadata.name":       name,
		"spec.accountId":      cfg.AccountID,
		"spec.billingGroupId": billingGroupID,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube object
	project := new(v1alpha1.Project)
	require.NoError(t, s.GetRunning(project, name))

	// THEN
	// Validates Project
	projectAvn, err := avnGen.ProjectGet(ctx, name)
	require.NoError(t, err)
	assert.Equal(t, name, project.GetName())
	assert.Equal(t, projectAvn.ProjectName, project.GetName())
	assert.Equal(t, "NYC", project.Spec.BillingAddress)
	assert.Equal(t, "NYC", projectAvn.BillingAddress)
	assert.Equal(t, "aws-eu-west-1", project.Spec.Cloud)

	// Validates Secret
	secret, err := s.GetSecret(project.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["CA_CERT"])
	assert.NotEmpty(t, secret.Data["PROJECT_CA_CERT"])
}
