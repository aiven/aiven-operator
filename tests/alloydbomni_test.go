package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	alloydbomniuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/alloydbomni"
)

func getAlloyDBOmniYaml(project, name, cloudName, serviceAccountCredentials string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: AlloyDBOmni
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: startup-4
  disk_space: 90GiB
  serviceAccountCredentials: %q

  tags:
    env: test
    instance: foo

  userConfig:
    service_log: true
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16

`, project, name, cloudName, serviceAccountCredentials)
}

func TestAlloyDBOmni(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("alloydbomni")
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// Test cases
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
			yml := getAlloyDBOmniYaml(cfg.Project, name, cfg.PrimaryCloudName, tc.serviceAccountCredentials)

			// WHEN
			err := s.Apply(yml)

			// THEN
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorMessage)
			} else {
				require.NoError(t, err)

				// Waits kube objects
				cs := new(v1alpha1.AlloyDBOmni)
				require.NoError(t, s.GetRunning(cs, name))

				// Validate the resource
				csAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
				require.NoError(t, err)
				assert.Equal(t, csAvn.ServiceName, cs.GetName())
				assert.Equal(t, serviceRunningState, cs.Status.State)
				assert.Contains(t, serviceRunningStatesAiven, csAvn.State)
				assert.Equal(t, csAvn.Plan, cs.Spec.Plan)
				assert.Equal(t, csAvn.CloudName, cs.Spec.CloudName)
				assert.Equal(t, "90GiB", cs.Spec.DiskSpace)
				assert.Equal(t, int(92160), *csAvn.DiskSpaceMb)
				assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, cs.Spec.Tags)
				csResp, err := avnClient.ServiceTags.Get(ctx, cfg.Project, name)
				require.NoError(t, err)
				assert.Equal(t, csResp.Tags, cs.Spec.Tags)

				// UserConfig test
				require.NotNil(t, cs.Spec.UserConfig)
				require.NotNil(t, cs.Spec.UserConfig.ServiceLog)
				assert.Equal(t, anyPointer(true), cs.Spec.UserConfig.ServiceLog)

				// Validates ip filters
				require.Len(t, cs.Spec.UserConfig.IpFilter, 2)

				// First entry
				assert.Equal(t, "0.0.0.0/32", cs.Spec.UserConfig.IpFilter[0].Network)
				assert.Equal(t, "bar", *cs.Spec.UserConfig.IpFilter[0].Description)

				// Second entry
				assert.Equal(t, "10.20.0.0/16", cs.Spec.UserConfig.IpFilter[1].Network)
				assert.Nil(t, cs.Spec.UserConfig.IpFilter[1].Description)

				// Compares with Aiven ip_filter
				var ipFilterAvn []*alloydbomniuserconfig.IpFilter
				require.NoError(t, castInterface(csAvn.UserConfig["ip_filter"], &ipFilterAvn))
				assert.Equal(t, ipFilterAvn, cs.Spec.UserConfig.IpFilter)

				// Secrets test
				secret, err := s.GetSecret(cs.GetName())
				require.NoError(t, err)
				assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_HOST"])
				assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_PORT"])
				assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_DATABASE"])
				assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_USER"])
				assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_PASSWORD"])
				assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_SSLMODE"])
				assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_DATABASE_URI"])
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
