package tests

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	alloydbomniuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/alloydbomni"
)

func TestAlloyDBOmni(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	name := randName("alloydbomni")
	yml, err := loadExampleYaml("alloydbomni.yaml", map[string]string{
		"spec.cloudName": cfg.PrimaryCloudName,
		"spec.project":   cfg.Project,
		"metadata.name":  name,
	})
	require.NoError(t, err)

	// WHEN
	err = s.Apply(yml)

	// THEN
	require.NoError(t, err)

	// Waits kube objects
	adbo := new(v1alpha1.AlloyDBOmni)
	require.NoError(t, s.GetRunning(adbo, name))

	// Validate the resource
	adboAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, adboAvn.ServiceName, adbo.GetName())
	assert.Equal(t, serviceRunningState, adbo.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, adboAvn.State)
	assert.Equal(t, adboAvn.Plan, adbo.Spec.Plan)
	assert.Equal(t, adboAvn.CloudName, adbo.Spec.CloudName)
	assert.Equal(t, "90GiB", adbo.Spec.DiskSpace)
	assert.Equal(t, int(92160), *adboAvn.DiskSpaceMb)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, adbo.Spec.Tags)
	adboResp, err := avnClient.ServiceTags.Get(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, adboResp.Tags, adbo.Spec.Tags)

	// UserConfig test
	require.NotNil(t, adbo.Spec.UserConfig)
	require.NotNil(t, adbo.Spec.UserConfig.ServiceLog)
	assert.Equal(t, anyPointer(true), adbo.Spec.UserConfig.ServiceLog)

	// Validates ip filters
	require.Len(t, adbo.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", adbo.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *adbo.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", adbo.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, adbo.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*alloydbomniuserconfig.IpFilter
	require.NoError(t, castInterface(adboAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, adbo.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(adbo.Spec.ConnInfoSecretTarget.Name)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_HOST"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_PORT"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_DATABASE"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_USER"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_PASSWORD"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_SSLMODE"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_DATABASE_URI"])
}

func TestAlloyDBOmniServiceAccountCredentials(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("alloydbomni")
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// Test cases for service account credentials
	cases := []struct {
		name                      string
		serviceAccountCredentials string
		expectError               bool
		expectedErrorMessage      string
	}{
		{
			name:                      "valid credentials",
			serviceAccountCredentials: getTestServiceAccountCredentials("valid_key_id"),
			expectError:               false,
		},
		{
			name:                      "invalid credentials",
			serviceAccountCredentials: `{"private_key": "-----BEGIN PRIVATE KEY--.........----END PRIVATE KEY-----\n","client_email": "example@aiven.io","client_id": "example_user_id","type": "service_account","project_id": "example_project_id"}`,
			expectError:               true,
			expectedErrorMessage:      "invalid serviceAccountCredentials: (root): private_key_id is required",
		},
		{
			name:                      "empty credentials",
			serviceAccountCredentials: "",
			expectError:               false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			yml, err := loadExampleYaml("alloydbomni.yaml", map[string]string{
				"spec.cloudName":                 cfg.PrimaryCloudName,
				"spec.project":                   cfg.Project,
				"metadata.name":                  name,
				"spec.serviceAccountCredentials": tc.serviceAccountCredentials,
			})
			require.NoError(t, err)

			// WHEN
			err = s.Apply(yml)

			// THEN
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorMessage)
			} else {
				require.NoError(t, err)

				// Waits kube objects
				adbo := new(v1alpha1.AlloyDBOmni)
				require.NoError(t, s.GetRunning(adbo, name))

				// Validate the service account credentials
				rsp, err := avnGen.AlloyDbOmniGoogleCloudPrivateKeyIdentify(ctx, cfg.Project, name)
				require.NoError(t, err)
				if tc.serviceAccountCredentials == "" {
					assert.Empty(t, rsp.PrivateKeyId)
				} else {
					assert.Equal(t, getPrivateKeyID(tc.serviceAccountCredentials), rsp.PrivateKeyId)
				}
			}
		})
	}
}

func getTestServiceAccountCredentials(privateKeyID string) string {
	return fmt.Sprintf(`{
	  "private_key_id": %q,
	  "private_key": "-----BEGIN PRIVATE KEY--.........----END PRIVATE KEY-----\n",
	  "client_email": "example@aiven.io",
	  "client_id": "example_user_id",
	  "type": "service_account",
	  "project_id": "example_project_id"
	}`, privateKeyID)
}

func getPrivateKeyID(credentials string) string {
	// Extract the private_key_id from the credentials JSON string
	var data map[string]string
	json.Unmarshal([]byte(credentials), &data) // nolint:errcheck
	return data["private_key_id"]
}
