package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getTechnicalEmailsYaml(project, name, cloudName string, includeTechnicalEmails bool) string {
	baseYaml := `
apiVersion: aiven.io/v1alpha1
kind: MySQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: startup-1
`
	if includeTechnicalEmails {
		baseYaml += `
  technicalEmails:
    - email: "test@example.com"
`
	}

	return fmt.Sprintf(baseYaml, project, name, cloudName)
}

func TestServiceTechnicalEmails(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx := context.Background()
	name := randName("mysql")
	yml := getTechnicalEmailsYaml(testProject, name, testPrimaryCloudName, true)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ms := new(v1alpha1.MySQL)
	require.NoError(t, s.GetRunning(ms, name))

	// THEN
	// Technical emails are set
	msAvn, err := avnClient.Services.Get(ctx, testProject, name)
	require.NoError(t, err)
	assert.Len(t, ms.Spec.TechnicalEmails, 1)
	assert.Equal(t, "test@example.com", msAvn.TechnicalEmails[0].Email)

	// WHEN
	// Technical emails are removed from manifest
	updatedYml := getTechnicalEmailsYaml(testProject, name, testPrimaryCloudName, false)

	// Applies updated manifest
	require.NoError(t, s.Apply(updatedYml))

	// Waits kube objects
	require.NoError(t, s.GetRunning(ms, name))

	// THEN
	// Technical emails are removed from service
	msAvnUpdated, err := avnClient.Services.Get(ctx, testProject, name)
	require.NoError(t, err)
	assert.Empty(t, msAvnUpdated.TechnicalEmails)
}
