package controllers

import (
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	avnopensearch "github.com/aiven/go-client-codegen/handler/opensearch"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/equality"
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

func TestOpenSearchACLConfigReconciler(t *testing.T) {
	t.Parallel()

	const yamlOpenSearchACLConfig = `
apiVersion: aiven.io/v1alpha1
kind: OpenSearchACLConfig
metadata:
  name: search-acl
  namespace: default
spec:
  project: test-project
  serviceName: test-service
  enabled: true
  acls:
    - username: admin*
      rules:
        - index: ind*
          permission: deny
        - index: logs*
          permission: read
    - username: ops*
      rules:
        - index: metrics*
          permission: write
`

	runScenario := func(t *testing.T, cfg *v1alpha1.OpenSearchACLConfig, avn avngen.Client, additionalObjects ...client.Object) (*Reconciler[*v1alpha1.OpenSearchACLConfig], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		objects := append([]client.Object{cfg}, additionalObjects...)

		r := newOpenSearchACLConfigReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.OpenSearchACLConfig{}).
				WithObjects(objects...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(10),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.OpenSearchACLConfig])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{
				Name:      cfg.Name,
				Namespace: cfg.Namespace,
			},
		})
		return r, res, err
	}

	t.Run("Returns error when OpenSearch ACL config get returns not found", func(t *testing.T) {
		cfg := newObjectFromYAML[v1alpha1.OpenSearchACLConfig](t, yamlOpenSearchACLConfig)
		cfg.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, cfg.Spec.Project, cfg.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()
		avn.EXPECT().
			ServiceOpenSearchAclGet(mock.Anything, cfg.Spec.Project, cfg.Spec.ServiceName).
			Return(nil, newAivenError(404, "not found")).Once()

		r, res, err := runScenario(t, cfg, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.OpenSearchACLConfig{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cfg.Name, Namespace: cfg.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Empty(t, got.Annotations[instanceIsRunningAnnotation])
		require.Empty(t, got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Updates OpenSearchACLConfig when remote config drifts", func(t *testing.T) {
		cfg := newObjectFromYAML[v1alpha1.OpenSearchACLConfig](t, yamlOpenSearchACLConfig)
		cfg.Generation = 1
		cfg.Annotations = map[string]string{
			instanceIsRunningAnnotation:   "true",
			processedGenerationAnnotation: "1",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, cfg.Spec.Project, cfg.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()
		avn.EXPECT().
			ServiceOpenSearchAclGet(mock.Anything, cfg.Spec.Project, cfg.Spec.ServiceName).
			Return(&avnopensearch.ServiceOpenSearchAclGetOut{
				OpensearchAclConfig: avnopensearch.OpensearchAclConfigOut{
					Enabled: false,
					Acls: []avnopensearch.AclOut{
						{
							Username: "admin*",
							Rules: []avnopensearch.RuleOut{
								{Index: "ind*", Permission: avnopensearch.PermissionTypeRead},
							},
						},
					},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceOpenSearchAclSet(mock.Anything, cfg.Spec.Project, cfg.Spec.ServiceName, mock.MatchedBy(func(in *avnopensearch.ServiceOpenSearchAclSetIn) bool {
				return equality.Semantic.DeepEqual(in, &avnopensearch.ServiceOpenSearchAclSetIn{
					OpensearchAclConfig: avnopensearch.OpensearchAclConfigIn{
						Acls:    buildOpenSearchACLsIn(cfg.Spec.Acls),
						Enabled: cfg.Spec.Enabled,
					},
				})
			})).
			Return(&avnopensearch.ServiceOpenSearchAclSetOut{}, nil).Once()

		r, res, err := runScenario(t, cfg, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.OpenSearchACLConfig{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cfg.Name, Namespace: cfg.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Skips write when remote config already matches ignoring ACL and rule order", func(t *testing.T) {
		cfg := newObjectFromYAML[v1alpha1.OpenSearchACLConfig](t, yamlOpenSearchACLConfig)
		cfg.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, cfg.Spec.Project, cfg.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()
		avn.EXPECT().
			ServiceOpenSearchAclGet(mock.Anything, cfg.Spec.Project, cfg.Spec.ServiceName).
			Return(&avnopensearch.ServiceOpenSearchAclGetOut{
				OpensearchAclConfig: avnopensearch.OpensearchAclConfigOut{
					Enabled: true,
					Acls: []avnopensearch.AclOut{
						{
							Username: "ops*",
							Rules: []avnopensearch.RuleOut{
								{Index: "metrics*", Permission: avnopensearch.PermissionTypeWrite},
							},
						},
						{
							Username: "admin*",
							Rules: []avnopensearch.RuleOut{
								{Index: "logs*", Permission: avnopensearch.PermissionTypeRead},
								{Index: "ind*", Permission: avnopensearch.PermissionTypeDeny},
							},
						},
					},
				},
			}, nil).Once()

		r, res, err := runScenario(t, cfg, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.OpenSearchACLConfig{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cfg.Name, Namespace: cfg.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Resets remote OpenSearch ACL config on deletion", func(t *testing.T) {
		cfg := newObjectFromYAML[v1alpha1.OpenSearchACLConfig](t, yamlOpenSearchACLConfig)
		cfg.Generation = 1
		cfg.Finalizers = []string{instanceDeletionFinalizer}
		cfg.DeletionTimestamp = new(metav1.Now())

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceOpenSearchAclSet(mock.Anything, cfg.Spec.Project, cfg.Spec.ServiceName, mock.MatchedBy(func(in *avnopensearch.ServiceOpenSearchAclSetIn) bool {
				return equality.Semantic.DeepEqual(in, &avnopensearch.ServiceOpenSearchAclSetIn{
					OpensearchAclConfig: avnopensearch.OpensearchAclConfigIn{
						Acls:    []avnopensearch.AclIn{},
						Enabled: false,
					},
				})
			})).
			Return(&avnopensearch.ServiceOpenSearchAclSetOut{}, nil).Once()

		r, res, err := runScenario(t, cfg, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.OpenSearchACLConfig{}
		err = r.Get(t.Context(), types.NamespacedName{Name: cfg.Name, Namespace: cfg.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Removes finalizer on deletion when OpenSearch ACL config is already gone", func(t *testing.T) {
		cfg := newObjectFromYAML[v1alpha1.OpenSearchACLConfig](t, yamlOpenSearchACLConfig)
		cfg.Generation = 1
		cfg.Finalizers = []string{instanceDeletionFinalizer}
		cfg.DeletionTimestamp = new(metav1.Now())

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceOpenSearchAclSet(mock.Anything, cfg.Spec.Project, cfg.Spec.ServiceName, mock.AnythingOfType("*opensearch.ServiceOpenSearchAclSetIn")).
			Return(nil, newAivenError(404, "not found")).Once()

		r, res, err := runScenario(t, cfg, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.OpenSearchACLConfig{}
		err = r.Get(t.Context(), types.NamespacedName{Name: cfg.Name, Namespace: cfg.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
