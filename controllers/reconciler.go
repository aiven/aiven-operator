package controllers

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// Reconciler handles the boilerplate reconciliation logic for Aiven resources.
// It orchestrates the ExternalClient lifecycle methods and manages:
// - Finalizers
// - Status conditions
// - Secrets (connection details)
// - Events
// - Requeue logic
type Reconciler[T v1alpha1.AivenManagedObject] struct {
	Controller
	newAivenGeneratedClient func(token, kubeVersion, operatorVersion string) (avngen.Client, error)
	newController           func(avnGen avngen.Client) AivenController[T]
	newObj                  func() T
	newSecret               func(o objWithSecret, stringData map[string]string, addPrefix bool) *corev1.Secret
}

// pollInterval controls how often we re-run reconciliation for resources that are in a steady state.
// This enables continuous reconciliation without overwhelming the Aiven API.
const pollInterval = 1 * time.Minute

// requeueTimeout sets timeout to requeue controller
const requeueTimeout = 10 * time.Second

// Reconcile performs the full reconciliation loop for a managed resource.
func (r *Reconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	obj := r.newObj()
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	ctx = logr.NewContext(ctx, setupLogger(r.Log, obj))

	if requeue, err := r.resolveK8sRefs(ctx, obj); err != nil {
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToWaitForPreconditions, err.Error())
		meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionPreconditions, err))
		return ctrl.Result{}, fmt.Errorf("unable to resolve references: %w", err)
	} else if requeue {
		r.Recorder.Event(obj, corev1.EventTypeNormal, eventWaitingForPreconditions, "waiting for referenced resources to be ready")
		return ctrl.Result{RequeueAfter: requeueTimeout}, nil
	}

	avnGen, err := r.newAivenClient(ctx, obj)
	if err != nil {
		return ctrl.Result{}, err
	}
	controller := r.newController(avnGen)

	if isMarkedForDeletion(obj) {
		return r.finalize(ctx, controller, obj)
	}

	orig := obj.DeepCopyObject().(v1alpha1.AivenManagedObject)
	defer func() {
		err = errors.Join(err, r.updateStatus(ctx, orig, obj))
	}()

	if controllerutil.AddFinalizer(obj, instanceDeletionFinalizer) {
		logr.FromContextOrDiscard(ctx).Info("added finalizer to instance")
		r.Recorder.Event(obj, corev1.EventTypeNormal, eventAddedFinalizer, "instance finalizer added")
	}

	meta.SetStatusCondition(obj.Conditions(), getInitializedCondition("Preconditions", "Checking preconditions"))
	obs, err := controller.Observe(ctx, obj)
	if err != nil {
		return r.handleObserveError(ctx, obj, err)
	}

	if !obs.ResourceExists {
		return r.createResource(ctx, controller, obj)
	}

	if !obs.ResourceUpToDate {
		return r.updateResource(ctx, controller, obj)
	}

	if err := r.publishSecretDetails(ctx, obj, obs.SecretDetails); err != nil {
		return ctrl.Result{}, err
	}

	return r.completeReconcileSuccess(obj)
}

func (r *Reconciler[T]) handleObserveError(ctx context.Context, obj T, err error) (ctrl.Result, error) {
	if errors.Is(err, errServicePoweredOff) {
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToWaitForPreconditions, err.Error())
		meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionPreconditions, err))
		return ctrl.Result{RequeueAfter: pollInterval}, nil
	}

	if errors.Is(err, errPreconditionNotMet) {
		const msg = "preconditions are not met, requeue"
		r.Recorder.Event(obj, corev1.EventTypeNormal, eventPreconditionsNotMet, msg)
		logr.FromContextOrDiscard(ctx).V(1).Info(msg)
		return ctrl.Result{RequeueAfter: requeueTimeout}, nil
	}

	if isRetryableAivenError(err) {
		logr.FromContextOrDiscard(ctx).Info("retryable Aiven API error, requeue", "error", err)
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToWaitForPreconditions, err.Error())
		return ctrl.Result{RequeueAfter: requeueTimeout}, nil
	}

	r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToWaitForPreconditions, err.Error())
	meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionPreconditions, err))
	return ctrl.Result{}, fmt.Errorf("cannot observe the resource: %w", err)
}

// resolveK8sRefs ensures that all referenced Kubernetes resources exist and are ready
func (r *Reconciler[T]) resolveK8sRefs(ctx context.Context, obj T) (requeue bool, err error) {
	refObj, ok := any(obj).(interface {
		client.Object
		GetRefs() []*v1alpha1.ResourceReferenceObject
	})
	if !ok {
		return false, nil
	}

	refs := refObj.GetRefs()
	for _, ref := range refs {
		runtimeObj, err := r.Scheme.New(ref.GroupVersionKind)
		if err != nil {
			return false, fmt.Errorf("creating %s: %w", ref.GroupVersionKind, err)
		}

		dep, ok := runtimeObj.(client.Object)
		if !ok {
			return false, fmt.Errorf("gvk %s is not client.Object", ref.GroupVersionKind)
		}

		if err := r.Get(ctx, ref.NamespacedName, dep); err != nil {
			// Matching legacy behaviour: missing or not-yet-created refs trigger a soft requeue.
			logr.FromContextOrDiscard(ctx).V(1).Info("referenced resource is not yet available", "ref", ref.NamespacedName, "gvk", ref.GroupVersionKind, "error", err)
			return true, nil
		}

		if !IsReadyToUse(dep) {
			return true, nil
		}
	}

	logr.FromContextOrDiscard(ctx).V(1).Info("all referenced resources are ready")
	return false, nil
}

func (r *Reconciler[T]) updateStatus(ctx context.Context, orig v1alpha1.AivenManagedObject, obj v1alpha1.AivenManagedObject) error {
	if equality.Semantic.DeepEqual(orig, obj) {
		return nil
	}

	// Order matters.
	// First need to update the object, and then update the status.
	// So dependent resources won't see READY before it has been updated with new values

	// Now we can update the status
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// When updated, object status is vanished.
		// So we waste a copy for that,
		// while the original object must already have all the fields updated in runtime
		// Additionally, it gets the "latest version" to resolve optimistic concurrency control conflict
		latest := obj.DeepCopyObject().(client.Object)
		if err := r.Get(ctx, types.NamespacedName{Name: latest.GetName(), Namespace: latest.GetNamespace()}, latest); err != nil {
			return err
		}

		updated := obj.DeepCopyObject().(client.Object)
		updated.SetResourceVersion(latest.GetResourceVersion())
		if err := r.Update(ctx, updated); err != nil {
			return err
		}

		obj.SetResourceVersion(updated.GetResourceVersion())
		return r.Status().Update(ctx, obj)
	})
}

func (r *Reconciler[T]) newAivenClient(ctx context.Context, obj T) (avngen.Client, error) {
	token, err := r.resolveToken(ctx, obj)
	if err != nil {
		return nil, err
	}

	avnGen, err := r.newAivenGeneratedClient(token, r.KubeVersion, r.OperatorVersion)
	if err != nil {
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToCreateClient, err.Error())
		return nil, fmt.Errorf("cannot initialize aiven generated client: %w", err)
	}

	return avnGen, nil
}

func (r *Reconciler[T]) resolveToken(ctx context.Context, obj T) (string, error) {
	if r.DefaultToken != "" {
		return r.DefaultToken, nil
	}

	auth := obj.AuthSecretRef()
	if auth == nil {
		return "", errNoTokenProvided
	}

	clientAuthSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: auth.Name, Namespace: obj.GetNamespace()}, clientAuthSecret); err != nil {
		r.Recorder.Eventf(obj, corev1.EventTypeWarning, eventUnableToGetAuthSecret, err.Error())
		return "", fmt.Errorf("cannot get secret %q: %w", auth.Name, err)
	}

	return string(clientAuthSecret.Data[auth.Key]), nil
}

func (r *Reconciler[T]) createResource(ctx context.Context, controller AivenController[T], obj T) (ctrl.Result, error) {
	r.Recorder.Event(obj, corev1.EventTypeNormal, eventCreateOrUpdatedAtAiven, "about to create instance at aiven")

	res, err := controller.Create(ctx, obj)
	if err != nil {
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
		meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionCreateOrUpdate, err))
		return ctrl.Result{}, fmt.Errorf("unable to create or update instance at aiven: %w", err)
	}
	r.Recorder.Event(obj, corev1.EventTypeNormal, eventCreatedOrUpdatedAtAiven, "instance was created at aiven but may not be running yet")

	if err := r.publishSecretDetails(ctx, obj, res.SecretDetails); err != nil {
		return ctrl.Result{}, err
	}

	return r.completeReconcileSuccess(obj)
}

func (r *Reconciler[T]) updateResource(ctx context.Context, controller AivenController[T], obj T) (ctrl.Result, error) {
	r.Recorder.Event(obj, corev1.EventTypeNormal, eventWaitingForTheInstanceToBeRunning, "waiting for the instance to be running")

	res, err := controller.Update(ctx, obj)
	if err != nil {
		if isNotFound(err) {
			return ctrl.Result{RequeueAfter: requeueTimeout}, nil
		}

		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToWaitForInstanceToBeRunning, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to wait until instance is running: %w", err)
	}

	if err := r.publishSecretDetails(ctx, obj, res.SecretDetails); err != nil {
		return ctrl.Result{}, err
	}

	return r.completeReconcileSuccess(obj)
}

func (r *Reconciler[T]) completeReconcileSuccess(obj v1alpha1.AivenManagedObject) (ctrl.Result, error) {
	metav1.SetMetaDataAnnotation(
		obj.GetObjectMeta(),
		processedGenerationAnnotation,
		strconv.FormatInt(obj.GetGeneration(), formatIntBaseDecimal),
	)

	return ctrl.Result{RequeueAfter: pollInterval}, nil
}

// publishSecretDetails publishes connection details to the connection secret if present.
// It emits appropriate events when secret creation is disabled or when syncing fails.
func (r *Reconciler[T]) publishSecretDetails(ctx context.Context, obj T, details map[string]string) error {
	if len(details) == 0 {
		return nil
	}

	if obj.NoSecret() {
		r.Recorder.Event(obj, corev1.EventTypeNormal, eventConnInfoSecretCreationDisabled,
			"connection info secret creation is disabled for this resource, secret won't be changed (check spec.connInfoSecretTargetDisabled)")
		return nil
	}

	withSecret, ok := any(obj).(objWithSecret)
	if !ok {
		logr.FromContextOrDiscard(ctx).Info(
			"object does not implement conn info secret target, skipping connection secret publish",
			"kind", obj.GetObjectKind().GroupVersionKind().String(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
		)
		return nil
	}

	goalSecret := r.newSecret(withSecret, details, false)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      goalSecret.Name,
			Namespace: goalSecret.Namespace,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if len(goalSecret.Data) > 0 {
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}
			maps.Copy(secret.Data, goalSecret.Data)
		}
		if goalSecret.StringData != nil {
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}
			for key, value := range goalSecret.StringData {
				secret.Data[key] = []byte(value)
			}
		}

		secret.Labels = goalSecret.Labels
		secret.Annotations = goalSecret.Annotations

		return controllerutil.SetControllerReference(obj, secret, r.Scheme)
	}); err != nil {
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventCannotPublishConnectionDetails, err.Error())
		meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionConnInfoSecret, err))
		return fmt.Errorf("unable to sync connection secret: %w", err)
	}

	return nil
}

func (r *Reconciler[T]) finalize(ctx context.Context, controller AivenController[T], obj T) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(obj, instanceDeletionFinalizer) {
		return ctrl.Result{}, nil
	}

	// Parse the annotations for the deletion policy. For simplicity, we only allow 'Orphan'.
	// If set will skip the deletion of the remote object. Disable by removing the annotation.
	if p, ok := obj.GetAnnotations()[deletionPolicyAnnotation]; !ok {
		r.Recorder.Event(obj, corev1.EventTypeNormal, eventTryingToDeleteAtAiven, "trying to delete instance at aiven")
		if err := controller.Delete(ctx, obj); err != nil {
			if isInvalidTokenError(err) && !hasIsRunningAnnotation(obj) {
				logr.FromContextOrDiscard(ctx).Info("invalid token error on deletion, removing finalizer", "apiError", err)
			} else {
				return r.handleDeleteError(ctx, obj, err)
			}
		}

		logr.FromContextOrDiscard(ctx).Info("instance was successfully deleted at Aiven, removing finalizer")
		r.Recorder.Event(obj, corev1.EventTypeNormal, eventSuccessfullyDeletedAtAiven, "instance is gone at aiven now")
	} else if ok && p == deletionPolicyOrphan {
		logr.FromContextOrDiscard(ctx).Info("finalizing with Orphan deletion policy - Aiven resource will be preserved on Kubernetes resource deletion")
	} else {
		msg := fmt.Sprintf("invalid deletion policy %q, only %q is allowed", p, deletionPolicyOrphan)
		meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionDelete, errors.New(msg)))
		return ctrl.Result{}, fmt.Errorf("unable to delete instance: %s", msg)
	}

	// remove finalizer, once all finalizers have been removed, the object will be deleted.
	if err := removeFinalizer(ctx, r.Client, obj, instanceDeletionFinalizer); err != nil {
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToDeleteFinalizer, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to remove finalizer: %w", err)
	}

	logr.FromContextOrDiscard(ctx).Info("finalizer was removed, resource is deleted")
	return ctrl.Result{}, nil
}

// handleDeleteError handles errors returned from Delete during finalization.
func (r *Reconciler[T]) handleDeleteError(ctx context.Context, obj T, err error) (ctrl.Result, error) {
	meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionDelete, err))

	// There are dependencies on Aiven side.
	// We keep the finalizer and trigger a soft requeue so that reconciliation can
	// be retried once dependencies are removed, but do not surface this as a hard error.
	if errors.Is(err, v1alpha1.ErrDeleteDependencies) {
		logr.FromContextOrDiscard(ctx).Info("object has dependencies, requeue delete", "apiError", err)
		return ctrl.Result{RequeueAfter: requeueTimeout}, nil
	}

	// If the deletion failed, don't remove the finalizer so that we can retry during the next reconciliation.
	switch {
	case isNotFound(err):
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToDeleteAtAiven, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to delete instance at aiven: %w", err)
	case isServerError(err):
		// If failed to delete due to a transient server error, keep the finalizer
		// and trigger a soft requeue so that deletion can be retried.
		logr.FromContextOrDiscard(ctx).Info("unable to delete instance at Aiven, will requeue delete", "apiError", err)
		return ctrl.Result{RequeueAfter: requeueTimeout}, nil
	default:
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToDelete, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to delete instance: %w", err)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler[T]) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.newObj()).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// isInvalidTokenError checks if the error is related to invalid token
func isInvalidTokenError(err error) bool {
	// When an instance was created but pointing to an invalid API token
	// and no generation was ever processed, allow deleting such instance
	msg := err.Error()
	return strings.Contains(msg, "Invalid token") || strings.Contains(msg, "Missing (expired) db token")
}
