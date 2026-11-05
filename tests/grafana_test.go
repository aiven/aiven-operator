//go:build grafana

package tests

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	grafanauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/grafana"
	"github.com/aiven/aiven-operator/controllers"
)

// grafanaTagsSpec is the tags block of the main Grafana manifest.
const grafanaTagsSpec = `
  tags:
    env: test
    instance: foo
`

// getGrafanaYaml returns a Grafana manifest; extraSpec is inserted into the spec as is.
func getGrafanaYaml(project, name, cloudName, extraSpec string) string {
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
%[4]s
  userConfig:
    alerting_enabled: true
    public_access:
      grafana: true
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16

`, project, name, cloudName, extraSpec)
}

// hasGrafanaEvent reports an event with the reason on the named Grafana whose message contains substr.
func hasGrafanaEvent(ctx context.Context, name, reason, substr string) bool {
	events := new(corev1.EventList)
	if err := k8sClient.List(ctx, events, client.InNamespace(defaultNamespace)); err != nil {
		return false
	}
	for _, e := range events.Items {
		if e.InvolvedObject.Kind == "Grafana" && e.InvolvedObject.Name == name &&
			e.Reason == reason && strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

func TestGrafana(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("grafana")
	yml := getGrafanaYaml(cfg.Project, name, cfg.PrimaryCloudName, grafanaTagsSpec)
	// Provisioned alongside the main service for the deletion subtest at the end.
	protectedName := randName("grafana-protected")
	protectedYml := getGrafanaYaml(cfg.Project, protectedName, cfg.PrimaryCloudName, `
  terminationProtection: true
  tags: {}
`)
	// Never created at Aiven.
	offName := randName("grafana-off")
	offYml := getGrafanaYaml(cfg.Project, offName, cfg.PrimaryCloudName, `
  powered: false
`)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))
	require.NoError(t, s.Apply(protectedYml))
	require.NoError(t, s.Apply(offYml))

	// Waits kube objects
	grafana := new(v1alpha1.Grafana)
	require.NoError(t, s.GetRunning(grafana, name))

	// THEN
	// Aiven may omit grafana_version from the metadata for a while after the service turns RUNNING
	var grafanaAvn *service.ServiceGetOut
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		var err error
		grafanaAvn, err = avnGen.ServiceGet(ctx, cfg.Project, name)
		require.NoError(collect, err)
		require.NoError(collect, k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: defaultNamespace}, grafana))
		assert.NotNil(collect, grafanaAvn.Metadata["grafana_version"])
		assert.NotEmpty(collect, grafana.Status.Version)
	}, 3*time.Minute, 10*time.Second)
	assert.Equal(t, grafanaAvn.ServiceName, grafana.GetName())
	assert.Equal(t, serviceRunningState, grafana.Status.State)
	assertServiceVersion(t, grafanaAvn.Metadata["grafana_version"], grafana.Status.Version)
	assert.Contains(t, serviceRunningStatesAiven, grafanaAvn.State)
	assert.Equal(t, grafanaAvn.Plan, grafana.Spec.Plan)
	assert.Equal(t, grafanaAvn.CloudName, grafana.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, grafana.Spec.Tags)
	// Tags are sent in the create request.
	grafanaTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, grafanaTags, grafana.Spec.Tags)

	// UserConfig test
	require.NotNil(t, grafana.Spec.UserConfig)
	assert.Equal(t, anyPointer(true), grafana.Spec.UserConfig.AlertingEnabled)

	// UserConfig nested options test
	require.NotNil(t, grafana.Spec.UserConfig.PublicAccess)
	assert.Equal(t, anyPointer(true), grafana.Spec.UserConfig.PublicAccess.Grafana)

	// Validates ip filters
	require.Len(t, grafana.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", grafana.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *grafana.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", grafana.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, grafana.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*grafanauserconfig.IpFilter
	require.NoError(t, castInterface(grafanaAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, grafana.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(grafana.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["GRAFANA_HOST"])
	assert.NotEmpty(t, secret.Data["GRAFANA_PORT"])
	assert.NotEmpty(t, secret.Data["GRAFANA_USER"])
	assert.NotEmpty(t, secret.Data["GRAFANA_PASSWORD"])
	assert.NotEmpty(t, secret.Data["GRAFANA_URI"])
	assert.NotEmpty(t, secret.Data["GRAFANA_HOSTS"])

	t.Run("a missing service is not created powered off", func(t *testing.T) {
		off := new(v1alpha1.Grafana)
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			require.NoError(collect, k8sClient.Get(ctx, types.NamespacedName{Name: offName, Namespace: defaultNamespace}, off))
			c := meta.FindStatusCondition(off.Status.Conditions, controllers.ConditionTypeError)
			require.NotNil(collect, c)
			assert.Contains(collect, c.Message, "cannot be created powered off")
		}, 3*time.Minute, 10*time.Second)
		_, err := avnGen.ServiceGet(ctx, cfg.Project, offName)
		assert.True(t, avngen.IsNotFound(err), "got %v", err)
	})

	t.Run("tags changed at Aiven are reverted", func(t *testing.T) {
		err := avnGen.ProjectServiceTagsReplace(ctx, cfg.Project, name, &service.ProjectServiceTagsReplaceIn{
			Tags: map[string]string{"env": "drifted"},
		})
		require.NoError(t, err)
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			tags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, name)
			require.NoError(collect, err)
			assert.Equal(collect, map[string]string{"env": "test", "instance": "foo"}, tags)
			assert.True(collect, hasGrafanaEvent(ctx, name, "DriftDetected", "tags: {env=drifted} (live)"))
		}, 5*time.Minute, 15*time.Second)
	})

	t.Run("omitted tags are left as they are at Aiven", func(t *testing.T) {
		require.NoError(t, s.Apply(getGrafanaYaml(cfg.Project, name, cfg.PrimaryCloudName, "")))
		require.NoError(t, s.GetRunning(grafana, name))
		assert.Nil(t, grafana.Spec.Tags)

		tags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, name)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, tags)
	})

	t.Run("tags: {} clears the tags at Aiven", func(t *testing.T) {
		require.NoError(t, s.Apply(getGrafanaYaml(cfg.Project, name, cfg.PrimaryCloudName, "\n  tags: {}\n")))
		require.NoError(t, s.GetRunning(grafana, name))
		require.NotNil(t, grafana.Spec.Tags)
		assert.Empty(t, grafana.Spec.Tags)

		tags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, name)
		require.NoError(t, err)
		assert.Empty(t, tags)
	})

	t.Run("a service deleted at Aiven is recreated", func(t *testing.T) {
		old, err := avnGen.ServiceGet(ctx, cfg.Project, name)
		require.NoError(t, err)
		require.NoError(t, avnGen.ServiceDelete(ctx, cfg.Project, name))

		// The resource stays ready until the next poll, so wait for the new service instead.
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			live, err := avnGen.ServiceGet(ctx, cfg.Project, name)
			require.NoError(collect, err)
			assert.True(collect, live.CreateTime.After(old.CreateTime))
			assert.Equal(collect, service.ServiceStateTypeRunning, live.State)
		}, waitRunningTimeout, 30*time.Second)
		require.NoError(t, s.GetRunning(grafana, name))
		assert.True(t, hasGrafanaEvent(ctx, name, "InstanceMissingAtAiven", "recreated from the spec"))

		// The connection Secret carries the new service's credentials.
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			live, err := avnGen.ServiceGet(ctx, cfg.Project, name, service.ServiceGetIncludeSecrets(true))
			require.NoError(collect, err)
			secret, err := s.GetSecret(name)
			require.NoError(collect, err)
			assert.Equal(collect, live.ServiceUriParams["password"], string(secret.Data["GRAFANA_PASSWORD"]))
		}, 3*time.Minute, 15*time.Second)
	})

	// Tests service power off functionality
	// Note: Power on testing is handled generically in generic_service_handler_test.go
	// since it's consistent across services. Power off testing is done here since
	// the flow can vary by service type and may require service-specific steps.
	require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: defaultNamespace}, grafana))
	poweredOff := grafana.DeepCopy()
	poweredOff.Spec.Powered = anyPointer(false)
	require.NoError(t, k8sClient.Update(ctx, poweredOff))
	require.NoError(t, s.GetRunning(poweredOff, name))

	poweredOffAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypePoweroff, poweredOffAvn.State)

	// The subtests below change the service behind the operator's back; CreateOnly keeps it from correcting that.
	patch := []byte(`{"metadata":{"annotations":{"controllers.aiven.io/management-policy":"CreateOnly"}}}`)
	require.NoError(t, k8sClient.Patch(ctx, poweredOff, client.RawPatch(types.MergePatchType, patch)))

	t.Run("CreateOnly reports drift without correcting it", func(t *testing.T) {
		drifted := map[string]string{"env": "drifted"}
		err := avnGen.ProjectServiceTagsReplace(ctx, cfg.Project, name, &service.ProjectServiceTagsReplaceIn{Tags: drifted})
		require.NoError(t, err)
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			require.NoError(collect, k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: defaultNamespace}, grafana))
			c := meta.FindStatusCondition(grafana.Status.Conditions, v1alpha1.ConditionTypeDrifted)
			require.NotNil(collect, c)
			assert.Contains(collect, c.Message, "tags")
		}, 5*time.Minute, 15*time.Second)

		tags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, name)
		require.NoError(t, err)
		assert.Equal(t, drifted, tags)
	})

	t.Run("CreateOnly ignores spec changes", func(t *testing.T) {
		patch := []byte(`{"spec":{"userConfig":{"alerting_enabled":false}}}`)
		require.NoError(t, k8sClient.Patch(ctx, grafana, client.RawPatch(types.MergePatchType, patch)))
		// The skipped generation is stamped as processed. Not asserted by event: the recorder's
		// spam filter drops events on an object with this many.
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			require.NoError(collect, k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: defaultNamespace}, grafana))
			assert.Equal(collect, strconv.FormatInt(grafana.Generation, 10), grafana.Annotations["controllers.aiven.io/generation-was-processed"])
		}, 3*time.Minute, 10*time.Second)

		live, err := avnGen.ServiceGet(ctx, cfg.Project, name)
		require.NoError(t, err)
		assert.Equal(t, true, live.UserConfig["alerting_enabled"])
	})

	// Drift is compared in RUNNING and POWEROFF only, so powering on must leave POWEROFF as soon as ServiceUpdate returns.
	t.Run("power on leaves POWEROFF immediately", func(t *testing.T) {
		_, err := avnGen.ServiceUpdate(ctx, cfg.Project, name, &service.ServiceUpdateIn{Powered: anyPointer(true)})
		require.NoError(t, err)
		live, err := avnGen.ServiceGet(ctx, cfg.Project, name)
		require.NoError(t, err)
		assert.NotEqual(t, service.ServiceStateTypePoweroff, live.State)
	})

	// The same for powering off: it must leave RUNNING as soon as ServiceUpdate returns.
	t.Run("power off leaves RUNNING immediately", func(t *testing.T) {
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			live, err := avnGen.ServiceGet(ctx, cfg.Project, name)
			require.NoError(collect, err)
			assert.Equal(collect, service.ServiceStateTypeRunning, live.State)
		}, 15*time.Minute, 15*time.Second)
		_, err := avnGen.ServiceUpdate(ctx, cfg.Project, name, &service.ServiceUpdateIn{Powered: anyPointer(false)})
		require.NoError(t, err)
		live, err := avnGen.ServiceGet(ctx, cfg.Project, name)
		require.NoError(t, err)
		assert.NotEqual(t, service.ServiceStateTypeRunning, live.State)
	})

	protected := new(v1alpha1.Grafana)
	require.NoError(t, s.GetRunning(protected, protectedName))

	// Finalizers are patched, so the first reconcile does not drop the empty map.
	t.Run("tags: {} in a new manifest survives the first reconcile", func(t *testing.T) {
		require.NotNil(t, protected.Spec.Tags)
		assert.Empty(t, protected.Spec.Tags)
	})

	t.Run("terminationProtection blocks deletion until it is switched off", func(t *testing.T) {
		live, err := avnGen.ServiceGet(ctx, cfg.Project, protectedName)
		require.NoError(t, err)
		require.True(t, live.TerminationProtection)

		// The validating webhook refuses the delete.
		err = k8sClient.Delete(ctx, protected)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "termination protection is on")
		_, err = avnGen.ServiceGet(ctx, cfg.Project, protectedName)
		require.NoError(t, err)

		patch := []byte(`{"spec":{"terminationProtection":false}}`)
		require.NoError(t, k8sClient.Patch(ctx, protected, client.RawPatch(types.MergePatchType, patch)))
		require.NoError(t, s.Delete(protected, func() error {
			_, err := avnGen.ServiceGet(ctx, cfg.Project, protectedName)
			return err
		}))
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			_, err := avnGen.ServiceGet(ctx, cfg.Project, protectedName)
			assert.True(collect, avngen.IsNotFound(err), "got %v", err)
		}, 10*time.Minute, 15*time.Second)
	})
}
