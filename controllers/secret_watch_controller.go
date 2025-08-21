// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// SecretWatchController watches for changes to secrets referenced by connInfoSecretSource
// and triggers reconciliation of the dependent resources
type SecretWatchController struct {
	client.Client

	Log logr.Logger
}

const (
	// connInfoSecretRefIndexKey is the key we index the name of the secret with
	connInfoSecretRefIndexKey = "spec.conn_info_secret_source.name"
)

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (c *SecretWatchController) SetupWithManager(mgr ctrl.Manager) error {
	resourcesWithSecretSource := c.getResourcesWithSecretSource()

	if err := c.indexConnInfoSecretRefFields(context.Background(), mgr, resourcesWithSecretSource...); err != nil {
		return fmt.Errorf("unable to add index for connInfoSecretSource fields: %w", err)
	}

	builder := ctrl.NewControllerManagedBy(mgr)
	builder.For(&corev1.Secret{})

	// only watch for update events on secrets
	builder.WithEventFilter(predicate.Funcs{
		CreateFunc:  func(_ event.CreateEvent) bool { return false },
		UpdateFunc:  func(e event.UpdateEvent) bool { return c.secretDataChanged(e) },
		DeleteFunc:  func(_ event.DeleteEvent) bool { return false },
		GenericFunc: func(_ event.GenericEvent) bool { return false },
	})

	// watch CRDs that have connInfoSecretSource to queue reconciliations
	for i := range resourcesWithSecretSource {
		builder.Watches(
			&source.Kind{Type: resourcesWithSecretSource[i]},
			handler.EnqueueRequestsFromMapFunc(func(a client.Object) []reconcile.Request {
				if resource, ok := a.(SecretSourceResource); ok {
					if secretSource := resource.GetConnInfoSecretSource(); secretSource != nil {
						sourceNamespace := secretSource.Namespace
						if sourceNamespace == "" {
							sourceNamespace = resource.GetNamespace()
						}

						return []reconcile.Request{
							{
								NamespacedName: types.NamespacedName{
									Name:      secretSource.Name,
									Namespace: sourceNamespace,
								},
							},
						}
					}
				}

				return nil
			}),
		)
	}

	return builder.Complete(c)
}

func (c *SecretWatchController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.Log.Info("[SECRET_WATCHER] Processing secret change", "secret", req.NamespacedName)

	secret := &corev1.Secret{}
	if err := c.Get(ctx, req.NamespacedName, secret); err != nil {
		if errors.IsNotFound(err) {
			c.Log.Info("[SECRET_WATCHER] Secret not found, skipping", "secret", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		c.Log.Error(err, "[SECRET_WATCHER] Failed to get secret", "secret", req.NamespacedName)
		return ctrl.Result{}, err
	}

	c.Log.Info("[SECRET_WATCHER] Secret found, searching for dependent resources",
		"secret", req.NamespacedName,
		"secretRV", secret.GetResourceVersion())

	dependentResources, err := c.findResourcesUsingSecret(ctx, secret)
	if err != nil {
		c.Log.Error(err, "[SECRET_WATCHER] Failed to find resources using secret", "secret", req.NamespacedName)
		return ctrl.Result{}, fmt.Errorf("unable to find resources using secret: %w", err)
	}

	if len(dependentResources) == 0 {
		c.Log.Info("[SECRET_WATCHER] No resources found using this secret", "secret", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	c.Log.Info("[SECRET_WATCHER] Triggering reconciliation for dependent resources",
		"secret", req.NamespacedName,
		"dependentCount", len(dependentResources))

	// trigger reconciliation for each dependent resource
	for i, resource := range dependentResources {
		resourceName := types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()}
		c.Log.Info("[SECRET_WATCHER] Processing dependent resource",
			"index", i+1,
			"total", len(dependentResources),
			"resource", resourceName,
			"resourceRV", resource.GetResourceVersion(),
			"kind", resource.GetObjectKind().GroupVersionKind().Kind)

		if err = c.triggerReconciliation(ctx, resource); err != nil {
			c.Log.Error(err, "[SECRET_WATCHER] Failed to trigger reconciliation for resource",
				"resource", resourceName,
				"kind", resource.GetObjectKind().GroupVersionKind().Kind)
		} else {
			c.Log.Info("[SECRET_WATCHER] Successfully triggered reconciliation for resource", "resource", resourceName)
		}
	}

	c.Log.Info("[SECRET_WATCHER] Completed processing secret change", "secret", req.NamespacedName)
	return ctrl.Result{}, nil
}

// SecretSourceResource defines an interface for resources that can have connInfoSecretSource
type SecretSourceResource interface {
	client.Object
	GetConnInfoSecretSource() *v1alpha1.ConnInfoSecretSource
}

// getResourcesWithSecretSource returns all known resource types that implement SecretSourceResource
func (c *SecretWatchController) getResourcesWithSecretSource() []SecretSourceResource {
	return []SecretSourceResource{
		&v1alpha1.ServiceUser{},
		&v1alpha1.ClickhouseUser{},
	}
}

// connInfoSecretRefIndexFunc indexes the secret names referenced by connInfoSecretSource
func connInfoSecretRefIndexFunc(o client.Object) []string {
	if resource, ok := o.(SecretSourceResource); ok {
		if secretSource := resource.GetConnInfoSecretSource(); secretSource != nil {
			sourceNamespace := secretSource.Namespace
			if sourceNamespace == "" {
				sourceNamespace = resource.GetNamespace()
			}

			return []string{fmt.Sprintf("%s/%s", sourceNamespace, secretSource.Name)}
		}
	}

	return nil
}

func (c *SecretWatchController) indexConnInfoSecretRefFields(ctx context.Context, mgr ctrl.Manager, resources ...SecretSourceResource) error {
	for i := range resources {
		if err := mgr.GetFieldIndexer().IndexField(ctx, resources[i], connInfoSecretRefIndexKey, connInfoSecretRefIndexFunc); err != nil {
			return err
		}
	}

	return nil
}

// secretDataChanged checks if the secret's data actually changed (not just metadata)
func (c *SecretWatchController) secretDataChanged(e event.UpdateEvent) bool {
	oldSec, oldOk := e.ObjectOld.(*corev1.Secret)
	newSec, newOk := e.ObjectNew.(*corev1.Secret)

	if !oldOk || !newOk {
		return false
	}

	return !reflect.DeepEqual(oldSec.Data, newSec.Data)
}

// findResourcesUsingSecret finds all resources that reference the given secret as connInfoSecretSource
func (c *SecretWatchController) findResourcesUsingSecret(ctx context.Context, secret *corev1.Secret) ([]SecretSourceResource, error) {
	secretKey := fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)
	var allResources []SecretSourceResource //nolint:prealloc

	serviceUserList := &v1alpha1.ServiceUserList{}
	err := c.List(ctx, serviceUserList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(connInfoSecretRefIndexKey, secretKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list ServiceUsers: %w", err)
	}

	for i := range serviceUserList.Items {
		allResources = append(allResources, &serviceUserList.Items[i])
	}

	clickhouseUserList := &v1alpha1.ClickhouseUserList{}
	err = c.List(ctx, clickhouseUserList, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(connInfoSecretRefIndexKey, secretKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list ClickhouseUsers: %w", err)
	}

	for i := range clickhouseUserList.Items {
		allResources = append(allResources, &clickhouseUserList.Items[i])
	}

	return allResources, nil
}

// triggerReconciliation triggers reconciliation by patching the resource's annotations
func (c *SecretWatchController) triggerReconciliation(ctx context.Context, resource SecretSourceResource) error {
	resourceName := types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()}

	timestamp := time.Now().Unix()
	c.Log.Info("[SECRET_WATCHER] Preparing to patch resource annotations",
		"resource", resourceName,
		"timestamp", timestamp,
		"currentRV", resource.GetResourceVersion(),
		"currentGeneration", resource.GetGeneration(),
		"currentAnnotations", resource.GetAnnotations(),
		"currentLabels", resource.GetLabels())

	patchData := map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]any{
				secretSourceUpdatedAnnotation: fmt.Sprintf("%d", timestamp),
				processedGenerationAnnotation: nil, // null remove the annotation
			},
		},
	}

	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		c.Log.Error(err, "[SECRET_WATCHER] Failed to marshal patch data", "resource", resourceName)
		return fmt.Errorf("failed to marshal patch data: %w", err)
	}

	c.Log.Info("[SECRET_WATCHER] Attempting patch operation",
		"resource", resourceName,
		"patchData", string(patchBytes))

	if err = c.Patch(ctx, resource, client.RawPatch(types.MergePatchType, patchBytes)); err != nil {
		if errors.IsConflict(err) {
			c.Log.Info("[SECRET_WATCHER] Resource modified by another controller, skipping annotation update",
				"resource", resourceName,
				"reason", "main controller is handling this change",
				"error", err.Error())
			return nil // this is expected - another controller is processing the resource
		}

		c.Log.Error(err, "[SECRET_WATCHER] Failed to patch resource annotations",
			"resource", resourceName,
			"patchData", string(patchBytes))
		return fmt.Errorf("failed to patch resource annotations: %w", err)
	}

	c.Log.Info("[SECRET_WATCHER] Successfully patched resource annotations",
		"resource", resourceName,
		"timestamp", timestamp)
	return nil
}
