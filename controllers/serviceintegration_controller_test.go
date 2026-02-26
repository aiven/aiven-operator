package controllers

import (
	"errors"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestServiceIntegrationController_integrationMatches(t *testing.T) {
	r := &ServiceIntegrationController{}

	tests := []struct {
		name      string
		existing  *service.ServiceIntegrationOut
		desired   *v1alpha1.ServiceIntegration
		wantMatch bool
	}{
		{
			name: "exact match",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "kafka_logs",
				SourceService:   "kafka-src",
				SourceProject:   "project-a",
				DestService:     ptr("kafka-dest"),
				DestProject:     "project-b",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:        "kafka_logs",
					SourceServiceName:      "kafka-src",
					SourceProjectName:      "project-a",
					DestinationServiceName: "kafka-dest",
					DestinationProjectName: "project-b",
				},
			},
			wantMatch: true,
		},
		{
			name: "match with endpoints",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "datadog",
				SourceService:   "pg-src",
				SourceProject:   "project-a",
				DestEndpointId:  ptr("endpoint-123"),
				DestProject:     "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:       "datadog",
					SourceServiceName:     "pg-src",
					DestinationEndpointID: "endpoint-123",
				},
			},
			wantMatch: true,
		},
		{
			name: "no match - different integration type",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "kafka_logs",
				SourceService:   "kafka-src",
				SourceProject:   "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:   "metrics",
					SourceServiceName: "kafka-src",
				},
			},
			wantMatch: false,
		},
		{
			name: "no match - different source service",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "kafka_logs",
				SourceService:   "kafka-src-1",
				SourceProject:   "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:   "kafka_logs",
					SourceServiceName: "kafka-src-2",
				},
			},
			wantMatch: false,
		},
		{
			name: "no match - different source project",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "kafka_logs",
				SourceService:   "kafka-src",
				SourceProject:   "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:   "kafka_logs",
					SourceServiceName: "kafka-src",
					SourceProjectName: "project-b",
				},
			},
			wantMatch: false,
		},
		{
			name: "no match - different destination service",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "kafka_logs",
				SourceService:   "kafka-src",
				SourceProject:   "project-a",
				DestService:     ptr("kafka-dest-1"),
				DestProject:     "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:        "kafka_logs",
					SourceServiceName:      "kafka-src",
					DestinationServiceName: "kafka-dest-2",
				},
			},
			wantMatch: false,
		},
		{
			name: "no match - missing endpoint in existing",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "datadog",
				SourceService:   "pg-src",
				SourceProject:   "project-a",
				DestProject:     "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:       "datadog",
					SourceServiceName:     "pg-src",
					DestinationEndpointID: "endpoint-123",
				},
			},
			wantMatch: false,
		},
		{
			name: "match - both have nil destination service",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "autoscaler",
				SourceService:   "pg-src",
				SourceProject:   "project-a",
				DestService:     nil,
				DestProject:     "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:        "autoscaler",
					SourceServiceName:      "pg-src",
					DestinationServiceName: "",
				},
			},
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.integrationMatches(tt.existing, tt.desired)
			assert.Equal(t, tt.wantMatch, got)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

const yamlServiceIntegrationAutoscalerLegacy = `
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: test-si
  namespace: default
spec:
  project: test-project
  integrationType: autoscaler
  sourceServiceName: test-pg
  destinationEndpointId: endpoint-123
`

func TestServiceIntegrationReconciler(t *testing.T) {
	t.Parallel()

	runScenario := func(t *testing.T, si *v1alpha1.ServiceIntegration, avn avngen.Client, additionalObjects ...client.Object) (*Reconciler[*v1alpha1.ServiceIntegration], ctrlruntime.Result) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		objects := append([]client.Object{si}, additionalObjects...)

		recorder := record.NewFakeRecorder(10)
		r := newServiceIntegrationReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.ServiceIntegration{}).
				WithObjects(objects...).
				Build(),
			Scheme:       scheme,
			Recorder:     recorder,
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.ServiceIntegration])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{
				Name:      si.Name,
				Namespace: si.Namespace,
			},
		})
		require.NoError(t, err)
		return r, res
	}

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegration](t, yamlServiceIntegrationAutoscalerLegacy)
		si.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, si.Spec.Project, si.Spec.SourceServiceName).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res := runScenario(t, si, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.ServiceIntegration{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Creates ServiceIntegration on Aiven when it doesn't exist", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegration](t, yamlServiceIntegrationAutoscalerLegacy)
		si.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, si.Spec.Project, si.Spec.SourceServiceName).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Twice()
		avn.EXPECT().
			ServiceIntegrationCreate(mock.Anything, si.Spec.Project, mock.MatchedBy(func(in *service.ServiceIntegrationCreateIn) bool {
				return in.IntegrationType == si.Spec.IntegrationType &&
					in.SourceService != nil && *in.SourceService == si.Spec.SourceServiceName &&
					in.DestEndpointId != nil && *in.DestEndpointId == si.Spec.DestinationEndpointID
			})).
			Return(&service.ServiceIntegrationCreateOut{ServiceIntegrationId: "si-123"}, nil).Once()

		r, res := runScenario(t, si, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegration{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "si-123", got.Status.ID)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Creates ServiceIntegration with endpoints only when source service name is empty", func(t *testing.T) {
		const yaml = `
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: test-si
  namespace: default
spec:
  project: test-project
  integrationType: autoscaler
  sourceEndpointId: source-endpoint-123
  destinationEndpointId: destination-endpoint-456
`
		si := newObjectFromYAML[v1alpha1.ServiceIntegration](t, yaml)
		si.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationCreate(mock.Anything, si.Spec.Project, mock.MatchedBy(func(in *service.ServiceIntegrationCreateIn) bool {
				return in.IntegrationType == si.Spec.IntegrationType &&
					in.SourceService == nil &&
					in.SourceEndpointId != nil && *in.SourceEndpointId == si.Spec.SourceEndpointID &&
					in.DestEndpointId != nil && *in.DestEndpointId == si.Spec.DestinationEndpointID
			})).
			Return(&service.ServiceIntegrationCreateOut{ServiceIntegrationId: "si-123"}, nil).Once()

		r, res := runScenario(t, si, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegration{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "si-123", got.Status.ID)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Adopts existing ServiceIntegration when configuration matches", func(t *testing.T) {
		const yaml = `
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: test-si
  namespace: default
spec:
  project: test-project
  integrationType: dashboard
  sourceServiceName: test-pg
  destinationServiceName: test-ch
`
		si := newObjectFromYAML[v1alpha1.ServiceIntegration](t, yaml)
		si.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, si.Spec.Project, si.Spec.DestinationServiceName).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()
		avn.EXPECT().
			ServiceGet(mock.Anything, si.Spec.Project, si.Spec.SourceServiceName).
			Return(&service.ServiceGetOut{
				State: service.ServiceStateTypeRunning,
				ServiceIntegrations: []service.ServiceIntegrationOut{
					{
						ServiceIntegrationId: "existing-123",
						IntegrationType:      si.Spec.IntegrationType,
						SourceService:        si.Spec.SourceServiceName,
						SourceProject:        si.Spec.Project,
						DestService:          ptr(si.Spec.DestinationServiceName),
						DestProject:          si.Spec.Project,
					},
				},
			}, nil).Twice()

		r, res := runScenario(t, si, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegration{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "existing-123", got.Status.ID)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Adopts existing ServiceIntegration and applies user config", func(t *testing.T) {
		const yaml = `
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: test-si
  namespace: default
spec:
  project: test-project
  integrationType: datadog
  sourceServiceName: test-pg
  datadog:
    datadog_dbm_enabled: true
`
		si := newObjectFromYAML[v1alpha1.ServiceIntegration](t, yaml)
		si.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, si.Spec.Project, si.Spec.SourceServiceName).
			Return(&service.ServiceGetOut{
				State: service.ServiceStateTypeRunning,
				ServiceIntegrations: []service.ServiceIntegrationOut{
					{
						ServiceIntegrationId: "existing-123",
						IntegrationType:      si.Spec.IntegrationType,
						SourceService:        si.Spec.SourceServiceName,
						SourceProject:        si.Spec.Project,
						DestProject:          si.Spec.Project,
					},
				},
			}, nil).Twice()
		avn.EXPECT().
			ServiceIntegrationUpdate(mock.Anything, si.Spec.Project, "existing-123", mock.MatchedBy(func(in *service.ServiceIntegrationUpdateIn) bool {
				enabled, ok := in.UserConfig["datadog_dbm_enabled"].(bool)
				return ok && enabled
			})).
			Return(nil, errors.New("User config not changed")).Once()

		r, res := runScenario(t, si, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegration{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "existing-123", got.Status.ID)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Updates ServiceIntegration user config and stores ID returned by API", func(t *testing.T) {
		const yaml = `
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: test-si
  namespace: default
spec:
  project: test-project
  integrationType: datadog
  datadog:
    datadog_dbm_enabled: true
`
		si := newObjectFromYAML[v1alpha1.ServiceIntegration](t, yaml)
		si.Generation = 1
		si.Status.ID = "si-123"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationGet(mock.Anything, si.Spec.Project, si.Status.ID).
			Return(&service.ServiceIntegrationGetOut{}, nil).Once()
		avn.EXPECT().
			ServiceIntegrationUpdate(mock.Anything, si.Spec.Project, si.Status.ID, mock.MatchedBy(func(in *service.ServiceIntegrationUpdateIn) bool {
				enabled, ok := in.UserConfig["datadog_dbm_enabled"].(bool)
				return ok && enabled
			})).
			Return(&service.ServiceIntegrationUpdateOut{ServiceIntegrationId: "si-456"}, nil).Once()

		r, res := runScenario(t, si, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegration{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "si-456", got.Status.ID)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Ignores \"user config not changed\" errors on update", func(t *testing.T) {
		const yaml = `
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: test-si
  namespace: default
spec:
  project: test-project
  integrationType: datadog
  datadog:
    datadog_dbm_enabled: true
`
		si := newObjectFromYAML[v1alpha1.ServiceIntegration](t, yaml)
		si.Generation = 1
		si.Status.ID = "si-123"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationGet(mock.Anything, si.Spec.Project, si.Status.ID).
			Return(&service.ServiceIntegrationGetOut{}, nil).Once()
		avn.EXPECT().
			ServiceIntegrationUpdate(mock.Anything, si.Spec.Project, si.Status.ID, mock.Anything).
			Return(nil, errors.New("User config not changed")).Once()

		r, res := runScenario(t, si, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegration{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "si-123", got.Status.ID)
	})

	t.Run("Removes finalizer on deletion when ServiceIntegration has no external ID", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegration](t, yamlServiceIntegrationAutoscalerLegacy)
		si.Generation = 1
		si.Finalizers = []string{instanceDeletionFinalizer}
		si.Spec.SourceServiceName = ""
		si.Spec.DestinationEndpointID = ""
		si.Status.ID = ""
		now := metav1.Now()
		si.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		r, res := runScenario(t, si, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ServiceIntegration{}
		err := r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Removes finalizer on deletion when Aiven returns NotFound", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegration](t, yamlServiceIntegrationAutoscalerLegacy)
		si.Generation = 1
		si.Finalizers = []string{instanceDeletionFinalizer}
		si.Spec.SourceServiceName = ""
		si.Spec.DestinationEndpointID = ""
		si.Status.ID = "si-123"
		now := metav1.Now()
		si.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationDelete(mock.Anything, si.Spec.Project, si.Status.ID).
			Return(newAivenError(404, "not found")).Once()

		r, res := runScenario(t, si, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ServiceIntegration{}
		err := r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
