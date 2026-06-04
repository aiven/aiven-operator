package controllers

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const (
	envtestExtraAnnotation      = "example.com/extra"
	envtestConcurrentAnnotation = "example.com/concurrent"
)

func TestReconciler_persistReconcileState_envtest(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	newEnvtestNamespace := func(t *testing.T, ctx context.Context, k8sClient client.Client) string {
		t.Helper()

		name := "reconciler-" + rand.String(8)
		require.NoError(t, k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}))

		return name
	}

	newEnvtestClickhouseUser := func(namespace string) *v1alpha1.ClickhouseUser {
		return &v1alpha1.ClickhouseUser{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.GroupVersion.String(),
				Kind:       "ClickhouseUser",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-user",
				Namespace: namespace,
			},
			Spec: v1alpha1.ClickhouseUserSpec{
				ServiceDependant: v1alpha1.ServiceDependant{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "test-project",
						},
					},
					ServiceField: v1alpha1.ServiceField{
						ServiceName: "test-service",
					},
				},
			},
		}
	}

	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := env.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, env.Stop())
	})

	k8sClient, err := client.NewWithWatch(cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)

	t.Run("External update before status update returns conflict and preserves state", func(t *testing.T) {
		ns := newEnvtestNamespace(t, ctx, k8sClient)
		stored := newEnvtestClickhouseUser(ns)
		require.NoError(t, k8sClient.Create(ctx, stored))

		stale := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(stored), stale))
		orig := stale.DeepCopy()
		stale.Status.UUID = "stale-status"
		stale.Status.Conditions = []metav1.Condition{}

		latest := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(stored), latest))
		metav1.SetMetaDataAnnotation(&latest.ObjectMeta, envtestExtraAnnotation, "fresh")
		require.NoError(t, k8sClient.Update(ctx, latest))

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		err := r.persistReconcileState(ctx, orig, stale)
		require.True(t, apierrors.IsConflict(err), err)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(stored), got))
		require.Equal(t, "fresh", got.Annotations[envtestExtraAnnotation])
		require.Empty(t, got.Status.UUID)
	})

	t.Run("External annotation update between status and annotation patch is not clobbered", func(t *testing.T) {
		ns := newEnvtestNamespace(t, ctx, k8sClient)
		stored := newEnvtestClickhouseUser(ns)
		require.NoError(t, k8sClient.Create(ctx, stored))

		obj := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(stored), obj))
		orig := obj.DeepCopy()
		obj.Status.UUID = "new-uuid"
		obj.Status.Conditions = []metav1.Condition{}
		metav1.SetMetaDataAnnotation(&obj.ObjectMeta, processedGenerationAnnotation, "1")

		key := client.ObjectKeyFromObject(stored)
		hookClient := interceptor.NewClient(k8sClient, interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if err := c.SubResource(subResourceName).Update(ctx, obj, opts...); err != nil {
					return err
				}

				latest := &v1alpha1.ClickhouseUser{}
				if err := k8sClient.Get(ctx, key, latest); err != nil {
					return err
				}

				metav1.SetMetaDataAnnotation(&latest.ObjectMeta, envtestConcurrentAnnotation, "fresh")
				return k8sClient.Update(ctx, latest)
			},
		})

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: hookClient,
			},
		}

		err := r.persistReconcileState(ctx, orig, obj)
		require.True(t, apierrors.IsConflict(err), err)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, key, got))
		require.Equal(t, "new-uuid", got.Status.UUID)
		require.Equal(t, "fresh", got.Annotations[envtestConcurrentAnnotation])
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	t.Run("Managed annotation patch preserves annotations unknown to local object", func(t *testing.T) {
		ns := newEnvtestNamespace(t, ctx, k8sClient)
		stored := newEnvtestClickhouseUser(ns)
		stored.Annotations = map[string]string{
			envtestExtraAnnotation: "fresh",
		}
		require.NoError(t, k8sClient.Create(ctx, stored))

		latest := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(stored), latest))

		obj := newEnvtestClickhouseUser(ns)
		obj.SetResourceVersion(latest.GetResourceVersion())
		orig := obj.DeepCopy()
		obj.Status.UUID = "new-uuid"
		obj.Status.Conditions = []metav1.Condition{}
		metav1.SetMetaDataAnnotation(&obj.ObjectMeta, processedGenerationAnnotation, "1")

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		require.NoError(t, r.persistReconcileState(ctx, orig, obj))

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(stored), got))
		require.Equal(t, "new-uuid", got.Status.UUID)
		require.Equal(t, "fresh", got.Annotations[envtestExtraAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Managed annotation removal removes only managed key", func(t *testing.T) {
		ns := newEnvtestNamespace(t, ctx, k8sClient)
		stored := newEnvtestClickhouseUser(ns)
		stored.Annotations = map[string]string{
			envtestExtraAnnotation:        "fresh",
			processedGenerationAnnotation: "1",
		}
		require.NoError(t, k8sClient.Create(ctx, stored))

		obj := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(stored), obj))
		orig := obj.DeepCopy()
		delete(obj.Annotations, processedGenerationAnnotation)
		obj.Status.UUID = "new-uuid"
		obj.Status.Conditions = []metav1.Condition{}

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		require.NoError(t, r.persistReconcileState(ctx, orig, obj))

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(stored), got))
		require.Equal(t, "new-uuid", got.Status.UUID)
		require.Equal(t, "fresh", got.Annotations[envtestExtraAnnotation])
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})
}
