package controllers

import (
	"errors"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
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
	prometheususerconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integrationendpoints/prometheus"
)

const yamlServiceIntegrationEndpointDatadog = `
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegrationEndpoint
metadata:
  name: test-sie
  namespace: default
spec:
  project: test-project
  endpointName: my-endpoint
  endpointType: datadog
  datadog:
    datadog_api_key: AAAAAAAAAAAAAAAA
    site: datadoghq.com
`

func runServiceIntegrationEndpointScenario(
	t *testing.T,
	si *v1alpha1.ServiceIntegrationEndpoint,
	avn avngen.Client,
	additionalObjects ...client.Object,
) (*Reconciler[*v1alpha1.ServiceIntegrationEndpoint], ctrlruntime.Result, error) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	objects := append([]client.Object{si}, additionalObjects...)

	r := newServiceIntegrationEndpointReconciler(Controller{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ServiceIntegrationEndpoint{}).
			WithObjects(objects...).
			Build(),
		Scheme:       scheme,
		Recorder:     record.NewFakeRecorder(10),
		DefaultToken: "test-token",
		PollInterval: testPollInterval,
	}).(*Reconciler[*v1alpha1.ServiceIntegrationEndpoint])
	r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
		return avn, nil
	}

	res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
		NamespacedName: types.NamespacedName{
			Name:      si.Name,
			Namespace: si.Namespace,
		},
	})
	return r, res, err
}

func TestServiceIntegrationEndpointReconciler(t *testing.T) {
	t.Parallel()

	t.Run("Creates ServiceIntegrationEndpoint on Aiven when it doesn't exist", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1

		avn := avngen.NewMockClient(t)
		// Observe: no Status.ID yet, so it lists to look for an adoptable endpoint and finds none.
		avn.EXPECT().
			ServiceIntegrationEndpointList(mock.Anything, si.Spec.Project).
			Return(nil, nil).Once()
		avn.EXPECT().
			ServiceIntegrationEndpointCreate(mock.Anything, si.Spec.Project, mock.MatchedBy(func(in *service.ServiceIntegrationEndpointCreateIn) bool {
				return in.EndpointName == si.Spec.EndpointName &&
					in.EndpointType == service.EndpointType(si.Spec.EndpointType) &&
					len(in.UserConfig) > 0
			})).
			Return(&service.ServiceIntegrationEndpointCreateOut{EndpointId: "endpoint-123"}, nil).Once()

		r, res, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegrationEndpoint{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "endpoint-123", got.Status.ID)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Adopts existing endpoint by name+type when Status.ID is empty", func(t *testing.T) {
		// Simulates a lost status write
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationEndpointList(mock.Anything, si.Spec.Project).
			Return([]service.ServiceIntegrationEndpointOut{
				{EndpointId: "other-id", EndpointName: "different-name", EndpointType: service.EndpointType(si.Spec.EndpointType)},
				{EndpointId: "adopted-id", EndpointName: si.Spec.EndpointName, EndpointType: service.EndpointType(si.Spec.EndpointType)},
			}, nil).Once()
		// No Create is expected. Adoption marks the resource as not up-to-date, so Update runs.
		avn.EXPECT().
			ServiceIntegrationEndpointUpdate(mock.Anything, si.Spec.Project, "adopted-id", mock.Anything).
			Return(&service.ServiceIntegrationEndpointUpdateOut{EndpointId: "adopted-id"}, nil).Once()

		r, res, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegrationEndpoint{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "adopted-id", got.Status.ID)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Marks running and requeues when endpoint already exists and is up to date", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1
		si.Status.ID = "endpoint-123"
		si.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationEndpointGet(mock.Anything, si.Spec.Project, si.Status.ID).
			Return(&service.ServiceIntegrationEndpointGetOut{EndpointId: si.Status.ID}, nil).Once()

		r, res, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegrationEndpoint{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "endpoint-123", got.Status.ID)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Updates endpoint user config and tolerates \"user config not changed\"", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1
		si.Status.ID = "endpoint-123"

		avn := avngen.NewMockClient(t)
		// Observe: endpoint exists but not up to date (no processed-generation annotation yet).
		avn.EXPECT().
			ServiceIntegrationEndpointGet(mock.Anything, si.Spec.Project, si.Status.ID).
			Return(&service.ServiceIntegrationEndpointGetOut{EndpointId: si.Status.ID}, nil).Once()
		avn.EXPECT().
			ServiceIntegrationEndpointUpdate(mock.Anything, si.Spec.Project, si.Status.ID, mock.MatchedBy(func(in *service.ServiceIntegrationEndpointUpdateIn) bool {
				return len(in.UserConfig) > 0
			})).
			Return(nil, errors.New("User config not changed")).Once()

		r, res, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegrationEndpoint{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "endpoint-123", got.Status.ID)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Updates endpoint user config and stores ID returned by API", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1
		si.Status.ID = "endpoint-123"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationEndpointGet(mock.Anything, si.Spec.Project, si.Status.ID).
			Return(&service.ServiceIntegrationEndpointGetOut{EndpointId: si.Status.ID}, nil).Once()
		avn.EXPECT().
			ServiceIntegrationEndpointUpdate(mock.Anything, si.Spec.Project, si.Status.ID, mock.Anything).
			Return(&service.ServiceIntegrationEndpointUpdateOut{EndpointId: "endpoint-456"}, nil).Once()

		r, res, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegrationEndpoint{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "endpoint-456", got.Status.ID)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Returns error when Get fails with a non-retryable error", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1
		si.Status.ID = "endpoint-123"

		avn := avngen.NewMockClient(t)
		// 400 is not retryable, so the reconciler surfaces it as an error rather than requeueing.
		avn.EXPECT().
			ServiceIntegrationEndpointGet(mock.Anything, si.Spec.Project, si.Status.ID).
			Return(nil, newAivenError(400, "bad request")).Once()

		_, _, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.Error(t, err)
	})

	t.Run("Returns error when Update fails with a non-tolerated error", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1
		si.Status.ID = "endpoint-123"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationEndpointGet(mock.Anything, si.Spec.Project, si.Status.ID).
			Return(&service.ServiceIntegrationEndpointGetOut{EndpointId: si.Status.ID}, nil).Once()
		avn.EXPECT().
			ServiceIntegrationEndpointUpdate(mock.Anything, si.Spec.Project, si.Status.ID, mock.Anything).
			Return(nil, errors.New("some other failure")).Once()

		_, _, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.Error(t, err)
	})

	t.Run("Returns error when more than one endpoint config is set", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1
		// datadog is set via YAML; add a conflicting second config block.
		si.Spec.Prometheus = &prometheususerconfig.PrometheusUserConfig{}

		avn := avngen.NewMockClient(t)
		// Observe lists for adoption and finds nothing; Create must then fail in GetUserConfig
		// before any ServiceIntegrationEndpointCreate call is made.
		avn.EXPECT().
			ServiceIntegrationEndpointList(mock.Anything, si.Spec.Project).
			Return(nil, nil).Once()

		_, _, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.Error(t, err)
	})

	t.Run("Resets Status.ID and recreates when Get returns NotFound", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1
		si.Status.ID = "stale-id"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationEndpointGet(mock.Anything, si.Spec.Project, "stale-id").
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceIntegrationEndpointCreate(mock.Anything, si.Spec.Project, mock.MatchedBy(func(in *service.ServiceIntegrationEndpointCreateIn) bool {
				return in.EndpointType == service.EndpointType(si.Spec.EndpointType)
			})).
			Return(&service.ServiceIntegrationEndpointCreateOut{EndpointId: "endpoint-789"}, nil).Once()

		r, res, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceIntegrationEndpoint{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got))
		require.Equal(t, "endpoint-789", got.Status.ID)
	})

	t.Run("Deletes endpoint and removes finalizer on deletion", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1
		si.Status.ID = "endpoint-123"
		si.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		si.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationEndpointDelete(mock.Anything, si.Spec.Project, si.Status.ID).
			Return(nil).Once()

		r, res, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ServiceIntegrationEndpoint{}
		err = r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Treats NotFound on delete as already deleted", func(t *testing.T) {
		si := newObjectFromYAML[v1alpha1.ServiceIntegrationEndpoint](t, yamlServiceIntegrationEndpointDatadog)
		si.Generation = 1
		si.Status.ID = "endpoint-123"
		si.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		si.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceIntegrationEndpointDelete(mock.Anything, si.Spec.Project, si.Status.ID).
			Return(newAivenError(404, "not found")).Once()

		r, res, err := runServiceIntegrationEndpointScenario(t, si, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ServiceIntegrationEndpoint{}
		err = r.Get(t.Context(), types.NamespacedName{Name: si.Name, Namespace: si.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
