// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// SecretFinalizerGCController manages the protection finalizer of the
// client token secrets, to give the controllers a chance to delete the
// aiven instances
type SecretFinalizerGCController struct {
	client.Client

	Log logr.Logger
}

func (c *SecretFinalizerGCController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	secret := &corev1.Secret{}
	if err := c.Get(ctx, req.NamespacedName, secret); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// we only care about secrets that are about to be deleted and have our finalizer
	if !markedForDeletion(secret) || !controllerutil.ContainsFinalizer(secret, secretProtectionFinalizer) {
		return ctrl.Result{}, nil
	}

	// check for dangeling instances that still need the secret for deletion
	if isStillNeeded, err := c.secretIsStillNeeded(ctx, secret); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to check if secret is still needed: %w", err)
	} else if isStillNeeded {
		c.Log.Info("secret is still needed, requeueing deletion")
		return requeueCtrlResult(), nil
	}

	c.Log.Info("removing secret protection finalizer")

	// secret is not needed anymore
	if err := removeFinalizer(ctx, c, secret, secretProtectionFinalizer); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to remove secret protection finalizer: %w", err)
	}

	return ctrl.Result{}, nil
}

func (c *SecretFinalizerGCController) SetupWithManager(mgr ctrl.Manager) error {
	if err := indexClientSecretRefFields(context.Background(), mgr, c.knownInstanceTypes()...); err != nil {
		return fmt.Errorf("unable to add index for secret ref fields: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).For(&corev1.Secret{}).Complete(c)
}

func (c *SecretFinalizerGCController) knownListTypes() []client.ObjectList {
	res := make([]client.ObjectList, 0)

	for _, t := range c.Scheme().KnownTypes(v1alpha1.GroupVersion) {
		if list, ok := reflect.New(t).Interface().(client.ObjectList); ok {
			res = append(res, list)
		}
	}
	return res
}

func (c *SecretFinalizerGCController) knownInstanceTypes() []aivenManagedObject {
	res := make([]aivenManagedObject, 0)

	for _, t := range c.Scheme().KnownTypes(v1alpha1.GroupVersion) {
		if obj, ok := reflect.New(t).Interface().(aivenManagedObject); ok {
			res = append(res, obj)
		}
	}
	return res
}

func (c *SecretFinalizerGCController) secretIsStillNeeded(ctx context.Context, secret *corev1.Secret) (bool, error) {
	for _, listType := range c.knownListTypes() {
		if needed, err := c.secretIsStillNeededBy(ctx, secret, listType); err != nil {
			return false, fmt.Errorf("unable to decide if secret is still used by some aiven resource: %w", err)
		} else if needed {
			return true, nil
		}
	}
	return false, nil
}

const (
	// secretRefIndexKey is the key we index the name of the secret with
	// so we can efficiently list all resources that use this secret
	secretRefIndexKey = "spec.auth_secret_ref.name"
)

// secretRefIndexFunc indexes the client token secret names of aiven managed objects
func secretRefIndexFunc(o client.Object) []string {
	if aivenObj, ok := o.(aivenManagedObject); ok {
		return []string{aivenObj.AuthSecretRef().Name}
	}
	return nil
}

func indexClientSecretRefFields(ctx context.Context, mgr ctrl.Manager, objs ...aivenManagedObject) error {
	for i := range objs {
		if err := mgr.GetFieldIndexer().IndexField(ctx, objs[i], secretRefIndexKey, secretRefIndexFunc); err != nil {
			return err
		}
	}
	return nil
}

// check if an instance uses this secret
func instancesThatUseThisSecret(secret *corev1.Secret) *client.ListOptions {
	return &client.ListOptions{
		Namespace:     secret.GetNamespace(),
		FieldSelector: fields.OneTermEqualSelector(secretRefIndexKey, secret.GetName()),
		Limit:         1,
	}
}

func (c *SecretFinalizerGCController) secretIsStillNeededBy(ctx context.Context, secret *corev1.Secret, list client.ObjectList) (bool, error) {
	if err := c.List(ctx, list, instancesThatUseThisSecret(secret)); err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return meta.LenList(list) > 0, nil
}
