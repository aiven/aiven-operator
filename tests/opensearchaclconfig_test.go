//go:build opensearch

package tests

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	avnopensearch "github.com/aiven/go-client-codegen/handler/opensearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestOpenSearchACLConfig(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	serviceName := randName("opensearch")
	configName := randName("opensearch-acl")
	initialConfig := avnopensearch.OpensearchAclConfigOut{
		Enabled: true,
		Acls: []avnopensearch.AclOut{
			{
				Username: "admin*",
				Rules: []avnopensearch.RuleOut{
					{Index: "ind*", Permission: avnopensearch.PermissionTypeDeny},
					{Index: "logs*", Permission: avnopensearch.PermissionTypeRead},
				},
			},
			{
				Username: "ops*",
				Rules: []avnopensearch.RuleOut{
					{Index: "metrics*", Permission: avnopensearch.PermissionTypeWrite},
				},
			},
		},
	}
	updatedConfig := avnopensearch.OpensearchAclConfigOut{
		Enabled: true,
		Acls: []avnopensearch.AclOut{
			{
				Username: "writer*",
				Rules: []avnopensearch.RuleOut{
					{Index: "logs-prod*", Permission: avnopensearch.PermissionTypeReadwrite},
				},
			},
		},
	}
	disabledConfig := avnopensearch.OpensearchAclConfigOut{
		Enabled: false,
		Acls: []avnopensearch.AclOut{
			{
				Username: "writer*",
				Rules: []avnopensearch.RuleOut{
					{Index: "logs-prod*", Permission: avnopensearch.PermissionTypeReadwrite},
				},
			},
		},
	}
	resetConfig := avnopensearch.OpensearchAclConfigOut{
		Enabled: false,
		Acls:    []avnopensearch.AclOut{},
	}

	updatedACLConfigYml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: OpenSearchACLConfig
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  enabled: true
  acls:
    - username: writer*
      rules:
        - index: logs-prod*
          permission: readwrite
`, cfg.Project, serviceName, configName)
	disabledACLConfigYml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: OpenSearchACLConfig
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  enabled: false
  acls:
    - username: writer*
      rules:
        - index: logs-prod*
          permission: readwrite
`, cfg.Project, serviceName, configName)
	aclConfigYml, err := loadExampleYaml("opensearchaclconfig.yaml", map[string]string{
		"doc[0].metadata.name":    serviceName,
		"doc[0].spec.project":     cfg.Project,
		"doc[0].spec.cloudName":   cfg.PrimaryCloudName,
		"doc[1].metadata.name":    configName,
		"doc[1].spec.project":     cfg.Project,
		"doc[1].spec.serviceName": serviceName,
	})
	require.NoError(t, err)

	s := NewSession(ctx, k8sClient).(*session)
	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(aclConfigYml))

	os := new(v1alpha1.OpenSearch)
	require.NoError(t, s.GetRunning(os, serviceName))

	aclConfig := new(v1alpha1.OpenSearchACLConfig)
	require.NoError(t, s.GetRunning(aclConfig, configName))

	// THEN
	osAvn, err := avnGen.ServiceGet(ctx, cfg.Project, serviceName)
	require.NoError(t, err)
	assert.Equal(t, osAvn.ServiceName, os.GetName())
	assert.Equal(t, serviceRunningState, os.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, osAvn.State)

	requireOpenSearchACLConfigEventually(t, ctx, cfg.Project, serviceName, initialConfig, "OpenSearch ACL config should match the initial desired snapshot")

	require.NoError(t, s.Apply(updatedACLConfigYml))

	updatedACLConfig := new(v1alpha1.OpenSearchACLConfig)
	require.NoError(t, s.GetRunning(updatedACLConfig, configName))

	requireOpenSearchACLConfigEventually(t, ctx, cfg.Project, serviceName, updatedConfig, "OpenSearch ACL config should be replaced by the updated desired snapshot")

	require.NoError(t, s.Apply(disabledACLConfigYml))

	disabledACLConfig := new(v1alpha1.OpenSearchACLConfig)
	require.NoError(t, s.GetRunning(disabledACLConfig, configName))

	requireOpenSearchACLConfigEventually(t, ctx, cfg.Project, serviceName, disabledConfig, "OpenSearch ACL config should preserve ACLs when disabled")

	require.NoError(t, s.delete(disabledACLConfig))

	serviceAvn, err := avnGen.ServiceGet(ctx, cfg.Project, serviceName)
	require.NoError(t, err)
	assert.Equal(t, serviceName, serviceAvn.ServiceName)
	assert.Contains(t, serviceRunningStatesAiven, serviceAvn.State)

	currentOpenSearch := new(v1alpha1.OpenSearch)
	require.NoError(t, s.GetRunning(currentOpenSearch, serviceName))
	assert.Equal(t, serviceRunningState, currentOpenSearch.Status.State)

	requireOpenSearchACLConfigEventually(t, ctx, cfg.Project, serviceName, resetConfig, "OpenSearch ACL config should be reset on resource deletion")
}

func TestOpenSearchACLConfig_validation(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	testCases := []struct {
		name string
		yml  string
		err  string
	}{
		{
			name: "duplicate usernames denied",
			yml: `
apiVersion: aiven.io/v1alpha1
kind: OpenSearchACLConfig
metadata:
  name: duplicate-usernames
spec:
  project: ` + cfg.Project + `
  serviceName: test-service
  enabled: true
  acls:
    - username: admin*
      rules:
        - index: logs*
          permission: read
    - username: admin*
      rules:
        - index: metrics*
          permission: write
`,
			errSnippet: "duplicate",
		},
		{
			name: "missing rules denied",
			yml: `
apiVersion: aiven.io/v1alpha1
kind: OpenSearchACLConfig
metadata:
  name: missing-rules
spec:
  project: ` + cfg.Project + `
  serviceName: test-service
  enabled: true
  acls:
    - username: admin*
`,
			err: "rules",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := testCtx()
			defer cancel()

			s := NewSession(ctx, k8sClient)
			defer s.Destroy(t)

			err := s.Apply(tc.yml)
			assert.ErrorContains(t, err, tc.err)
		})
	}
}

func requireOpenSearchACLConfigEventually(
	t *testing.T,
	ctx context.Context,
	project string,
	serviceName string,
	expected avnopensearch.OpensearchAclConfigOut,
	message string,
) {
	t.Helper()

	require.Eventually(t, func() bool {
		current, err := avnGen.ServiceOpenSearchAclGet(ctx, project, serviceName)
		if err != nil {
			return false
		}

		return openSearchACLConfigEquals(current.OpensearchAclConfig, expected)
	}, 3*time.Minute, 10*time.Second, message)
}

func openSearchACLConfigEquals(actual, expected avnopensearch.OpensearchAclConfigOut) bool {
	normalizeACLs := func(acls []avnopensearch.AclOut) []avnopensearch.AclOut {
		out := make([]avnopensearch.AclOut, 0, len(acls))
		for _, acl := range acls {
			rules := append([]avnopensearch.RuleOut(nil), acl.Rules...)
			slices.SortFunc(rules, func(a, b avnopensearch.RuleOut) int {
				if diff := cmp.Compare(a.Index, b.Index); diff != 0 {
					return diff
				}

				return cmp.Compare(a.Permission, b.Permission)
			})

			out = append(out, avnopensearch.AclOut{
				Username: acl.Username,
				Rules:    rules,
			})
		}

		slices.SortFunc(out, func(a, b avnopensearch.AclOut) int {
			return cmp.Compare(a.Username, b.Username)
		})

		return out
	}
	return actual.Enabled == expected.Enabled &&
		slices.EqualFunc(
			normalizeACLs(actual.Acls),
			normalizeACLs(expected.Acls),
			func(a, b avnopensearch.AclOut) bool {
				return a.Username == b.Username && slices.Equal(a.Rules, b.Rules)
			},
		)
}
