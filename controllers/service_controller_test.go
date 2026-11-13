package controllers

import (
	"context"
	"strconv"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	pguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/pg"
)

const yamlGrafanaSvc = `
apiVersion: aiven.io/v1alpha1
kind: Grafana
metadata:
  name: my-grafana
  namespace: default
spec:
  project: test-project
  plan: startup-1
  cloudName: google-europe-west1
`

// newTestController builds a controller for direct Observe/Create/Update/Delete calls. k8s may be
// nil when the method never touches the cluster.
func newTestController[T v1alpha1.ServiceObject](avn avngen.Client, kind serviceKind[T], k8s client.Client) *serviceController[T] {
	c, _ := newRecordingTestController(avn, kind, k8s)
	return c
}

// newRecordingTestController is newTestController with the event recorder exposed for assertions.
func newRecordingTestController[T v1alpha1.ServiceObject](avn avngen.Client, kind serviceKind[T], k8s client.Client) (*serviceController[T], *record.FakeRecorder) {
	rec := record.NewFakeRecorder(10)
	return newServiceController(Controller{
		Client:   k8s,
		Log:      logr.Discard(),
		Recorder: rec,
	}, avn, kind), rec
}

// inSync sets live's plan and cloud to the spec's, the owned fields the test specs set.
func inSync(obj v1alpha1.ServiceObject, live *service.ServiceGetOut) *service.ServiceGetOut {
	spec := obj.ServiceCommonSpec()
	live.Plan, live.CloudName = spec.Plan, spec.CloudName
	return live
}

// markGeneration stamps gen as both the generation and the processed generation.
func markGeneration(obj metav1.Object, gen int64) {
	obj.SetGeneration(gen)
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	ann[processedGenerationAnnotation] = strconv.FormatInt(gen, formatIntBaseDecimal)
	obj.SetAnnotations(ann)
}

func TestServiceController_Observe(t *testing.T) {
	t.Parallel()

	newRunningGrafana := func(t *testing.T) *v1alpha1.Grafana {
		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.Powered = new(true)
		markGeneration(gf, 1)
		return gf
	}

	t.Run("service not found -> ResourceExists false", func(t *testing.T) {
		t.Parallel()

		gf := newRunningGrafana(t)
		// Previously Ready: readiness must be revoked while Create runs.
		markInstanceRunning(gf)
		gf.Status.State = service.ServiceStateTypeRunning
		gf.Status.Version = "11.4"
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		c, rec := newRecordingTestController(avn, grafanaKind, nil)
		obs, err := c.Observe(t.Context(), gf)
		require.NoError(t, err)
		require.False(t, obs.ResourceExists)
		require.False(t, hasIsRunningAnnotation(gf))
		require.False(t, IsReadyToUse(gf))
		running := meta.FindStatusCondition(gf.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, running)
		require.Equal(t, metav1.ConditionFalse, running.Status)
		// State/Version cleared.
		require.Empty(t, gf.Status.State)
		require.Empty(t, gf.Status.Version)
		// A 404 may be a read-after-write blip; the recreate Warning belongs to Create.
		require.Empty(t, recorderEvents(rec))
	})

	t.Run("stale generation -> ResourceUpToDate false, live secret details published before Update", func(t *testing.T) {
		t.Parallel()

		gf := newRunningGrafana(t)
		gf.Generation = 2 // annotation still says "1" -> stale

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, [][2]string{service.ServiceGetIncludeSecrets(true)}).
			Return(&service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "grafana.example.com", "port": "3000", "user": "avnadmin", "password": "secret"},
				ServiceUri:       "https://grafana.example.com:3000",
				ConnectionInfo:   &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
			}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		obs, err := c.Observe(t.Context(), gf)
		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.False(t, obs.ResourceUpToDate)
		require.Equal(t, "grafana.example.com", obs.SecretDetails["GRAFANA_HOST"])
		require.True(t, obs.PruneSecretKeys)
		// Observed state is recorded even while not up to date.
		require.Equal(t, service.ServiceStateTypeRunning, gf.Status.State)
		// No running annotation on the update path.
		require.False(t, hasIsRunningAnnotation(gf))
	})

	t.Run("stale generation under management-policy CreateOnly -> secret details", func(t *testing.T) {
		t.Parallel()
		var _ managementPolicySupported = (*serviceController[*v1alpha1.Grafana])(nil)

		gf := newRunningGrafana(t)
		gf.Generation = 2 // annotation still says "1" -> stale
		ctx := withManagementPolicy(t.Context(), managementPolicyCreateOnly)

		avn := avngen.NewMockClient(t)
		// Secrets are fetched on the stale-generation path: they are published in place of the skipped Update.
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, [][2]string{service.ServiceGetIncludeSecrets(true)}).
			Return(&service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "grafana.example.com", "port": "3000", "user": "avnadmin", "password": "secret"},
				ServiceUri:       "https://grafana.example.com:3000",
				ConnectionInfo:   &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
			}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		obs, err := c.Observe(ctx, gf)
		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.False(t, obs.ResourceUpToDate)
		require.Equal(t, "grafana.example.com", obs.SecretDetails["GRAFANA_HOST"])
		require.True(t, obs.PruneSecretKeys)
	})

	t.Run("stale generation under management-policy CreateOnly while not running -> no secret details", func(t *testing.T) {
		t.Parallel()

		gf := newRunningGrafana(t)
		gf.Generation = 2
		ctx := withManagementPolicy(t.Context(), managementPolicyCreateOnly)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRebuilding}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		obs, err := c.Observe(ctx, gf)
		require.NoError(t, err)
		require.False(t, obs.ResourceUpToDate)
		require.Nil(t, obs.SecretDetails)
	})

	t.Run("ignored power off under management-policy CreateOnly -> live state wins", func(t *testing.T) {
		t.Parallel()

		running := &service.ServiceGetOut{
			State:            service.ServiceStateTypeRunning,
			ServiceUriParams: map[string]string{"host": "grafana.example.com", "port": "3000", "user": "avnadmin", "password": "secret"},
			ServiceUri:       "https://grafana.example.com:3000",
			ConnectionInfo:   &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
		}
		ctx := withManagementPolicy(t.Context(), managementPolicyCreateOnly)

		t.Run("stale generation without backups -> no precondition calls, secret details", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			gf.Spec.Powered = new(false)
			gf.Generation = 2

			avn := avngen.NewMockClient(t)
			// No ServiceBackupsGet: the power-off it guards is never applied.
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, [][2]string{service.ServiceGetIncludeSecrets(true)}).
				Return(running, nil).Once()

			c := newTestController(avn, grafanaKind, nil)
			obs, err := c.Observe(ctx, gf)
			require.NoError(t, err)
			require.False(t, obs.ResourceUpToDate)
			require.Equal(t, "grafana.example.com", obs.SecretDetails["GRAFANA_HOST"])
		})

		t.Run("processed generation -> Drifted powered + running annotation + secret details", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			gf.Spec.Powered = new(false)

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, [][2]string{service.ServiceGetIncludeSecrets(true)}).
				Return(inSync(gf, running), nil).Once()

			c := newTestController(avn, grafanaKind, nil)
			obs, err := c.Observe(ctx, gf)
			require.NoError(t, err)
			require.False(t, obs.ResourceUpToDate)
			require.Equal(t, "grafana.example.com", obs.SecretDetails["GRAFANA_HOST"])
			require.Equal(t, "true", gf.Annotations[instanceIsRunningAnnotation])
			require.Equal(t, "powered", meta.FindStatusCondition(gf.Status.Conditions, v1alpha1.ConditionTypeDrifted).Message)
		})
	})

	t.Run("powered off at Aiven under management-policy CreateOnly -> Drifted powered + markInstancePoweredOff", func(t *testing.T) {
		t.Parallel()

		gf := newRunningGrafana(t)
		ctx := withManagementPolicy(t.Context(), managementPolicyCreateOnly)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, [][2]string{service.ServiceGetIncludeSecrets(true)}).
			Return(inSync(gf, &service.ServiceGetOut{State: service.ServiceStateTypePoweroff}), nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		obs, err := c.Observe(ctx, gf)
		require.NoError(t, err)
		require.False(t, obs.ResourceUpToDate)
		require.Nil(t, obs.SecretDetails)
		require.Equal(t, "false", gf.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "powered", meta.FindStatusCondition(gf.Status.Conditions, v1alpha1.ConditionTypeDrifted).Message)
	})

	t.Run("running -> running annotation + secret details", func(t *testing.T) {
		t.Parallel()

		gf := newRunningGrafana(t)
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, [][2]string{service.ServiceGetIncludeSecrets(true)}).
			Return(inSync(gf, &service.ServiceGetOut{
				State: service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{
					"host":     "grafana.example.com",
					"port":     "3000",
					"user":     "avnadmin",
					"password": "secret",
				},
				ServiceUri:     "https://grafana.example.com:3000",
				ConnectionInfo: &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
				Metadata:       map[string]any{"grafana_version": "11.4"},
			}), nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		obs, err := c.Observe(t.Context(), gf)
		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.True(t, obs.ResourceUpToDate)
		require.Equal(t, "true", gf.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "11.4", gf.Status.Version)
		require.Equal(t, "grafana.example.com", obs.SecretDetails["GRAFANA_HOST"])
		require.Equal(t, "3000", obs.SecretDetails["GRAFANA_PORT"])
		require.Equal(t, "https://grafana.example.com:3000", obs.SecretDetails["GRAFANA_URI"])
		// Services rebuild the secret from scratch, matching the legacy handler.
		require.True(t, obs.PruneSecretKeys)
		require.Zero(t, obs.RequeueAfter)
	})

	t.Run("not yet running -> no annotation, no details", func(t *testing.T) {
		t.Parallel()

		gf := newRunningGrafana(t)
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRebuilding}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		obs, err := c.Observe(t.Context(), gf)
		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.True(t, obs.ResourceUpToDate)
		require.False(t, hasIsRunningAnnotation(gf))
		require.Nil(t, obs.SecretDetails)
	})

	t.Run("powered off -> markInstancePoweredOff, no secret details, no precondition calls", func(t *testing.T) {
		t.Parallel()

		gf := newRunningGrafana(t)
		gf.Spec.Powered = new(false)

		avn := avngen.NewMockClient(t)
		// Steady state: no backup check, no secrets fetch.
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(inSync(gf, &service.ServiceGetOut{State: service.ServiceStateTypePoweroff}), nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		obs, err := c.Observe(t.Context(), gf)
		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.True(t, obs.ResourceUpToDate)
		require.Nil(t, obs.SecretDetails)
		require.Equal(t, "false", gf.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("drift", func(t *testing.T) {
		t.Parallel()

		t.Run("running -> Warning per field, not up to date, live secret details published before Update", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			markInstanceRunning(gf)
			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, [][2]string{service.ServiceGetIncludeSecrets(true)}).
				Return(&service.ServiceGetOut{
					State:            service.ServiceStateTypeRunning,
					Plan:             "startup-4",
					CloudName:        "aws-eu-west-1",
					ServiceUriParams: map[string]string{"host": "grafana.example.com", "password": "rotated"},
					ConnectionInfo:   &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
				}, nil).Once()

			c, rec := newRecordingTestController(avn, grafanaKind, nil)
			obs, err := c.Observe(t.Context(), gf)
			require.NoError(t, err)
			require.True(t, obs.ResourceExists)
			require.False(t, obs.ResourceUpToDate)
			// A failing correction must not leave the Secret stale: the live credentials go out first.
			require.True(t, obs.PruneSecretKeys)
			require.Equal(t, "rotated", obs.SecretDetails["GRAFANA_PASSWORD"])
			require.Equal(t, []string{
				"Warning DriftDetected plan: startup-4 (live) -> startup-1 (spec)",
				"Warning DriftDetected cloudName: aws-eu-west-1 (live) -> google-europe-west1 (spec)",
			}, recorderEvents(rec))
			// Readiness is revoked by the correcting Update, not by Observe.
			require.True(t, hasIsRunningAnnotation(gf))
		})

		t.Run("not compared while REBALANCING", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
				Return(&service.ServiceGetOut{State: service.ServiceStateTypeRebalancing, Plan: "startup-4"}, nil).Once()

			c, rec := newRecordingTestController(avn, grafanaKind, nil)
			obs, err := c.Observe(t.Context(), gf)
			require.NoError(t, err)
			require.True(t, obs.ResourceUpToDate)
			require.Empty(t, recorderEvents(rec))
		})

		t.Run("a field omitted from the spec is unmanaged", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
				Return(inSync(gf, &service.ServiceGetOut{
					State:          service.ServiceStateTypeRunning,
					Tags:           map[string]string{"console": "yes"},
					ConnectionInfo: &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
				}), nil).Once()

			c, rec := newRecordingTestController(avn, grafanaKind, nil)
			obs, err := c.Observe(t.Context(), gf)
			require.NoError(t, err)
			require.True(t, obs.ResourceUpToDate)
			require.NotNil(t, obs.SecretDetails)
			require.Empty(t, recorderEvents(rec))
		})

		t.Run("Clickhouse plan reported under another name -> up to date, running", func(t *testing.T) {
			t.Parallel()

			ch := &v1alpha1.Clickhouse{ObjectMeta: metav1.ObjectMeta{Name: "my-clickhouse", Namespace: "default"}}
			ch.Spec.Project = "test-project"
			ch.Spec.Plan = "startup-v2-16"
			ch.Spec.CloudName = "google-europe-west1"
			markGeneration(ch, 1)
			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, ch.Spec.Project, ch.Name, mock.Anything).
				Return(&service.ServiceGetOut{
					State:            service.ServiceStateTypeRunning,
					Plan:             "startup-16",
					CloudName:        ch.Spec.CloudName,
					ServiceUriParams: map[string]string{"host": "ch.example.com", "port": "1", "user": "u", "password": "p"},
				}, nil).Once()

			c, rec := newRecordingTestController(avn, clickhouseKind, nil)
			obs, err := c.Observe(t.Context(), ch)
			require.NoError(t, err)
			require.True(t, obs.ResourceUpToDate)
			require.Empty(t, recorderEvents(rec))
			require.True(t, hasIsRunningAnnotation(ch))
		})

		t.Run("powered off in the Console -> powered drift, not marked powered off", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			markInstanceRunning(gf)
			avn := avngen.NewMockClient(t)
			// Spec powered: no backup check.
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
				Return(inSync(gf, &service.ServiceGetOut{State: service.ServiceStateTypePoweroff}), nil).Once()

			c, rec := newRecordingTestController(avn, grafanaKind, nil)
			obs, err := c.Observe(t.Context(), gf)
			require.NoError(t, err)
			require.False(t, obs.ResourceUpToDate)
			require.Equal(t, []string{"Warning DriftDetected powered: false (live) -> true (spec)"}, recorderEvents(rec))
			require.Equal(t, "true", gf.Annotations[instanceIsRunningAnnotation])
		})

		t.Run("powered: false while running -> powered drift, no backup check in Observe", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			gf.Spec.Powered = new(false)
			avn := avngen.NewMockClient(t)
			// Spec powered off: no secrets fetched, no ServiceBackupsGet. The precondition gates Update.
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
				Return(inSync(gf, &service.ServiceGetOut{State: service.ServiceStateTypeRunning}), nil).Once()

			c, rec := newRecordingTestController(avn, grafanaKind, nil)
			obs, err := c.Observe(t.Context(), gf)
			require.NoError(t, err)
			require.False(t, obs.ResourceUpToDate)
			require.Equal(t, []string{"Warning DriftDetected powered: true (live) -> false (spec)"}, recorderEvents(rec))
		})

		t.Run("projectVPCRef is resolved and compared", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			gf.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "test-vpc"}
			vpc := &v1alpha1.ProjectVPC{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vpc", Namespace: "default"},
				Status:     v1alpha1.ProjectVPCStatus{ID: "vpc-2"},
			}
			k8s := fake.NewClientBuilder().WithScheme(newLifecycleScheme(t)).WithObjects(vpc).Build()

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
				Return(inSync(gf, &service.ServiceGetOut{
					State:          service.ServiceStateTypeRunning,
					ProjectVpcId:   "vpc-1",
					ConnectionInfo: &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
				}), nil).Once()

			c, rec := newRecordingTestController(avn, grafanaKind, k8s)
			obs, err := c.Observe(t.Context(), gf)
			require.NoError(t, err)
			require.False(t, obs.ResourceUpToDate)
			require.Equal(t, []string{"Warning DriftDetected projectVpcId: vpc-1 (live) -> vpc-2 (spec)"}, recorderEvents(rec))
		})

		t.Run("projectVPCRef without an ID -> errPreconditionNotMet", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			gf.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "test-vpc"}
			vpc := &v1alpha1.ProjectVPC{ObjectMeta: metav1.ObjectMeta{Name: "test-vpc", Namespace: "default"}}
			k8s := fake.NewClientBuilder().WithScheme(newLifecycleScheme(t)).WithObjects(vpc).Build()

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
				Return(inSync(gf, &service.ServiceGetOut{State: service.ServiceStateTypeRunning}), nil).Once()

			c := newTestController(avn, grafanaKind, k8s)
			_, err := c.Observe(t.Context(), gf)
			require.ErrorIs(t, err, errPreconditionNotMet)
		})

		t.Run("CreateOnly -> Drifted condition, no events, secret details", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			ctx := withManagementPolicy(t.Context(), managementPolicyCreateOnly)
			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, [][2]string{service.ServiceGetIncludeSecrets(true)}).
				Return(&service.ServiceGetOut{
					State:          service.ServiceStateTypeRunning,
					Plan:           "startup-4",
					CloudName:      "aws-eu-west-1",
					ConnectionInfo: &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
				}, nil).Once()

			c, rec := newRecordingTestController(avn, grafanaKind, nil)
			obs, err := c.Observe(ctx, gf)
			require.NoError(t, err)
			require.False(t, obs.ResourceUpToDate)
			require.NotNil(t, obs.SecretDetails)
			require.True(t, obs.PruneSecretKeys)
			require.Empty(t, recorderEvents(rec))
			require.Equal(t, "true", gf.Annotations[instanceIsRunningAnnotation])
			drifted := meta.FindStatusCondition(gf.Status.Conditions, v1alpha1.ConditionTypeDrifted)
			require.NotNil(t, drifted)
			require.Equal(t, metav1.ConditionTrue, drifted.Status)
			require.Equal(t, "plan, cloudName", drifted.Message)
		})

		t.Run("Drifted is removed once live matches, under either policy", func(t *testing.T) {
			t.Parallel()

			for _, policy := range []managementPolicy{managementPolicyCreateOnly, managementPolicyFull} {
				t.Run(string(policy), func(t *testing.T) {
					t.Parallel()

					gf := newRunningGrafana(t)
					setDriftedCondition(&gf.Status.Conditions, []fieldDrift{{name: "plan"}})
					avn := avngen.NewMockClient(t)
					avn.EXPECT().
						ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
						Return(inSync(gf, &service.ServiceGetOut{
							State:          service.ServiceStateTypeRunning,
							ConnectionInfo: &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
						}), nil).Once()

					c := newTestController(avn, grafanaKind, nil)
					obs, err := c.Observe(withManagementPolicy(t.Context(), policy), gf)
					require.NoError(t, err)
					require.True(t, obs.ResourceUpToDate)
					require.Nil(t, meta.FindStatusCondition(gf.Status.Conditions, v1alpha1.ConditionTypeDrifted))
				})
			}
		})

		t.Run("Full drops Drifted when the policy is lifted while live still differs", func(t *testing.T) {
			t.Parallel()

			gf := newRunningGrafana(t)
			setDriftedCondition(&gf.Status.Conditions, []fieldDrift{{name: "plan"}})
			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
				Return(&service.ServiceGetOut{
					State:          service.ServiceStateTypeRunning,
					Plan:           "startup-4",
					CloudName:      "google-europe-west1",
					ConnectionInfo: &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
				}, nil).Once()

			c := newTestController(avn, grafanaKind, nil)
			obs, err := c.Observe(t.Context(), gf)
			require.NoError(t, err)
			require.False(t, obs.ResourceUpToDate)
			require.Nil(t, meta.FindStatusCondition(gf.Status.Conditions, v1alpha1.ConditionTypeDrifted))
		})
	})

	t.Run("powered off and missing at Aiven -> hard error before the backup check", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name    string
			applied bool
		}{
			{name: "never created"},
			{name: "applied service deleted at Aiven", applied: true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
				gf.Spec.Powered = new(false)
				if tc.applied {
					markGeneration(gf, 1)
					markInstancePoweredOff(gf)
				}

				// No ServiceBackupsGet: it would 404 on the missing service.
				avn := avngen.NewMockClient(t)
				avn.EXPECT().
					ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
					Return(nil, newAivenError(404, "service not found")).Once()

				c := newTestController(avn, grafanaKind, nil)
				_, err := c.Observe(t.Context(), gf)
				require.ErrorIs(t, err, errServiceCreatePoweredOff)
				// Neither a soft requeue nor a retryable error.
				require.NotErrorIs(t, err, errPreconditionNotMet)
				require.False(t, isRetryableAivenError(err))
				// A deleted powered-off service is no longer Ready.
				require.False(t, hasIsRunningAnnotation(gf))
			})
		}
	})

	t.Run("ProjectKmsGetCA error -> errConnInfoSecret, conditions untouched", func(t *testing.T) {
		t.Parallel()

		pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
		pg.Spec.Powered = new(true)
		markGeneration(pg, 1)
		markInstanceRunning(pg)
		before := pg.DeepCopy()

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
			Return(inSync(pg, &service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "pg.example.com", "port": "5432"},
			}), nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, pg.Spec.Project).
			Return("", newAivenError(500, "kms boom")).Once()

		c := newTestController(avn, postgreSQLKind, nil)
		_, err := c.Observe(t.Context(), pg)
		require.ErrorIs(t, err, errConnInfoSecret)
		require.ErrorContains(t, err, "cannot retrieve project CA certificate")

		// The reconciler owns readiness and the Error condition; only State is recorded.
		before.Status.State = service.ServiceStateTypeRunning
		require.Equal(t, before.Annotations, pg.Annotations)
		require.Equal(t, before.Status, pg.Status)
	})

	t.Run("nil secret -> markInstanceRunning, nil details", func(t *testing.T) {
		t.Parallel()

		// KafkaConnect's adapter returns a nil secret, exercising the secret == nil branch.
		kc := &v1alpha1.KafkaConnect{
			ObjectMeta: metav1.ObjectMeta{Name: "kafka-connect", Namespace: "default", Generation: 1},
			Spec: v1alpha1.KafkaConnectSpec{
				BaseServiceFields: v1alpha1.BaseServiceFields{
					ProjectDependant: v1alpha1.ProjectDependant{ProjectField: v1alpha1.ProjectField{Project: "test-project"}},
					Plan:             "business-4",
				},
			},
		}
		markGeneration(kc, 1)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, kc.Spec.Project, kc.Name, mock.Anything).
			Return(inSync(kc, &service.ServiceGetOut{State: service.ServiceStateTypeRunning}), nil).Once()

		c := newTestController(avn, kafkaConnectKind, nil)
		obs, err := c.Observe(t.Context(), kc)
		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.True(t, obs.ResourceUpToDate)
		require.Nil(t, obs.SecretDetails)
		require.Equal(t, "true", kc.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("NoSecret disabled -> event fired once per transition to running, nil details", func(t *testing.T) {
		t.Parallel()

		gf := newRunningGrafana(t)
		gf.Spec.ConnInfoSecretTargetDisabled = new(true)

		avn := avngen.NewMockClient(t)
		// No include-secrets query: secret creation is disabled for the resource.
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(inSync(gf, &service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "grafana.example.com", "port": "3000"},
				ConnectionInfo:   &service.ConnectionInfoOut{Grafana: []string{"grafana.example.com"}},
			}), nil).Twice()

		c, rec := newRecordingTestController(avn, grafanaKind, nil)

		// First poll after Create/Update: not yet marked running, the event explains the missing Secret.
		obs, err := c.Observe(t.Context(), gf)
		require.NoError(t, err)
		require.True(t, obs.ResourceUpToDate)
		require.Nil(t, obs.SecretDetails)
		require.Equal(t, "true", gf.Annotations[instanceIsRunningAnnotation])
		events := recorderEvents(rec)
		require.Len(t, events, 1)
		require.Contains(t, events[0], eventConnInfoSecretCreationDisabled)

		// Steady-state poll: already running, no repeat.
		obs, err = c.Observe(t.Context(), gf)
		require.NoError(t, err)
		require.Nil(t, obs.SecretDetails)
		require.Empty(t, recorderEvents(rec))
	})

	t.Run("NoSecret disabled -> adapter never sees the secret-less response", func(t *testing.T) {
		t.Parallel()

		// Without include_secrets the Kafka adapter would panic on the missing certs.
		kf := newObjectFromYAML[v1alpha1.Kafka](t, `
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: my-kafka
  namespace: default
spec:
  project: test-project
  plan: startup-2
  cloudName: google-europe-west1
  powered: true
  connInfoSecretTargetDisabled: true
`)
		markGeneration(kf, 1)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, kf.Spec.Project, kf.Name).
			Return(inSync(kf, &service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "kafka.example.com", "port": "9092"},
				ConnectionInfo:   &service.ConnectionInfoOut{},
			}), nil).Once()

		c := newTestController(avn, kafkaKind, nil)
		obs, err := c.Observe(t.Context(), kf)
		require.NoError(t, err)
		require.True(t, obs.ResourceUpToDate)
		require.Nil(t, obs.SecretDetails)
		require.Equal(t, "true", kf.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("ServiceGet errors", func(t *testing.T) {
		t.Parallel()

		t.Run("ServiceGet non-404 error propagates", func(t *testing.T) {
			t.Parallel()

			gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
			gf.Spec.Powered = new(true)
			markGeneration(gf, 1)

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
				Return(nil, newAivenError(500, "boom")).Once()

			c := newTestController(avn, grafanaKind, nil)
			_, err := c.Observe(t.Context(), gf)
			require.ErrorContains(t, err, "failed to get service from Aiven")
		})
	})

	t.Run("read replica preconditions", func(t *testing.T) {
		t.Parallel()

		// newReplica is a not-yet-created replica: the source checks gate its Create.
		newReplica := func(t *testing.T) *v1alpha1.Grafana {
			gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
			gf.Spec.Powered = new(true)
			gf.Spec.ServiceIntegrations = []*v1alpha1.ServiceIntegrationItem{
				{IntegrationType: service.IntegrationTypeReadReplica, SourceServiceName: "source-svc"},
			}
			return gf
		}

		expectReplicaMissing := func(avn *avngen.MockClient, gf *v1alpha1.Grafana) {
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
				Return(nil, newAivenError(404, "not found")).Once()
		}

		t.Run("source not operational -> errPreconditionNotMet, not external", func(t *testing.T) {
			t.Parallel()

			gf := newReplica(t)
			avn := avngen.NewMockClient(t)
			expectReplicaMissing(avn, gf)
			// checkServiceIsOperational fetches the source (without secrets); still rebuilding.
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, "source-svc").
				Return(&service.ServiceGetOut{State: service.ServiceStateTypeRebuilding}, nil).Once()

			c := newTestController(avn, grafanaKind, nil)
			_, err := c.Observe(t.Context(), gf)
			// The source starts on its own: keep the short requeue.
			require.ErrorIs(t, err, errPreconditionNotMet)
			require.NotErrorIs(t, err, errPreconditionExternal)
		})

		t.Run("source running but no backups -> errPreconditionExternal", func(t *testing.T) {
			t.Parallel()

			gf := newReplica(t)
			avn := avngen.NewMockClient(t)
			expectReplicaMissing(avn, gf)
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, "source-svc").
				Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()
			avn.EXPECT().
				ServiceBackupsGet(mock.Anything, gf.Spec.Project, "source-svc").
				Return(&service.ServiceBackupsGetOut{Backups: []service.BackupOut{}}, nil).Once()

			c := newTestController(avn, grafanaKind, nil)
			_, err := c.Observe(t.Context(), gf)
			require.ErrorIs(t, err, errPreconditionExternal)
		})

		t.Run("running replica in steady state skips the source checks", func(t *testing.T) {
			t.Parallel()

			gf := newReplica(t)
			markGeneration(gf, 1)
			avn := avngen.NewMockClient(t)
			// Only the replica's own fetch: no source ServiceGet or ServiceBackupsGet per poll.
			avn.EXPECT().
				ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, [][2]string{service.ServiceGetIncludeSecrets(true)}).
				Return(inSync(gf, &service.ServiceGetOut{
					State:          service.ServiceStateTypeRunning,
					ConnectionInfo: &service.ConnectionInfoOut{Grafana: []string{"replica.example.com"}},
				}), nil).Once()

			c := newTestController(avn, grafanaKind, nil)
			obs, err := c.Observe(t.Context(), gf)
			require.NoError(t, err)
			require.True(t, obs.ResourceUpToDate)
			require.Equal(t, "true", gf.Annotations[instanceIsRunningAnnotation])
		})
	})

	t.Run("KafkaConnect is exempt from the power-off backup check", func(t *testing.T) {
		t.Parallel()

		kc := &v1alpha1.KafkaConnect{
			ObjectMeta: metav1.ObjectMeta{Name: "kafka-connect", Namespace: "default", Generation: 1},
			Spec: v1alpha1.KafkaConnectSpec{
				BaseServiceFields: v1alpha1.BaseServiceFields{
					ProjectDependant: v1alpha1.ProjectDependant{ProjectField: v1alpha1.ProjectField{Project: "test-project"}},
					Plan:             "business-4",
				},
			},
		}

		avn := avngen.NewMockClient(t)
		// No ServiceBackupsGet: KafkaConnect is exempt from the power-off backup check.
		avn.EXPECT().
			ServiceGet(mock.Anything, kc.Spec.Project, kc.Name, mock.Anything).
			Return(nil, newAivenError(404, "not found")).Once()

		c := newTestController(avn, kafkaConnectKind, nil)
		obs, err := c.Observe(t.Context(), kc)
		require.NoError(t, err)
		require.False(t, obs.ResourceExists)
	})

	t.Run("migration", func(t *testing.T) {
		t.Parallel()

		scheme := runtime.NewScheme()
		require.NoError(t, corev1.AddToScheme(scheme))

		runningPGOut := &service.ServiceGetOut{
			Plan:  "hobby",
			State: service.ServiceStateTypeRunning,
			ServiceUriParams: map[string]string{
				"host":     "pg.example.com",
				"port":     "5432",
				"dbname":   "defaultdb",
				"user":     "avnadmin",
				"password": "secret",
				"sslmode":  "require",
			},
			ServiceUri: "postgres://avnadmin:secret@pg.example.com:5432/defaultdb?sslmode=require",
		}

		newMigratingPG := func(t *testing.T) *v1alpha1.PostgreSQL {
			pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
			pg.Namespace = "default"
			pg.Spec.Powered = new(true)
			pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: "creds"}
			markGeneration(pg, 1)
			return pg
		}

		t.Run("migration in progress -> InProgress condition + 10s requeue", func(t *testing.T) {
			t.Parallel()

			pg := newMigratingPG(t)
			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
				Return(runningPGOut, nil).Once()
			avn.EXPECT().
				ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
				Return(&service.ServiceGetMigrationStatusOut{Migration: service.MigrationOut{Status: "syncing"}}, nil).Once()
			avn.EXPECT().
				ProjectKmsGetCA(mock.Anything, pg.Spec.Project).Return("ca-cert", nil).Once()

			c := newTestController(avn, postgreSQLKind, nil)
			obs, err := c.Observe(t.Context(), pg)
			require.NoError(t, err)
			require.True(t, obs.ResourceUpToDate)
			require.Equal(t, requeueTimeout, obs.RequeueAfter)

			cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
			require.NotNil(t, cond)
			require.Equal(t, metav1.ConditionFalse, cond.Status)
			require.Equal(t, v1alpha1.MigrationReasonInProgress, cond.Reason)
		})

		t.Run("migration done -> True condition, no forced requeue, secret cleaned up", func(t *testing.T) {
			t.Parallel()

			pg := newMigratingPG(t)
			pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: "creds", DeleteAfterMigration: true}

			secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "default"}}
			k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
				Return(runningPGOut, nil).Once()
			avn.EXPECT().
				ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
				Return(&service.ServiceGetMigrationStatusOut{Migration: service.MigrationOut{Status: service.MigrationStatusTypeDone}}, nil).Once()
			avn.EXPECT().
				ProjectKmsGetCA(mock.Anything, pg.Spec.Project).Return("ca-cert", nil).Once()

			c := newTestController(avn, postgreSQLKind, k8s)
			obs, err := c.Observe(t.Context(), pg)
			require.NoError(t, err)
			require.Zero(t, obs.RequeueAfter)

			cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
			require.NotNil(t, cond)
			require.Equal(t, metav1.ConditionTrue, cond.Status)

			err = k8s.Get(t.Context(), types.NamespacedName{Name: "creds", Namespace: "default"}, &corev1.Secret{})
			require.Error(t, err, "migration secret should have been deleted")
		})

		t.Run("migration status 404 after the config was applied -> True condition (already cleaned up)", func(t *testing.T) {
			t.Parallel()

			pg := newMigratingPG(t)
			markMigrationApplied(&pg.Status.Conditions, pg.Generation)
			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
				Return(runningPGOut, nil).Once()
			avn.EXPECT().
				ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
				Return(nil, newAivenError(404, "no active migration")).Once()
			avn.EXPECT().
				ProjectKmsGetCA(mock.Anything, pg.Spec.Project).Return("ca-cert", nil).Once()

			c := newTestController(avn, postgreSQLKind, nil)
			obs, err := c.Observe(t.Context(), pg)
			require.NoError(t, err)
			require.Zero(t, obs.RequeueAfter)

			cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
			require.NotNil(t, cond)
			require.Equal(t, metav1.ConditionTrue, cond.Status)
		})

		t.Run("migration status 404 under management-policy CreateOnly -> not complete, secret kept", func(t *testing.T) {
			t.Parallel()

			pg := newMigratingPG(t)
			pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: "creds", DeleteAfterMigration: true}
			ctx := withManagementPolicy(t.Context(), managementPolicyCreateOnly)

			secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "default"}}
			k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
				Return(runningPGOut, nil).Once()
			avn.EXPECT().
				ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
				Return(nil, newAivenError(404, "no active migration")).Maybe()
			avn.EXPECT().
				ProjectKmsGetCA(mock.Anything, pg.Spec.Project).Return("ca-cert", nil).Maybe()

			c := newTestController(avn, postgreSQLKind, k8s)
			_, err := c.Observe(ctx, pg)
			require.NoError(t, err)

			require.False(t, meta.IsStatusConditionTrue(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete),
				"a migration CreateOnly never applied must not be reported complete")
			require.NoError(t, k8s.Get(t.Context(), types.NamespacedName{Name: "creds", Namespace: "default"}, &corev1.Secret{}),
				"the unused migration secret must be kept")
		})
	})

	// The flattened SecretDetails must match exactly what the legacy adapter.newSecret produced,
	// including the injected CA_CERT keys.
	t.Run("secret parity with the legacy adapter", func(t *testing.T) {
		t.Parallel()

		t.Run("postgresql prefixed CA_CERT", func(t *testing.T) {
			t.Parallel()

			pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
			pg.Spec.Powered = new(true)
			markGeneration(pg, 1)

			svcOut := &service.ServiceGetOut{
				State: service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{
					"host":     "pg.example.com",
					"port":     "5432",
					"dbname":   "defaultdb",
					"user":     "avnadmin",
					"password": "secret",
					"sslmode":  "require",
				},
				ServiceUri: "postgres://avnadmin:secret@pg.example.com:5432/defaultdb?sslmode=require",
			}

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
				Return(inSync(pg, svcOut), nil).Once()
			avn.EXPECT().
				ProjectKmsGetCA(mock.Anything, pg.Spec.Project).Return("ca-cert", nil).Once()

			c := newTestController(avn, postgreSQLKind, nil)
			obs, err := c.Observe(t.Context(), pg)
			require.NoError(t, err)

			// Build the legacy expectation directly from the adapter.
			adapter := &postgreSQLAdapter{PostgreSQL: pg}
			expected := adapter.newSecret(svcOut).StringData
			expected["POSTGRESQL_CA_CERT"] = "ca-cert"

			require.Equal(t, expected, obs.SecretDetails)
			require.Equal(t, "ca-cert", obs.SecretDetails["POSTGRESQL_CA_CERT"])
		})

		t.Run("kafka prefixed and unprefixed CA_CERT", func(t *testing.T) {
			t.Parallel()

			kafka := &v1alpha1.Kafka{
				TypeMeta:   metav1.TypeMeta{Kind: "Kafka", APIVersion: "aiven.io/v1alpha1"},
				ObjectMeta: metav1.ObjectMeta{Name: "my-kafka", Namespace: "default", Generation: 1},
				Spec: v1alpha1.KafkaSpec{
					ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
						BaseServiceFields: v1alpha1.BaseServiceFields{
							ProjectDependant: v1alpha1.ProjectDependant{ProjectField: v1alpha1.ProjectField{Project: "test-project"}},
							Plan:             "business-4",
							Powered:          new(true),
						},
					},
				},
			}
			markGeneration(kafka, 1)

			accessCert := "access-cert"
			accessKey := "access-key"
			restURI := "https://rest.example.com"
			schemaURI := "https://schema.example.com"
			svcOut := &service.ServiceGetOut{
				State: service.ServiceStateTypeRunning,
				Users: []service.UserOut{{Username: "avnadmin", Password: "pw"}},
				ServiceUriParams: map[string]string{
					"host": "kafka.example.com",
					"port": "9092",
				},
				ConnectionInfo: &service.ConnectionInfoOut{
					KafkaAccessCert:   &accessCert,
					KafkaAccessKey:    &accessKey,
					KafkaRestUri:      &restURI,
					SchemaRegistryUri: &schemaURI,
				},
			}

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, kafka.Spec.Project, kafka.Name, mock.Anything).
				Return(inSync(kafka, svcOut), nil).Once()
			avn.EXPECT().
				ProjectKmsGetCA(mock.Anything, kafka.Spec.Project).Return("ca-cert", nil).Once()

			c := newTestController(avn, kafkaKind, nil)
			obs, err := c.Observe(t.Context(), kafka)
			require.NoError(t, err)

			// Both the prefixed key and the backward-compatible unprefixed key are present.
			require.Equal(t, "ca-cert", obs.SecretDetails["KAFKA_CA_CERT"])
			require.Equal(t, "ca-cert", obs.SecretDetails["CA_CERT"])

			adapter := &kafkaAdapter{Kafka: kafka}
			expected := adapter.newSecret(svcOut).StringData
			expected["KAFKA_CA_CERT"] = "ca-cert"
			expected["CA_CERT"] = "ca-cert"
			require.Equal(t, expected, obs.SecretDetails)
		})
	})
}

func newLifecycleScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	return scheme
}

func TestServiceController_Create(t *testing.T) {
	t.Parallel()

	t.Run("builds ServiceCreateIn from spec", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.MaintenanceWindowDow = "monday"
		gf.Spec.MaintenanceWindowTime = "10:00:00"
		gf.Spec.TechnicalEmails = []v1alpha1.ServiceTechEmail{{Email: "ops@example.com"}}
		gf.Spec.ServiceIntegrations = []*v1alpha1.ServiceIntegrationItem{
			{IntegrationType: service.IntegrationTypeReadReplica, SourceServiceName: "source-svc"},
		}
		gf.SetAnnotations(map[string]string{instanceIsRunningAnnotation: "true"})

		var createReq *service.ServiceCreateIn

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceCreate(mock.Anything, gf.Spec.Project, mock.Anything).
			Run(func(_ context.Context, _ string, in *service.ServiceCreateIn) { createReq = in }).
			Return(&service.ServiceCreateOut{}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Create(t.Context(), gf)
		require.NoError(t, err)

		require.NotNil(t, createReq)
		require.Equal(t, "startup-1", createReq.Plan)
		require.Equal(t, gf.Name, createReq.ServiceName)
		require.Equal(t, string(serviceTypeGrafana), createReq.ServiceType)
		require.NotNil(t, createReq.Cloud)
		require.Equal(t, "google-europe-west1", *createReq.Cloud)
		require.NotNil(t, createReq.Maintenance)
		require.Equal(t, service.DowType("monday"), createReq.Maintenance.Dow)
		require.NotNil(t, createReq.Maintenance.Time)
		require.Equal(t, "10:00:00", *createReq.Maintenance.Time)
		require.NotNil(t, createReq.TechEmails)
		require.Equal(t, []service.TechEmailIn{{Email: "ops@example.com"}}, *createReq.TechEmails)
		require.NotNil(t, createReq.ServiceIntegrations)
		require.Len(t, *createReq.ServiceIntegrations, 1)
		require.Equal(t, service.IntegrationTypeReadReplica, (*createReq.ServiceIntegrations)[0].IntegrationType)
		require.NotNil(t, (*createReq.ServiceIntegrations)[0].SourceService)
		require.Equal(t, "source-svc", *(*createReq.ServiceIntegrations)[0].SourceService)
		// No VPC id when neither ProjectVPCID nor ProjectVPCRef is set.
		require.Nil(t, createReq.ProjectVpcId)

		// Omitted tags are unmanaged: not on the request, no tags call.
		require.Nil(t, createReq.Tags)

		// CreatedOrUpdated conditions recorded; stale running marker cleared.
		_, hasRunningAnn := gf.GetAnnotations()[instanceIsRunningAnnotation]
		require.False(t, hasRunningAnn)
		initialized := meta.FindStatusCondition(gf.Status.Conditions, conditionTypeInitialized)
		require.NotNil(t, initialized)
		require.Equal(t, "CreatedOrUpdated", initialized.Reason)
		running := meta.FindStatusCondition(gf.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, running)
		require.Equal(t, metav1.ConditionUnknown, running.Status)
		require.Equal(t, "CreatedOrUpdated", running.Reason)
	})

	t.Run("resolves VPC id from ProjectVPCRef", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.CloudName = ""
		gf.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "test-vpc"}

		vpc := &v1alpha1.ProjectVPC{
			ObjectMeta: metav1.ObjectMeta{Name: "test-vpc", Namespace: "default"},
			Status:     v1alpha1.ProjectVPCStatus{ID: "vpc-123"},
		}
		k8s := fake.NewClientBuilder().WithScheme(newLifecycleScheme(t)).WithObjects(vpc).Build()

		var createReq *service.ServiceCreateIn
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceCreate(mock.Anything, gf.Spec.Project, mock.Anything).
			Run(func(_ context.Context, _ string, in *service.ServiceCreateIn) { createReq = in }).
			Return(&service.ServiceCreateOut{}, nil).Once()

		c := newTestController(avn, grafanaKind, k8s)
		_, err := c.Create(t.Context(), gf)
		require.NoError(t, err)

		require.NotNil(t, createReq)
		require.NotNil(t, createReq.ProjectVpcId)
		require.Equal(t, "vpc-123", *createReq.ProjectVpcId)
	})

	t.Run("ServiceCreate error propagates", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceCreate(mock.Anything, gf.Spec.Project, mock.Anything).
			Return(nil, newAivenError(500, "boom")).Once()

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Create(t.Context(), gf)
		require.ErrorContains(t, err, "failed to create service")
	})

	t.Run("ServiceCreate already-exists soft-requeues without claiming the generation", func(t *testing.T) {
		t.Parallel()

		// Applied service, pending spec change, transient 404 in Observe. A 409 treated as success
		// would stamp generation 2 without sending ServiceUpdate.
		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		markGeneration(gf, 1)
		gf.Generation = 2
		before := gf.DeepCopy()

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceCreate(mock.Anything, gf.Spec.Project, mock.Anything).
			Return(nil, newAivenError(409, "Service name is already in use in this project")).Once()

		c, rec := newRecordingTestController(avn, grafanaKind, nil)
		_, err := c.Create(t.Context(), gf)
		require.ErrorIs(t, err, errPreconditionNotMet)
		require.ErrorContains(t, err, "service already exists at Aiven")

		// No tags/hook calls, no events, no state change.
		require.Empty(t, recorderEvents(rec))
		require.Equal(t, before.Annotations, gf.Annotations)
		require.Equal(t, before.Status, gf.Status)
	})

	t.Run("ServiceCreate 409 other than a taken name is a hard error", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceCreate(mock.Anything, gf.Spec.Project, mock.Anything).
			Return(nil, newAivenError(409, "Project has reached its service limit")).Once()

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Create(t.Context(), gf)
		require.ErrorContains(t, err, "failed to create service")
		require.NotErrorIs(t, err, errPreconditionNotMet)
	})

	t.Run("recreate Warning only when an applied service was actually created", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name      string
			applied   bool
			wantEvent bool
		}{
			{name: "first create is silent", applied: false, wantEvent: false},
			{name: "applied service recreated emits InstanceMissingAtAiven", applied: true, wantEvent: true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
				if tc.applied {
					markGeneration(gf, 1)
				}

				avn := avngen.NewMockClient(t)
				avn.EXPECT().
					ServiceCreate(mock.Anything, gf.Spec.Project, mock.Anything).
					Return(&service.ServiceCreateOut{}, nil).Once()

				c, rec := newRecordingTestController(avn, grafanaKind, nil)
				_, err := c.Create(t.Context(), gf)
				require.NoError(t, err)

				events := recorderEvents(rec)
				if !tc.wantEvent {
					require.Empty(t, events)
					return
				}
				require.Len(t, events, 1)
				require.Contains(t, events[0], "Warning "+eventInstanceMissingAtAiven)
			})
		}
	})

	t.Run("recreate un-stamps the generation so the next Observe routes to Update", func(t *testing.T) {
		t.Parallel()

		// Applied service deleted at Aiven; the generation annotation still matches.
		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.Powered = new(true)
		markGeneration(gf, 1)
		markInstanceRunning(gf)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()
		avn.EXPECT().
			ServiceCreate(mock.Anything, gf.Spec.Project, mock.Anything).
			Return(&service.ServiceCreateOut{}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		obs, err := c.Observe(t.Context(), gf)
		require.NoError(t, err)
		require.False(t, obs.ResourceExists)

		_, err = c.Create(t.Context(), gf)
		require.NoError(t, err)
		require.False(t, hasLatestGeneration(gf))

		// The service is back, still rebuilding: Observe must route to Update, not poll.
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRebuilding}, nil).Once()
		obs, err = c.Observe(t.Context(), gf)
		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.False(t, obs.ResourceUpToDate)
	})

	t.Run(`projectVpcId: "" creates outside a VPC`, func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.ProjectVPCID = new("")

		var createReq *service.ServiceCreateIn
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceCreate(mock.Anything, gf.Spec.Project, mock.Anything).
			Run(func(_ context.Context, _ string, in *service.ServiceCreateIn) { createReq = in }).
			Return(&service.ServiceCreateOut{}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Create(t.Context(), gf)
		require.NoError(t, err)
		require.Nil(t, createReq.ProjectVpcId)
	})

	t.Run("projectVpcId with projectVPCRef is a hard error before ServiceCreate", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.ProjectVPCID = new("")
		gf.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "vpc"}

		// No k8s or Aiven calls: the spec is rejected first.
		c := newTestController(avngen.NewMockClient(t), grafanaKind, nil)
		_, err := c.Create(t.Context(), gf)
		require.ErrorContains(t, err, "not both")
		require.NotErrorIs(t, err, errPreconditionNotMet)
	})

	t.Run("missing ProjectVPCRef errors before ServiceCreate", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.CloudName = ""
		gf.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "missing-vpc"}

		// Empty cluster: the referenced ProjectVPC does not exist.
		k8s := fake.NewClientBuilder().WithScheme(newLifecycleScheme(t)).Build()

		// No Aiven calls expected: VPC resolution fails first.
		avn := avngen.NewMockClient(t)

		c := newTestController(avn, grafanaKind, k8s)
		_, err := c.Create(t.Context(), gf)
		require.ErrorContains(t, err, "failed to get referenced ProjectVPC")
	})

	t.Run("tags and technical emails: omitted not sent, empty sent empty", func(t *testing.T) {
		t.Parallel()

		for name, tc := range map[string]struct {
			tags   map[string]string
			emails []v1alpha1.ServiceTechEmail
		}{
			"omitted": {},
			"empty":   {tags: map[string]string{}, emails: []v1alpha1.ServiceTechEmail{}},
			"set":     {tags: map[string]string{"env": "prod"}, emails: []v1alpha1.ServiceTechEmail{{Email: "ops@example.com"}}},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
				gf.Spec.Tags = tc.tags
				gf.Spec.TechnicalEmails = tc.emails

				var createReq *service.ServiceCreateIn
				avn := avngen.NewMockClient(t)
				avn.EXPECT().
					ServiceCreate(mock.Anything, gf.Spec.Project, mock.Anything).
					Run(func(_ context.Context, _ string, in *service.ServiceCreateIn) { createReq = in }).
					Return(&service.ServiceCreateOut{}, nil).Once()

				c := newTestController(avn, grafanaKind, nil)
				_, err := c.Create(t.Context(), gf)
				require.NoError(t, err)

				if tc.tags == nil {
					require.Nil(t, createReq.Tags)
					require.Nil(t, createReq.TechEmails)
					return
				}
				require.Equal(t, &tc.tags, createReq.Tags)
				require.Len(t, *createReq.TechEmails, len(tc.emails))
			})
		}
	})

	t.Run("resolves migration user config on create", func(t *testing.T) {
		t.Parallel()

		pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
		pg.Namespace = "default"
		pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: "creds"}
		pg.Spec.UserConfig = &pguserconfig.PgUserConfig{}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "default"},
			Data: map[string][]byte{
				"host": []byte("source.example.com"),
				"port": []byte("5432"),
			},
		}
		k8s := fake.NewClientBuilder().WithScheme(newLifecycleScheme(t)).WithObjects(secret).Build()

		var createReq *service.ServiceCreateIn
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceCreate(mock.Anything, pg.Spec.Project, mock.Anything).
			Run(func(_ context.Context, _ string, in *service.ServiceCreateIn) { createReq = in }).
			Return(&service.ServiceCreateOut{}, nil).Once()

		c := newTestController(avn, postgreSQLKind, k8s)
		_, err := c.Create(t.Context(), pg)
		require.NoError(t, err)

		// The migration block from the secret lands in ServiceCreateIn.UserConfig with host/port.
		require.NotNil(t, createReq)
		require.NotNil(t, createReq.UserConfig)
		migration, ok := (*createReq.UserConfig)["migration"]
		require.True(t, ok, "expected migration block in submitted user config")
		mig, ok := migration.(map[string]any)
		require.True(t, ok, "migration block should be a map")
		require.Equal(t, "source.example.com", mig["host"])
		require.EqualValues(t, 5432, mig["port"])

		// The sent config is recorded, so a later status 404 reads as done.
		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		require.NotNil(t, cond)
		require.Equal(t, v1alpha1.MigrationReasonInProgress, cond.Reason)
	})

	t.Run("recreate resets MigrationComplete so the migration block is sent again", func(t *testing.T) {
		t.Parallel()

		// Applied PostgreSQL whose migration finished, then deleted at Aiven.
		pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
		pg.Namespace = "default"
		pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: "creds"}
		pg.Spec.UserConfig = &pguserconfig.PgUserConfig{}
		pg.Spec.Powered = new(true)
		markGeneration(pg, 1)
		markInstanceRunning(pg)
		meta.SetStatusCondition(&pg.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.ConditionTypeMigrationComplete,
			Status: metav1.ConditionTrue,
			Reason: v1alpha1.MigrationReasonDone,
		})

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "default"},
			Data: map[string][]byte{
				"host": []byte("source.example.com"),
				"port": []byte("5432"),
			},
		}
		k8s := fake.NewClientBuilder().WithScheme(newLifecycleScheme(t)).WithObjects(secret).Build()

		var createReq *service.ServiceCreateIn
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()
		avn.EXPECT().
			ServiceCreate(mock.Anything, pg.Spec.Project, mock.Anything).
			Run(func(_ context.Context, _ string, in *service.ServiceCreateIn) { createReq = in }).
			Return(&service.ServiceCreateOut{}, nil).Once()

		c := newTestController(avn, postgreSQLKind, k8s)
		obs, err := c.Observe(t.Context(), pg)
		require.NoError(t, err)
		require.False(t, obs.ResourceExists)
		require.Nil(t, meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete))

		_, err = c.Create(t.Context(), pg)
		require.NoError(t, err)

		// Without the reset, resolveUserConfig would skip the Secret and send no migration block.
		require.NotNil(t, createReq)
		require.NotNil(t, createReq.UserConfig)
		mig, ok := (*createReq.UserConfig)["migration"].(map[string]any)
		require.True(t, ok, "expected migration block in submitted user config")
		require.Equal(t, "source.example.com", mig["host"])
	})

	t.Run("recreate with deleteAfterMigration", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name          string
			secretExists  bool
			wantMigration bool
		}{
			{name: "Secret deleted: recreates without the migration and keeps MigrationComplete", secretExists: false, wantMigration: false},
			{name: "Secret still present: resets MigrationComplete and sends the migration again", secretExists: true, wantMigration: true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Applied PostgreSQL whose migration finished, then deleted at Aiven.
				pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
				pg.Namespace = "default"
				pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: "creds", DeleteAfterMigration: true}
				pg.Spec.UserConfig = &pguserconfig.PgUserConfig{}
				pg.Spec.Powered = new(true)
				markGeneration(pg, 1)
				markInstanceRunning(pg)
				meta.SetStatusCondition(&pg.Status.Conditions, metav1.Condition{
					Type:   v1alpha1.ConditionTypeMigrationComplete,
					Status: metav1.ConditionTrue,
					Reason: v1alpha1.MigrationReasonDone,
				})

				builder := fake.NewClientBuilder().WithScheme(newLifecycleScheme(t))
				if tc.secretExists {
					builder = builder.WithObjects(&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "default"},
						Data: map[string][]byte{
							"host": []byte("source.example.com"),
							"port": []byte("5432"),
						},
					})
				}

				var createReq *service.ServiceCreateIn
				avn := avngen.NewMockClient(t)
				avn.EXPECT().
					ServiceGet(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
					Return(nil, newAivenError(404, "service not found")).Once()
				avn.EXPECT().
					ServiceCreate(mock.Anything, pg.Spec.Project, mock.Anything).
					Run(func(_ context.Context, _ string, in *service.ServiceCreateIn) { createReq = in }).
					Return(&service.ServiceCreateOut{}, nil).Once()

				c, rec := newRecordingTestController(avn, postgreSQLKind, builder.Build())
				obs, err := c.Observe(t.Context(), pg)
				require.NoError(t, err)
				require.False(t, obs.ResourceExists)
				require.Equal(t, !tc.wantMigration, meta.IsStatusConditionTrue(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete))

				_, err = c.Create(t.Context(), pg)
				require.NoError(t, err)

				require.NotNil(t, createReq)
				var userCfg map[string]any
				if createReq.UserConfig != nil {
					userCfg = *createReq.UserConfig
				}
				_, hasMigration := userCfg["migration"]
				require.Equal(t, tc.wantMigration, hasMigration)

				require.Len(t, rec.Events, 1)
				event := <-rec.Events
				require.Contains(t, event, eventInstanceMissingAtAiven)
				if tc.wantMigration {
					require.NotContains(t, event, "without the migration")
				} else {
					require.Contains(t, event, "without the migration")
				}
			})
		}
	})
}

func TestServiceController_Update(t *testing.T) {
	t.Parallel()

	t.Run("builds ServiceUpdateIn including Powered", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.Powered = new(false)
		gf.Spec.Tags = map[string]string{"team": "core"}
		gf.SetAnnotations(map[string]string{instanceIsRunningAnnotation: "true"})

		var updateReq *service.ServiceUpdateIn
		var tagsReq *service.ProjectServiceTagsReplaceIn

		avn := avngen.NewMockClient(t)
		// Powering off requires a backup: the precondition runs before ServiceUpdate.
		avn.EXPECT().
			ServiceBackupsGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(&service.ServiceBackupsGetOut{Backups: []service.BackupOut{{BackupName: "b1"}}}, nil).Once()
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()
		avn.EXPECT().
			ServiceUpdate(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Run(func(_ context.Context, _, _ string, in *service.ServiceUpdateIn, _ ...[2]string) { updateReq = in }).
			Return(&service.ServiceUpdateOut{}, nil).Once()
		avn.EXPECT().
			ProjectServiceTagsReplace(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Run(func(_ context.Context, _, _ string, in *service.ProjectServiceTagsReplaceIn) { tagsReq = in }).
			Return(nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Update(t.Context(), gf)
		require.NoError(t, err)

		require.NotNil(t, updateReq)
		require.NotNil(t, updateReq.Powered)
		require.False(t, *updateReq.Powered)
		require.NotNil(t, updateReq.Plan)
		require.Equal(t, "startup-1", *updateReq.Plan)
		// Omitted fields are not sent; the service is not in a VPC, so null is a no-op.
		require.Nil(t, updateReq.TechEmails)
		require.Nil(t, updateReq.Maintenance)
		require.Nil(t, updateReq.ProjectVpcId)

		require.NotNil(t, tagsReq)
		require.Equal(t, map[string]string{"team": "core"}, tagsReq.Tags)

		// Stale running marker cleared, CreatedOrUpdated conditions recorded.
		_, hasRunningAnn := gf.GetAnnotations()[instanceIsRunningAnnotation]
		require.False(t, hasRunningAnn)
		running := meta.FindStatusCondition(gf.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, running)
		require.Equal(t, metav1.ConditionUnknown, running.Status)
	})

	t.Run("drift correction keeps readiness", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		markGeneration(gf, 1)
		markInstanceRunning(gf)
		require.True(t, IsReadyToUse(gf))

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()
		avn.EXPECT().
			ServiceUpdate(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Return(&service.ServiceUpdateOut{}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Update(t.Context(), gf)
		require.NoError(t, err)

		require.True(t, hasLatestGeneration(gf))
		require.True(t, IsReadyToUse(gf))
		running := meta.FindStatusCondition(gf.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, running)
		require.Equal(t, metav1.ConditionTrue, running.Status)
	})

	t.Run("power drift correction resets readiness", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		markGeneration(gf, 1)
		markInstanceRunning(gf)
		require.True(t, IsReadyToUse(gf))

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypePoweroff}, nil).Once()
		avn.EXPECT().
			ServiceUpdate(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Return(&service.ServiceUpdateOut{}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Update(t.Context(), gf)
		require.NoError(t, err)

		require.True(t, hasLatestGeneration(gf))
		require.False(t, IsReadyToUse(gf))
		running := meta.FindStatusCondition(gf.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, running)
		require.Equal(t, metav1.ConditionUnknown, running.Status)
	})

	t.Run("disk space with autoscaler integration is not sent -> Warning on a spec change only", func(t *testing.T) {
		t.Parallel()

		for name, tc := range map[string]struct {
			processed bool
			events    []string
		}{
			"spec change":      {events: []string{"Warning DiskSpaceNotApplied diskSpace is not applied: the autoscaler integration manages the disk"}},
			"drift correction": {processed: true},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
				gf.Spec.DiskSpace = "10GiB"
				if tc.processed {
					markGeneration(gf, 1)
				}

				avn := avngen.NewMockClient(t)
				avn.EXPECT().
					ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
					Return(&service.ServiceGetOut{
						State:               service.ServiceStateTypeRunning,
						ServiceIntegrations: []service.ServiceIntegrationOut{{IntegrationType: service.IntegrationTypeAutoscaler}},
					}, nil).Once()
				var updateReq *service.ServiceUpdateIn
				avn.EXPECT().
					ServiceUpdate(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
					Run(func(_ context.Context, _, _ string, in *service.ServiceUpdateIn, _ ...[2]string) { updateReq = in }).
					Return(&service.ServiceUpdateOut{}, nil).Once()

				c, rec := newRecordingTestController(avn, grafanaKind, nil)
				_, err := c.Update(t.Context(), gf)
				require.NoError(t, err)
				require.Nil(t, updateReq.DiskSpaceMb)
				require.Equal(t, tc.events, recorderEvents(rec))
			})
		}
	})

	t.Run("upgrade task runs before ServiceUpdate and blocks on error", func(t *testing.T) {
		t.Parallel()

		pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
		pg.Namespace = "default"
		targetVersion := "16"
		pg.Spec.UserConfig = &pguserconfig.PgUserConfig{PgVersion: &targetVersion}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetOut{
				State:      service.ServiceStateTypeRunning,
				UserConfig: map[string]any{"pg_version": "15"},
			}, nil).Once()
		// Upgrade check task fails -> Update returns error and never calls ServiceUpdate.
		avn.EXPECT().
			ServiceTaskCreate(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
			Return(nil, newAivenError(400, "upgrade not possible")).Once()

		c := newTestController(avn, postgreSQLKind, nil)
		_, err := c.Update(t.Context(), pg)
		require.ErrorContains(t, err, "upgrade check task")
	})

	t.Run("ServiceGet failure propagates before update", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(nil, newAivenError(500, "boom")).Once()
		// ServiceUpdate must NOT be called.

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Update(t.Context(), gf)
		require.ErrorContains(t, err, "failed to fetch service")
	})

	t.Run("ProjectServiceTagsReplace error propagates after update", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.Tags = map[string]string{} // explicit {} clears tags, so the endpoint is called

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()
		avn.EXPECT().
			ServiceUpdate(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Return(&service.ServiceUpdateOut{}, nil).Once()
		avn.EXPECT().
			ProjectServiceTagsReplace(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Return(newAivenError(500, "tags boom")).Once()

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Update(t.Context(), gf)
		require.ErrorContains(t, err, "failed to update tags")
	})

	t.Run("omitted tags and emails are not sent, an unmanaged VPC is kept", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)

		var updateReq *service.ServiceUpdateIn
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning, ProjectVpcId: "vpc-console"}, nil).Once()
		avn.EXPECT().
			ServiceUpdate(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Run(func(_ context.Context, _, _ string, in *service.ServiceUpdateIn, _ ...[2]string) { updateReq = in }).
			Return(&service.ServiceUpdateOut{}, nil).Once()
		// No ProjectServiceTagsReplace: the mock fails on an unexpected call.

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Update(t.Context(), gf)
		require.NoError(t, err)

		require.Nil(t, updateReq.TechEmails)
		// A null project_vpc_id would move the service out of its VPC.
		require.Equal(t, new("vpc-console"), updateReq.ProjectVpcId)
	})

	t.Run("projectVpcId: omitted echoes the live VPC, empty sends null, an ID is sent", func(t *testing.T) {
		t.Parallel()

		for name, tc := range map[string]struct {
			spec *string
			want *string
		}{
			"omitted": {spec: nil, want: new("vpc-console")},
			"empty":   {spec: new(""), want: nil},
			"id":      {spec: new("vpc-new"), want: new("vpc-new")},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
				gf.Spec.ProjectVPCID = tc.spec

				var updateReq *service.ServiceUpdateIn
				avn := avngen.NewMockClient(t)
				avn.EXPECT().
					ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
					Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning, ProjectVpcId: "vpc-console"}, nil).Once()
				avn.EXPECT().
					ServiceUpdate(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
					Run(func(_ context.Context, _, _ string, in *service.ServiceUpdateIn, _ ...[2]string) { updateReq = in }).
					Return(&service.ServiceUpdateOut{}, nil).Once()

				c := newTestController(avn, grafanaKind, nil)
				_, err := c.Update(t.Context(), gf)
				require.NoError(t, err)
				require.Equal(t, tc.want, updateReq.ProjectVpcId)
			})
		}
	})

	t.Run("projectVpcId with projectVPCRef is a hard error before ServiceGet", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.ProjectVPCID = new("vpc-1")
		gf.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "vpc"}

		c := newTestController(avngen.NewMockClient(t), grafanaKind, nil)
		_, err := c.Update(t.Context(), gf)
		require.ErrorContains(t, err, "not both")
		require.NotErrorIs(t, err, errPreconditionNotMet)
	})

	t.Run("ProjectVPCRef without an ID is a precondition, not an unmanaged VPC", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "new-vpc"}
		vpc := &v1alpha1.ProjectVPC{ObjectMeta: metav1.ObjectMeta{Name: "new-vpc", Namespace: "default"}}
		k8s := fake.NewClientBuilder().WithScheme(newLifecycleScheme(t)).WithObjects(vpc).Build()

		// No ServiceGet or ServiceUpdate: the live VPC must not be echoed in place of the requested one.
		c := newTestController(avngen.NewMockClient(t), grafanaKind, k8s)
		_, err := c.Update(t.Context(), gf)
		require.ErrorIs(t, err, errPreconditionNotMet)
	})

	t.Run("maintenance dow only is sent without a time key", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.MaintenanceWindowDow = service.DowTypeMonday

		var updateReq *service.ServiceUpdateIn
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()
		avn.EXPECT().
			ServiceUpdate(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Run(func(_ context.Context, _, _ string, in *service.ServiceUpdateIn, _ ...[2]string) { updateReq = in }).
			Return(&service.ServiceUpdateOut{}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Update(t.Context(), gf)
		require.NoError(t, err)
		// time: "" is a 400 at Aiven; powered defaults to true.
		require.Equal(t, &service.MaintenanceIn{Dow: service.DowTypeMonday}, updateReq.Maintenance)
		require.Equal(t, new(true), updateReq.Powered)
	})

	t.Run("empty technicalEmails clears them", func(t *testing.T) {
		t.Parallel()

		gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
		gf.Spec.TechnicalEmails = []v1alpha1.ServiceTechEmail{}

		var updateReq *service.ServiceUpdateIn
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, gf.Spec.Project, gf.Name).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()
		avn.EXPECT().
			ServiceUpdate(mock.Anything, gf.Spec.Project, gf.Name, mock.Anything).
			Run(func(_ context.Context, _, _ string, in *service.ServiceUpdateIn, _ ...[2]string) { updateReq = in }).
			Return(&service.ServiceUpdateOut{}, nil).Once()

		c := newTestController(avn, grafanaKind, nil)
		_, err := c.Update(t.Context(), gf)
		require.NoError(t, err)
		require.Equal(t, &[]service.TechEmailIn{}, updateReq.TechEmails)
	})

	t.Run("migration user config", func(t *testing.T) {
		t.Parallel()

		newMigratingPG := func(t *testing.T) *v1alpha1.PostgreSQL {
			pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
			pg.Namespace = "default"
			pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: "creds"}
			pg.Spec.UserConfig = &pguserconfig.PgUserConfig{}
			return pg
		}

		t.Run("resolves migration user config while not complete", func(t *testing.T) {
			t.Parallel()

			pg := newMigratingPG(t)
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "default"},
				Data: map[string][]byte{
					"host": []byte("source.example.com"),
					"port": []byte("5432"),
				},
			}
			k8s := fake.NewClientBuilder().WithScheme(newLifecycleScheme(t)).WithObjects(secret).Build()

			var updateReq *service.ServiceUpdateIn
			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
				Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning, UserConfig: map[string]any{"pg_version": "16"}}, nil).Once()
			avn.EXPECT().
				ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
				Run(func(_ context.Context, _, _ string, in *service.ServiceUpdateIn, _ ...[2]string) { updateReq = in }).
				Return(&service.ServiceUpdateOut{}, nil).Once()

			c := newTestController(avn, postgreSQLKind, k8s)
			_, err := c.Update(t.Context(), pg)
			require.NoError(t, err)

			// The migration block from the secret is merged into the submitted user config,
			// with host/port resolved from the Secret data.
			require.NotNil(t, updateReq)
			require.NotNil(t, updateReq.UserConfig)
			migration, ok := (*updateReq.UserConfig)["migration"]
			require.True(t, ok, "expected migration block in submitted user config")
			mig, ok := migration.(map[string]any)
			require.True(t, ok, "migration block should be a map")
			require.Equal(t, "source.example.com", mig["host"])
			require.EqualValues(t, 5432, mig["port"])

			// The sent config is recorded, so a later status 404 reads as done.
			cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
			require.NotNil(t, cond)
			require.Equal(t, metav1.ConditionFalse, cond.Status)
			require.Equal(t, v1alpha1.MigrationReasonInProgress, cond.Reason)
		})

		t.Run("skips migration secret once complete", func(t *testing.T) {
			t.Parallel()

			pg := newMigratingPG(t)
			// Migration complete and the secret absent: a successful Update proves it was not read.
			meta.SetStatusCondition(&pg.Status.Conditions, metav1.Condition{
				Type:   v1alpha1.ConditionTypeMigrationComplete,
				Status: metav1.ConditionTrue,
				Reason: v1alpha1.MigrationReasonDone,
			})
			k8s := fake.NewClientBuilder().WithScheme(newLifecycleScheme(t)).Build()

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
				Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning, UserConfig: map[string]any{"pg_version": "16"}}, nil).Once()
			avn.EXPECT().
				ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
				Return(&service.ServiceUpdateOut{}, nil).Once()

			c := newTestController(avn, postgreSQLKind, k8s)
			_, err := c.Update(t.Context(), pg)
			require.NoError(t, err)
		})
	})

	t.Run("preconditions gate ServiceUpdate", func(t *testing.T) {
		t.Parallel()

		t.Run("power off without backups -> errPreconditionExternal", func(t *testing.T) {
			t.Parallel()

			gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
			gf.Spec.Powered = new(false)

			avn := avngen.NewMockClient(t)
			// No ServiceGet, no ServiceUpdate: the check runs first.
			avn.EXPECT().
				ServiceBackupsGet(mock.Anything, gf.Spec.Project, gf.Name).
				Return(&service.ServiceBackupsGetOut{Backups: []service.BackupOut{}}, nil).Once()

			c := newTestController(avn, grafanaKind, nil)
			_, err := c.Update(t.Context(), gf)
			// Only Aiven taking a backup unblocks it: the reconciler polls at PollInterval.
			require.ErrorIs(t, err, errPreconditionExternal)
		})

		t.Run("drift correction without backups -> errPreconditionExternal, readiness kept", func(t *testing.T) {
			t.Parallel()

			gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
			gf.Spec.Powered = new(false)
			markGeneration(gf, 1)
			markInstanceRunning(gf)

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceBackupsGet(mock.Anything, gf.Spec.Project, gf.Name).
				Return(&service.ServiceBackupsGetOut{Backups: []service.BackupOut{}}, nil).Once()

			c := newTestController(avn, grafanaKind, nil)
			_, err := c.Update(t.Context(), gf)
			require.ErrorIs(t, err, errPreconditionExternal)
			require.True(t, IsReadyToUse(gf))
		})

		t.Run("backup check API error propagates as a plain error", func(t *testing.T) {
			t.Parallel()

			gf := newObjectFromYAML[v1alpha1.Grafana](t, yamlGrafanaSvc)
			gf.Spec.Powered = new(false)

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceBackupsGet(mock.Anything, gf.Spec.Project, gf.Name).
				Return(nil, newAivenError(500, "backups unavailable")).Once()

			c := newTestController(avn, grafanaKind, nil)
			_, err := c.Update(t.Context(), gf)
			require.Error(t, err)
			require.NotErrorIs(t, err, errPreconditionNotMet)
		})
	})
}

// TestServiceController_Delete: parity with the legacy genericServiceHandler.delete.
func TestServiceController_Delete(t *testing.T) {
	t.Parallel()

	newPG := func(t *testing.T, tp *bool) *v1alpha1.PostgreSQL {
		pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
		pg.Spec.TerminationProtection = tp
		return pg
	}

	t.Run("termination protection on -> hard error, no API calls", func(t *testing.T) {
		t.Parallel()

		pg := newPG(t, new(true))
		// No API calls: the guard blocks first.
		avn := avngen.NewMockClient(t)

		c := newTestController(avn, postgreSQLKind, nil)
		err := c.Delete(t.Context(), pg)
		require.ErrorIs(t, err, errTerminationProtectionOn)
	})

	t.Run("live protection on: switched off with the VPC echoed, then ServiceDelete", func(t *testing.T) {
		t.Parallel()

		pg := newPG(t, new(false))

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetOut{TerminationProtection: true, ProjectVpcId: "vpc-1"}, nil).Once()
		// A null project_vpc_id would move the service out of its VPC right before the delete.
		updateCall := avn.EXPECT().
			ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, &service.ServiceUpdateIn{
				TerminationProtection: new(false),
				ProjectVpcId:          new("vpc-1"),
			}).
			Return(nil, nil).Once()
		avn.EXPECT().
			ServiceDelete(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil).Once().
			NotBefore(updateCall)

		c := newTestController(avn, postgreSQLKind, nil)
		err := c.Delete(t.Context(), pg)
		require.NoError(t, err)
	})

	t.Run("live protection off: no ServiceUpdate, nil spec protection deletes", func(t *testing.T) {
		t.Parallel()

		pg := newPG(t, nil)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetOut{}, nil).Once()
		// No ServiceUpdate: the mock fails on an unexpected call.
		avn.EXPECT().
			ServiceDelete(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil).Once()

		c := newTestController(avn, postgreSQLKind, nil)
		err := c.Delete(t.Context(), pg)
		require.NoError(t, err)
	})

	t.Run("service already gone at ServiceGet -> nil, no further calls", func(t *testing.T) {
		t.Parallel()

		pg := newPG(t, new(false))

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil, newAivenError(404, "Service not found")).Once()

		c := newTestController(avn, postgreSQLKind, nil)
		err := c.Delete(t.Context(), pg)
		require.NoError(t, err)
	})

	t.Run("ServiceGet non-404 error propagates, no ServiceDelete", func(t *testing.T) {
		t.Parallel()

		pg := newPG(t, new(false))

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil, newAivenError(500, "Internal error")).Once()

		c := newTestController(avn, postgreSQLKind, nil)
		err := c.Delete(t.Context(), pg)
		require.ErrorContains(t, err, "failed to fetch service")
	})

	t.Run("TP sync tolerates 404 and proceeds", func(t *testing.T) {
		t.Parallel()

		pg := newPG(t, new(false))

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetOut{TerminationProtection: true}, nil).Once()
		avn.EXPECT().
			ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
			Return(nil, newAivenError(404, "Service not found")).Once()
		avn.EXPECT().
			ServiceDelete(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil).Once()

		c := newTestController(avn, postgreSQLKind, nil)
		err := c.Delete(t.Context(), pg)
		require.NoError(t, err)
	})

	t.Run("TP sync non-404 error propagates, no ServiceDelete", func(t *testing.T) {
		t.Parallel()

		pg := newPG(t, new(false))

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetOut{TerminationProtection: true}, nil).Once()
		avn.EXPECT().
			ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
			Return(nil, newAivenError(403, "Forbidden")).Once()
		// ServiceDelete must NOT be called.

		c := newTestController(avn, postgreSQLKind, nil)
		err := c.Delete(t.Context(), pg)
		require.ErrorContains(t, err, "failed to disable termination protection")
	})

	t.Run("ServiceDelete 404 -> nil (already gone)", func(t *testing.T) {
		t.Parallel()

		pg := newPG(t, new(false))

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetOut{}, nil).Once()
		avn.EXPECT().
			ServiceDelete(mock.Anything, pg.Spec.Project, pg.Name).
			Return(newAivenError(404, "Service not found")).Once()

		c := newTestController(avn, postgreSQLKind, nil)
		err := c.Delete(t.Context(), pg)
		require.NoError(t, err)
	})

	t.Run("ServiceDelete non-404 error propagates", func(t *testing.T) {
		t.Parallel()

		pg := newPG(t, new(false))

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetOut{}, nil).Once()
		avn.EXPECT().
			ServiceDelete(mock.Anything, pg.Spec.Project, pg.Name).
			Return(newAivenError(500, "Internal error")).Once()

		c := newTestController(avn, postgreSQLKind, nil)
		err := c.Delete(t.Context(), pg)
		require.ErrorContains(t, err, "failed to delete service in Aiven")
	})
}
