package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/pointer"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getTechnicalEmailsYaml(project, name, cloudName string, includeTechnicalEmails bool) string {
	baseYaml := `
apiVersion: aiven.io/v1alpha1
kind: Grafana
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
	name := randName("grafana")
	yml := getTechnicalEmailsYaml(testProject, name, testPrimaryCloudName, true)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	grafana := new(v1alpha1.Grafana)
	require.NoError(t, s.GetRunning(grafana, name))

	// THEN
	// Technical emails are set
	grafanaAvn, err := avnClient.Services.Get(ctx, testProject, name)
	require.NoError(t, err)
	assert.Len(t, grafana.Spec.TechnicalEmails, 1)
	assert.Equal(t, "test@example.com", grafanaAvn.TechnicalEmails[0].Email)

	// WHEN
	// Technical emails are removed from manifest
	updatedYml := getTechnicalEmailsYaml(testProject, name, testPrimaryCloudName, false)

	// Applies updated manifest
	require.NoError(t, s.Apply(updatedYml))

	// Waits kube objects
	require.NoError(t, s.GetRunning(grafana, name))

	// THEN
	// Technical emails are removed from service
	grafanaAvnUpdated, err := avnClient.Services.Get(ctx, testProject, name)
	require.NoError(t, err)
	assert.Empty(t, grafanaAvnUpdated.TechnicalEmails)
}

func getConnInfoBaseYaml(project, name, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Grafana
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: startup-1
`, project, name, cloudName)
}

func getYamlWithDisabledOption(baseYaml string, disabledOption *bool) string {
	if disabledOption == nil {
		return baseYaml
	}
	return fmt.Sprintf("%s\n  connInfoSecretTargetDisabled: %t", baseYaml, *disabledOption)
}

func runTest(t *testing.T, scenario TestScenario) {
	defer recoverPanic(t)

	// GIVEN
	baseYaml := getConnInfoBaseYaml(testProject, scenario.serviceName, testPrimaryCloudName)
	createYaml := getYamlWithDisabledOption(baseYaml, scenario.connInfoSecretTargetDisabledChange[0])
	updateYaml := getYamlWithDisabledOption(baseYaml, scenario.connInfoSecretTargetDisabledChange[1])

	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(createYaml))

	// Waits kube objects
	grafana := new(v1alpha1.Grafana)
	require.NoError(t, s.GetRunning(grafana, scenario.serviceName))

	// THEN
	// Check if resource connection info secret is created
	secret, err := s.GetSecret(grafana.GetName())
	assert.Equal(t, scenario.expectSecretGetErrIsNil, err == nil)
	assert.Equal(t, scenario.expectSecretIsNil, secret == nil)

	// WHEN
	// Applies updated manifest
	err = s.Apply(updateYaml)
	expectedErrorMsg := fmt.Sprintf(scenario.expectedErrorMsg, scenario.serviceName)
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}

type TestScenario struct {
	testName                           string
	serviceName                        string
	connInfoSecretTargetDisabledChange []*bool
	expectSecretGetErrIsNil            bool
	expectSecretIsNil                  bool
	expectedErrorMsg                   string
}

func TestServiceConnInfoSecretTargetDisabled(t *testing.T) {
	cases := []TestScenario{
		{
			// Modifying `connInfoSecretTargetDisabled` from missing to true --> A secret is created, but the update fails
			// because `connInfoSecretTargetDisabled` can only be set during resource creation.
			testName:                           "modify from missing to true",
			serviceName:                        randName("grafana-modify-conn-info-secret-target-disabled-from-missing-to-true"),
			connInfoSecretTargetDisabledChange: []*bool{nil, pointer.Bool(true)},
			expectSecretGetErrIsNil:            true,
			expectSecretIsNil:                  false,
			expectedErrorMsg:                   `Grafana.aiven.io "%s" is invalid: spec: Invalid value: "object": connInfoSecretTargetDisabled can only be set during resource creation.`,
		},
		{
			// Modifying `connInfoSecretTargetDisabled` from true to missing --> A secret is not created and the update fails
			// because `connInfoSecretTargetDisabled` can only be set during resource creation.
			testName:                           "modify from true to missing",
			serviceName:                        randName("grafana-modify-conn-info-secret-target-disabled-from-true-to-missing"),
			connInfoSecretTargetDisabledChange: []*bool{pointer.Bool(true), nil},
			expectSecretGetErrIsNil:            false,
			expectSecretIsNil:                  true,
			expectedErrorMsg:                   `Grafana.aiven.io "%s" is invalid: spec: Invalid value: "object": connInfoSecretTargetDisabled can only be set during resource creation.`,
		},
		{
			// Modifying `connInfoSecretTargetDisabled` from false to true --> A secret is created and the update fails
			// due to the immutability of `connInfoSecretTargetDisabled`.
			testName:                           "modify from false to true",
			serviceName:                        randName("grafana-modify-conn-info-secret-target-disabled-from-false-to-true"),
			connInfoSecretTargetDisabledChange: []*bool{pointer.Bool(false), pointer.Bool(true)},
			expectSecretGetErrIsNil:            true,
			expectSecretIsNil:                  false,
			expectedErrorMsg:                   `Grafana.aiven.io "%s" is invalid: spec.connInfoSecretTargetDisabled: Invalid value: "boolean": connInfoSecretTargetDisabled is immutable.`,
		},
	}

	for _, opt := range cases {
		opt := opt
		t.Run(opt.testName, func(t *testing.T) {
			t.Parallel()
			runTest(t, opt)
		})
	}
}
