package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getProjectYaml(name string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Project
metadata:
  name: %[1]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  tags:
    env: prod

  billingAddress: NYC
  cloud: aws-eu-west-1
`, name)
}

func TestProject(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	name := randName("project")
	yml := getProjectYaml(name)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube object
	project := new(v1alpha1.Project)
	require.NoError(t, s.GetRunning(project, name))

	// THEN
	// Validates Project
	projectAvn, err := avnClient.Projects.Get(name)
	require.NoError(t, err)
	assert.Equal(t, name, project.GetName())
	assert.Equal(t, projectAvn.Name, project.GetName())
	assert.Equal(t, "NYC", project.Spec.BillingAddress)
	assert.Equal(t, "NYC", projectAvn.BillingAddress)
	assert.Equal(t, "aws-eu-west-1", project.Spec.Cloud)

	// Validates Secret
	secret, err := s.GetSecret(project.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["CA_CERT"])
}
