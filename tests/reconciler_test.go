//go:build reconciler

package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	reconcilerStateProbeGroup    = "reconciler-tests.aiven.io"
	reconcilerStateProbeVersion  = "v1"
	reconcilerStateProbeKind     = "ReconcilerStateProbe"
	reconcilerStateProbeResource = "reconcilerstateprobes"

	reconcilerStateExtraAnnotation      = "example.com/extra"
	reconcilerStateConcurrentAnnotation = "example.com/concurrent"
	reconcilerStateProcessedAnnotation  = "controllers.aiven.io/generation-was-processed"
)

var reconcilerStateProbeGVK = schema.GroupVersionKind{
	Group:   reconcilerStateProbeGroup,
	Version: reconcilerStateProbeVersion,
	Kind:    reconcilerStateProbeKind,
}

func TestReconcilerStatePersistence(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	requireReconcilerStateProbeCRD(t, ctx)

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	reconcilerClient, err := client.NewWithWatch(restConfig, client.Options{Scheme: scheme})
	require.NoError(t, err)

	t.Run("External update before status update returns conflict and preserves state", func(t *testing.T) {
		ns := newReconcilerStateNamespace(t, ctx, reconcilerClient)
		stored := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Create(ctx, stored))

		stale := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Get(ctx, client.ObjectKeyFromObject(stored), stale))
		setProbeStatus(stale, "stale-status")

		latest := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Get(ctx, client.ObjectKeyFromObject(stored), latest))
		setAnnotation(latest, reconcilerStateExtraAnnotation, "fresh")
		require.NoError(t, reconcilerClient.Update(ctx, latest))

		err := reconcilerClient.Status().Update(ctx, stale)
		require.True(t, apierrors.IsConflict(err), err)

		got := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Get(ctx, client.ObjectKeyFromObject(stored), got))
		require.Equal(t, "fresh", got.GetAnnotations()[reconcilerStateExtraAnnotation])
		require.Empty(t, probeStatusUUID(got))
	})

	t.Run("External annotation update between status and annotation patch is not clobbered", func(t *testing.T) {
		ns := newReconcilerStateNamespace(t, ctx, reconcilerClient)
		stored := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Create(ctx, stored))

		obj := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Get(ctx, client.ObjectKeyFromObject(stored), obj))
		setProbeStatus(obj, "new-uuid")
		require.NoError(t, reconcilerClient.Status().Update(ctx, obj))

		latest := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Get(ctx, client.ObjectKeyFromObject(stored), latest))
		setAnnotation(latest, reconcilerStateConcurrentAnnotation, "fresh")
		require.NoError(t, reconcilerClient.Update(ctx, latest))

		err := patchProbeManagedAnnotations(ctx, reconcilerClient, obj, map[string]any{
			reconcilerStateProcessedAnnotation: "1",
		})
		require.True(t, apierrors.IsConflict(err), err)

		got := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Get(ctx, client.ObjectKeyFromObject(stored), got))
		require.Equal(t, "new-uuid", probeStatusUUID(got))
		require.Equal(t, "fresh", got.GetAnnotations()[reconcilerStateConcurrentAnnotation])
		require.NotContains(t, got.GetAnnotations(), reconcilerStateProcessedAnnotation)
	})

	t.Run("Managed annotation patch preserves annotations unknown to local object", func(t *testing.T) {
		ns := newReconcilerStateNamespace(t, ctx, reconcilerClient)
		stored := newReconcilerStateProbe(ns)
		setAnnotation(stored, reconcilerStateExtraAnnotation, "fresh")
		require.NoError(t, reconcilerClient.Create(ctx, stored))

		latest := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Get(ctx, client.ObjectKeyFromObject(stored), latest))

		obj := newReconcilerStateProbe(ns)
		obj.SetResourceVersion(latest.GetResourceVersion())
		setProbeStatus(obj, "new-uuid")
		require.NoError(t, reconcilerClient.Status().Update(ctx, obj))
		require.NoError(t, patchProbeManagedAnnotations(ctx, reconcilerClient, obj, map[string]any{
			reconcilerStateProcessedAnnotation: "1",
		}))

		got := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Get(ctx, client.ObjectKeyFromObject(stored), got))
		require.Equal(t, "new-uuid", probeStatusUUID(got))
		require.Equal(t, "fresh", got.GetAnnotations()[reconcilerStateExtraAnnotation])
		require.Equal(t, "1", got.GetAnnotations()[reconcilerStateProcessedAnnotation])
	})

	t.Run("Managed annotation removal removes only managed key", func(t *testing.T) {
		ns := newReconcilerStateNamespace(t, ctx, reconcilerClient)
		stored := newReconcilerStateProbe(ns)
		setAnnotation(stored, reconcilerStateExtraAnnotation, "fresh")
		setAnnotation(stored, reconcilerStateProcessedAnnotation, "1")
		require.NoError(t, reconcilerClient.Create(ctx, stored))

		obj := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Get(ctx, client.ObjectKeyFromObject(stored), obj))
		setProbeStatus(obj, "new-uuid")
		require.NoError(t, reconcilerClient.Status().Update(ctx, obj))
		require.NoError(t, patchProbeManagedAnnotations(ctx, reconcilerClient, obj, map[string]any{
			reconcilerStateProcessedAnnotation: nil,
		}))

		got := newReconcilerStateProbe(ns)
		require.NoError(t, reconcilerClient.Get(ctx, client.ObjectKeyFromObject(stored), got))
		require.Equal(t, "new-uuid", probeStatusUUID(got))
		require.Equal(t, "fresh", got.GetAnnotations()[reconcilerStateExtraAnnotation])
		require.NotContains(t, got.GetAnnotations(), reconcilerStateProcessedAnnotation)
	})
}

func requireReconcilerStateProbeCRD(t *testing.T, ctx context.Context) {
	t.Helper()

	extClient, err := apiextensionsclient.NewForConfig(restConfig)
	require.NoError(t, err)

	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: reconcilerStateProbeResource + "." + reconcilerStateProbeGroup,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: reconcilerStateProbeGroup,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:     reconcilerStateProbeResource,
				Singular:   "reconcilerstateprobe",
				Kind:       reconcilerStateProbeKind,
				ShortNames: []string{"rsp"},
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    reconcilerStateProbeVersion,
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"spec": {
									Type:                   "object",
									XPreserveUnknownFields: boolPtr(true),
								},
								"status": {
									Type:                   "object",
									XPreserveUnknownFields: boolPtr(true),
								},
							},
						},
					},
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
					},
				},
			},
		},
	}

	_, err = extClient.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		require.NoError(t, err)
	}

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		got, getErr := extClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crd.Name, metav1.GetOptions{})
		require.NoError(collect, getErr)
		require.Condition(collect, func() bool {
			for _, condition := range got.Status.Conditions {
				if condition.Type == apiextensionsv1.Established && condition.Status == apiextensionsv1.ConditionTrue {
					return true
				}
			}
			return false
		})
	}, time.Minute, time.Second)
}

func newReconcilerStateNamespace(t *testing.T, ctx context.Context, k8s client.Client) string {
	t.Helper()

	ns := randName("reconciler-state")
	require.NoError(t, k8s.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}))

	return ns
}

func newReconcilerStateProbe(namespace string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": reconcilerStateProbeGroup + "/" + reconcilerStateProbeVersion,
			"kind":       reconcilerStateProbeKind,
			"metadata": map[string]any{
				"name":      "test-probe",
				"namespace": namespace,
			},
			"spec": map[string]any{
				"project": "test-project",
			},
		},
	}
	obj.SetGroupVersionKind(reconcilerStateProbeGVK)
	return obj
}

func setProbeStatus(obj *unstructured.Unstructured, uuid string) {
	obj.Object["status"] = map[string]any{
		"uuid": uuid,
	}
}

func probeStatusUUID(obj *unstructured.Unstructured) string {
	status, ok := obj.Object["status"].(map[string]any)
	if !ok {
		return ""
	}

	value, ok := status["uuid"].(string)
	if !ok {
		return ""
	}

	return value
}

func setAnnotation(obj *unstructured.Unstructured, key, value string) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[key] = value
	obj.SetAnnotations(annotations)
}

func patchProbeManagedAnnotations(ctx context.Context, k8s client.Client, obj client.Object, annotations map[string]any) error {
	payload, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"resourceVersion": obj.GetResourceVersion(),
			"annotations":     annotations,
		},
	})
	if err != nil {
		return err
	}

	return k8s.Patch(ctx, obj, client.RawPatch(types.MergePatchType, payload))
}

func boolPtr(v bool) *bool {
	return &v
}
