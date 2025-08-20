// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
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
	c.Log.Info("SECRET WATCHER: Starting reconciliation", "secret", req.NamespacedName)

	secret := &corev1.Secret{}
	if err := c.Get(ctx, req.NamespacedName, secret); err != nil {
		if errors.IsNotFound(err) {
			c.Log.Info("SECRET WATCHER: Secret not found, skipping", "secret", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		c.Log.Error(err, "SECRET WATCHER: Failed to get secret", "secret", req.NamespacedName)
		return ctrl.Result{}, err
	}

	c.Log.Info("SECRET WATCHER: Secret found, looking for dependent resources",
		"secret", req.NamespacedName,
		"resourceVersion", secret.ResourceVersion,
		"generation", secret.Generation)

	dependentResources, err := c.findResourcesUsingSecret(ctx, secret)
	if err != nil {
		c.Log.Error(err, "SECRET WATCHER: Failed to find dependent resources", "secret", req.NamespacedName)
		return ctrl.Result{}, fmt.Errorf("unable to find resources using secret: %w", err)
	}

	if len(dependentResources) == 0 {
		c.Log.Info("SECRET WATCHER: No resources found using this secret", "secret", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	c.Log.Info("SECRET WATCHER: Found dependent resources, triggering reconciliation",
		"secret", req.NamespacedName,
		"dependentCount", len(dependentResources))

	// trigger reconciliation for each dependent resource
	for i, resource := range dependentResources {
		resourceName := types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()}
		c.Log.Info("SECRET WATCHER: Processing dependent resource",
			"index", i,
			"resource", resourceName,
			"kind", resource.GetObjectKind().GroupVersionKind().Kind)

		if err = c.triggerReconciliation(ctx, resource); err != nil {
			c.Log.Error(err, "SECRET WATCHER: Failed to trigger reconciliation for resource",
				"resource", resourceName,
				"kind", resource.GetObjectKind().GroupVersionKind().Kind)
		}
	}

	c.Log.Info("SECRET WATCHER: Completed reconciliation", "secret", req.NamespacedName)
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
		c.Log.Info("SECRET WATCHER: secretDataChanged called with invalid secret types")
		return false
	}

	secretName := types.NamespacedName{Name: newSec.Name, Namespace: newSec.Namespace}
	dataChanged := !reflect.DeepEqual(oldSec.Data, newSec.Data)

	c.Log.Info("SECRET WATCHER: Checking if secret data changed",
		"secret", secretName,
		"oldResourceVersion", oldSec.ResourceVersion,
		"newResourceVersion", newSec.ResourceVersion,
		"dataChanged", dataChanged)

	return dataChanged
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

// triggerReconciliation triggers reconciliation by updating the resource's annotation
func (c *SecretWatchController) triggerReconciliation(ctx context.Context, resource SecretSourceResource) error {
	resourceName := types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()}

	c.Log.Info("SECRET WATCHER: Getting latest resource version", "resource", resourceName)

	latest := resource.DeepCopyObject().(client.Object)
	if err := c.Get(ctx, resourceName, latest); err != nil {
		c.Log.Error(err, "SECRET WATCHER: Failed to get latest version of resource", "resource", resourceName)
		return fmt.Errorf("failed to get latest version of resource: %w", err)
	}

	c.Log.Info("SECRET WATCHER: Current resource state",
		"resource", resourceName,
		"resourceVersion", latest.GetResourceVersion(),
		"generation", latest.GetGeneration(),
		"annotations", latest.GetAnnotations())

	annotations := latest.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Store current state for logging
	oldProcessedGen := annotations[processedGenerationAnnotation]
	oldSecretSourceUpdated := annotations[secretSourceUpdatedAnnotation]

	annotations[secretSourceUpdatedAnnotation] = fmt.Sprintf("%d", time.Now().Unix())

	// clear the processed generation annotation to force the basic controller to reconcile
	delete(annotations, processedGenerationAnnotation)

	latest.SetAnnotations(annotations)

	c.Log.Info("SECRET WATCHER: Updating resource annotations",
		"resource", resourceName,
		"oldProcessedGeneration", oldProcessedGen,
		"oldSecretSourceUpdated", oldSecretSourceUpdated,
		"newSecretSourceUpdated", annotations[secretSourceUpdatedAnnotation],
		"deletingProcessedGeneration", oldProcessedGen != "")

	if err := c.Update(ctx, latest); err != nil {
		if errors.IsConflict(err) {
			c.Log.Info("SECRET WATCHER: Resource modified by another controller, skipping annotation update",
				"resource", resourceName,
				"resourceVersion", latest.GetResourceVersion(),
				"reason", "main controller is handling this change")
			return nil // this is expected - another controller is processing the resource
		}

		c.Log.Error(err, "SECRET WATCHER: Failed to update resource annotation", "resource", resourceName)
		return fmt.Errorf("failed to update resource annotation: %w", err)
	}

	c.Log.Info("SECRET WATCHER: Successfully triggered reconciliation for resource",
		"resource", resourceName,
		"kind", latest.GetObjectKind().GroupVersionKind().Kind)

	return nil
}
