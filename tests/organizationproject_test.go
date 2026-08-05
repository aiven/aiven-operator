//go:build misc

package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/aiven/go-client-codegen/handler/organizationprojects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestOrganizationProject(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	account, err := avnGen.AccountGet(ctx, cfg.AccountID)
	require.NoError(t, err)
	require.NotEmpty(t, account.OrganizationId)
	organizationID := account.OrganizationId

	// Picks a real billing group that belongs to the test account.
	billingGroups, err := avnGen.BillingGroupList(ctx)
	require.NoError(t, err)
	var billingGroupID string
	for _, bg := range billingGroups {
		if bg.AccountId == cfg.AccountID {
			billingGroupID = bg.BillingGroupId
			break
		}
	}
	require.NotEmpty(t, billingGroupID, "no billing group found for account %q", cfg.AccountID)

	name := randName("org-project")
	projectID := name

	yml, err := loadExampleYaml("organizationproject.yaml", map[string]string{
		"metadata.name":                  name,
		"spec.connInfoSecretTarget.name": name,
		"spec.organizationId":            organizationID,
		"spec.projectId":                 projectID,
		"spec.billingGroupId":            billingGroupID,
		"spec.parentId":                  organizationID,
	})
	require.NoError(t, err)

	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	orgProject := new(v1alpha1.OrganizationProject)
	require.NoError(t, s.GetRunning(orgProject, name))

	// THEN
	assert.Equal(t, name, orgProject.GetName())
	assert.Equal(t, organizationID, orgProject.Spec.OrganizationID)
	assert.Equal(t, projectID, orgProject.Spec.ProjectID)
	assert.Equal(t, billingGroupID, orgProject.Spec.BillingGroupID)
	assert.Equal(t, organizationID, orgProject.Spec.ParentID)

	// Validate the project exists in Aiven.
	orgProjectAvn, err := avnGen.OrganizationProjectsGet(ctx, organizationID, projectID)
	require.NoError(t, err)
	assert.Equal(t, projectID, orgProjectAvn.ProjectId)
	// The API normalizes parent_id: a project parented to the organization itself is
	// echoed back with the organization's account ID.
	assert.Contains(t, []string{organizationID, cfg.AccountID}, orgProjectAvn.ParentId)
	assert.Equal(t, "prod", orgProjectAvn.Tags["env"])
	require.Len(t, orgProjectAvn.TechEmails, 1)
	assert.Equal(t, "tech@example.com", orgProjectAvn.TechEmails[0].Email)

	// Validates the CA cert Secret is written with the resource-kind prefix.
	secret, err := s.GetSecret(name)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["ORGANIZATIONPROJECT_CA_CERT"])

	// WHEN
	// Updates tags and technical emails.
	updatedYml, err := loadExampleYaml("organizationproject.yaml", map[string]string{
		"metadata.name":                  name,
		"spec.connInfoSecretTarget.name": name,
		"spec.organizationId":            organizationID,
		"spec.projectId":                 projectID,
		"spec.billingGroupId":            billingGroupID,
		"spec.parentId":                  organizationID,
		"spec.tags.env":                  "staging",
		"spec.technicalEmails[0]":        "updated@example.com",
	})
	require.NoError(t, err)
	require.NoError(t, s.Apply(updatedYml))

	updatedOrgProject := new(v1alpha1.OrganizationProject)
	require.NoError(t, s.GetRunning(updatedOrgProject, name))

	// THEN
	// The update reconciles cleanly and the new values round-trip through the API.
	require.Eventually(t, func() bool {
		out, err := avnGen.OrganizationProjectsGet(ctx, organizationID, projectID)
		if err != nil {
			return false
		}
		if out.Tags["env"] != "staging" {
			return false
		}
		if len(out.TechEmails) != 1 {
			return false
		}
		return out.TechEmails[0].Email == "updated@example.com"
	}, time.Minute, 5*time.Second)

	t.Run("reverts out-of-band tags when spec has no tags", func(t *testing.T) {
		// GIVEN
		// A project whose spec has no tags at all: the controller treats nil tags
		// as "must be empty" in its drift check, so tags added directly in Aiven
		// are expected to be reverted.
		noTagsName := randName("org-project")
		noTagsYml, err := loadExampleYaml("organizationproject.yaml", map[string]string{
			"metadata.name":                  noTagsName,
			"spec.connInfoSecretTarget.name": noTagsName,
			"spec.organizationId":            organizationID,
			"spec.projectId":                 noTagsName,
			"spec.billingGroupId":            billingGroupID,
			"spec.parentId":                  organizationID,
			"spec.tags":                      "REMOVE",
		})
		require.NoError(t, err)
		require.NoError(t, s.Apply(noTagsYml))

		noTagsProject := new(v1alpha1.OrganizationProject)
		require.NoError(t, s.GetRunning(noTagsProject, noTagsName))
		require.Empty(t, noTagsProject.Spec.Tags)

		// WHEN
		// Adds a tag directly in Aiven, bypassing Kubernetes.
		_, err = avnGen.OrganizationProjectsUpdate(ctx, organizationID, noTagsName, &organizationprojects.OrganizationProjectsUpdateIn{
			Tags: &map[string]string{"env": "hijacked"},
		})
		require.NoError(t, err)

		out, err := avnGen.OrganizationProjectsGet(ctx, organizationID, noTagsName)
		require.NoError(t, err)
		require.Equal(t, "hijacked", out.Tags["env"], "out-of-band tag update did not land")

		// THEN
		// The controller re-observes on its poll interval (1m in the test suite),
		// detects the drift and must clear the remote tags. A nil spec.Tags is sent
		// as an empty map ("tags": {}), not null, so the API clears the field
		// instead of ignoring it.
		require.Eventually(t, func() bool {
			out, err := avnGen.OrganizationProjectsGet(ctx, organizationID, noTagsName)
			if err != nil {
				return false
			}
			return len(out.Tags) == 0
		}, 3*time.Minute, 5*time.Second, "remote tags were not cleared; spec has no tags so the drift must be reverted")
	})

	t.Run("parent_id is echoed in the account form", func(t *testing.T) {
		// orgProjectMatchesSpec resolves spec.parentId to the account form before comparing it with parent_id.
		// If the API echoed the org form instead, the comparison would never match and Update would re-run on every poll.
		out, err := avnGen.OrganizationProjectsGet(ctx, organizationID, projectID)
		require.NoError(t, err)
		org, err := avnGen.OrganizationGet(ctx, organizationID)
		require.NoError(t, err)

		assert.Equal(t, org.AccountId, out.ParentId)
		assert.NotEqual(t, organizationID, out.ParentId, "the org form must not be echoed back")
	})

	t.Run("immutable fields are rejected by the API server", func(t *testing.T) {
		// organizationId and projectId are guarded by CEL rules on the CRD, so the
		// API server must reject any change before it ever reaches the controller.
		projectIDUpdate, err := loadExampleYaml("organizationproject.yaml", map[string]string{
			"metadata.name":                  name,
			"spec.connInfoSecretTarget.name": name,
			"spec.organizationId":            organizationID,
			"spec.projectId":                 projectID + "-renamed",
			"spec.billingGroupId":            billingGroupID,
			"spec.parentId":                  organizationID,
		})
		require.NoError(t, err)
		require.ErrorContains(t, s.Apply(projectIDUpdate), "Value is immutable")

		organizationIDUpdate, err := loadExampleYaml("organizationproject.yaml", map[string]string{
			"metadata.name":                  name,
			"spec.connInfoSecretTarget.name": name,
			"spec.organizationId":            "org000000000a",
			"spec.projectId":                 projectID,
			"spec.billingGroupId":            billingGroupID,
			"spec.parentId":                  organizationID,
		})
		require.NoError(t, err)
		require.ErrorContains(t, s.Apply(organizationIDUpdate), "Value is immutable")
	})

	t.Run("duplicate technical emails are rejected by the API server", func(t *testing.T) {
		// Aiven deduplicates technicalEmails, which the controller would then see as
		// permanent drift and try to fix on every poll. A CEL rule on the CRD stops
		// the manifest at admission, so the resource is never created.
		duplicateEmails := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: OrganizationProject
metadata:
  name: %[1]s-duplicate-emails
spec:
  authSecretRef:
    name: aiven-token
    key: token
  organizationId: %[2]s
  projectId: %[1]s-duplicate-emails
  billingGroupId: %[3]s
  parentId: %[2]s
  technicalEmails:
    - tech@example.com
    - tech@example.com
`, name, organizationID, billingGroupID)

		require.ErrorContains(t, s.Apply(duplicateEmails), "Emails must be unique")
	})

	// WHEN/THEN
	// Deletes the project and asserts it is gone from Aiven.
	require.NoError(t, s.Delete(updatedOrgProject, func() error {
		_, err := avnGen.OrganizationProjectsGet(ctx, organizationID, projectID)
		return err
	}))
}
