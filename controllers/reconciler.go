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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func newManagedReconciler[
	T any,
	Obj interface {
		*T
		v1alpha1.AivenManagedObject
	},
](
	c Controller,
	newController func(c Controller, avnGen avngen.Client) AivenController[Obj],
	options *controller.Options,
) *Reconciler[Obj] {
	r := &Reconciler[Obj]{
		Controller:              c,
		newAivenGeneratedClient: NewAivenGeneratedClient,
		newObj: func() Obj {
			return new(T)
		},
		newController: func(avnGen avngen.Client) AivenController[Obj] {
			return newController(c, avnGen)
		},
		newSecret: newSecret,
		options:   options,
	}

	return r
}

// Reconciler handles the boilerplate reconciliation logic for Aiven resources.
//
// It orchestrates the ExternalClient lifecycle and shared status/metadata persistence.
type Reconciler[T v1alpha1.AivenManagedObject] struct {
	Controller
	newAivenGeneratedClient func(token, kubeVersion, operatorVersion string) (avngen.Client, error)
	newController           func(avnGen avngen.Client) AivenController[T]
	newObj                  func() T
	newSecret               func(o objWithSecret, stringData map[string]string, addPrefix bool) *corev1.Secret
	options                 *controller.Options
}

// requeueTimeout sets timeout to requeue controller
const requeueTimeout = 10 * time.Second

// Reconcile performs one full reconcile cycle.
//
// The reconciliation process:
// 1. Load latest object state and scope logs to this resource.
// 2. Prioritize deletion so dependency checks never block cleanup.
// 3. Gate create/update until referenced resources exist and are ready.
// 4. Protect per-resource auth secret from early deletion.
// 5. Initialize the external client used by Observe/Create/Update.
// 6. Persist finalizer before any external mutation path.
// 7. Observe remote state to decide flow and collect connection details.
// 8. If the remote resource is missing, continue with creation.
// 9. If remote state is stale, continue with update.
// 10. Persist metadata side effects from Observe, even on no-op reconcile.
// 11. Sync Kubernetes Secret with current remote connection details.
// 12. Report successful sync and schedule the next poll.
func (r *Reconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	defer func() {
		if apierrors.IsConflict(err) {
			res, err = ctrl.Result{Requeue: true}, nil
		}
	}()

	// 1. Load latest object state and scope logs to this resource.
	obj := r.newObj()
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	ctx = logr.NewContext(ctx, setupLogger(r.Log, obj))

	// 2. Prioritize deletion so dependency checks never block cleanup.
	if isMarkedForDeletion(obj) {
		return r.finalize(ctx, obj)
	}

	// 3. Gate create/update until referenced resources exist and are ready.
	if requeue, err := r.resolveK8sRefs(ctx, obj); err != nil {
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToWaitForPreconditions, err.Error())
		meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionPreconditions, err))
		r.setSyncedErrorCondition(obj, err)
		return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
	} else if requeue {
		err := errors.New("waiting for referenced resources to be ready")
		r.Recorder.Event(obj, corev1.EventTypeNormal, eventWaitingForPreconditions, err.Error())
		r.setSyncedErrorCondition(obj, err)
		return ctrl.Result{RequeueAfter: requeueTimeout}, r.updateStatus(ctx, obj)
	}

	// 4. Protect per-resource auth secret from early deletion before external operations.
	if err := r.ensureAuthSecretFinalizer(ctx, obj); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		r.setSyncedErrorCondition(obj, err)
		return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
	}

	// 5. Initialize the external client used by Observe/Create/Update.
	avnGen, err := r.newAivenClient(ctx, obj)
	if err != nil {
		r.setSyncedErrorCondition(obj, err)
		return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
	}
	controller := r.newController(avnGen)

	// 6. Persist finalizer before any external mutation path.
	if controllerutil.AddFinalizer(obj, instanceDeletionFinalizer) {
		logr.FromContextOrDiscard(ctx).Info("added finalizer to instance")
		r.Recorder.Event(obj, corev1.EventTypeNormal, eventAddedFinalizer, "instance finalizer added")
		if err := r.updateObject(ctx, obj); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
	}

	annotationsBeforeObserve := maps.Clone(obj.GetAnnotations())

	// 7. Observe remote state to decide flow and collect connection details.
	meta.SetStatusCondition(obj.Conditions(), getInitializedCondition("Preconditions", "Checking preconditions"))
	obs, err := controller.Observe(ctx, obj)
	if err != nil {
		return r.handleObserveError(ctx, obj, err)
	}

	// 8. If the remote resource is missing, continue with creation.
	if !obs.ResourceExists {
		return r.createResource(ctx, controller, obj)
	}

	// 9. If remote state is stale, continue with update.
	if !obs.ResourceUpToDate {
		return r.updateResource(ctx, controller, obj)
	}

	// 10. Persist metadata side effects from Observe, even on no-op reconcile.
	if !maps.Equal(annotationsBeforeObserve, obj.GetAnnotations()) {
		if err := r.updateObject(ctx, obj); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
	}

	// 11. Sync Kubernetes Secret with current remote connection details.
	if err := r.publishSecretDetails(ctx, obj, obs.SecretDetails); err != nil {
		r.setSyncedErrorCondition(obj, err)
		return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
	}

	// 12. Report successful sync and schedule the next poll.
	r.setSyncedSuccessCondition(obj)
	return ctrl.Result{RequeueAfter: r.PollInterval}, r.updateStatus(ctx, obj)
}

func (r *Reconciler[T]) handleObserveError(ctx context.Context, obj T, err error) (ctrl.Result, error) {
	var requeueNeeded ErrRequeueNeeded
	if errors.As(err, &requeueNeeded) {
		r.Recorder.Event(obj, corev1.EventTypeNormal, eventWaitingForPreconditions, requeueNeeded.OriginalError.Error())
		r.setSyncedErrorCondition(obj, requeueNeeded.OriginalError)
		return ctrl.Result{RequeueAfter: requeueTimeout}, r.updateStatus(ctx, obj)
	}

	r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToObserve, err.Error())
	meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionObserve, err))
	r.setSyncedErrorCondition(obj, err)
	return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
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
			if apierrors.IsNotFound(err) {
				// Matching legacy behaviour: missing or not-yet-created refs trigger a soft requeue.
				logr.FromContextOrDiscard(ctx).V(1).Info("referenced resource is not yet available", "ref", ref.NamespacedName, "gvk", ref.GroupVersionKind, "error", err)
				return true, nil
			}

			return false, fmt.Errorf("getting referenced resource %s %s: %w", ref.GroupVersionKind, ref.NamespacedName, err)
		}

		if !IsReadyToUse(dep) {
			return true, nil
		}
	}

	logr.FromContextOrDiscard(ctx).V(1).Info("all referenced resources are ready")
	return false, nil
}

func (r *Reconciler[T]) updateStatus(ctx context.Context, obj T) error {
	if err := r.Status().Update(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) && isMarkedForDeletion(obj) {
			return nil
		}
		return err
	}

	return nil
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

func (r *Reconciler[T]) ensureAuthSecretFinalizer(ctx context.Context, obj T) error {
	if r.DefaultToken != "" {
		return nil
	}

	auth := obj.AuthSecretRef()
	if auth == nil {
		return nil
	}

	clientAuthSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: auth.Name, Namespace: obj.GetNamespace()}, clientAuthSecret); err == nil {
		if controllerutil.ContainsFinalizer(clientAuthSecret, secretProtectionFinalizer) {
			return nil
		}

		// We cannot add new finalizers to a Secret that is already terminating.
		// During managed resource deletion proceed best-effort to avoid getting stuck.
		if isMarkedForDeletion(obj) && isMarkedForDeletion(clientAuthSecret) {
			return nil
		}

		if err := addFinalizer(ctx, r.Client, clientAuthSecret, secretProtectionFinalizer); err != nil {
			if isMarkedForDeletion(obj) {
				logr.FromContextOrDiscard(ctx).V(1).Info("unable to protect auth secret during deletion, proceeding", "error", err)
			} else {
				return fmt.Errorf("unable to add finalizer to secret: %w", err)
			}
		}
	}

	// resolveToken emits user-facing auth errors; keep this step best-effort.
	return nil
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

func (r *Reconciler[T]) updateObject(ctx context.Context, obj T) error {
	// Update a deep copy to avoid clobbering in-memory status changes with the API server response.
	updated := obj.DeepCopyObject().(client.Object)
	if err := r.Update(ctx, updated); err != nil {
		if apierrors.IsNotFound(err) && isMarkedForDeletion(obj) {
			return nil
		}
		return err
	}
	obj.SetResourceVersion(updated.GetResourceVersion())
	return nil
}

func (r *Reconciler[T]) createResource(ctx context.Context, controller AivenController[T], obj T) (ctrl.Result, error) {
	meta.SetStatusCondition(obj.Conditions(), getInitializedCondition("Creating", "creating resource at Aiven"))
	r.Recorder.Event(obj, corev1.EventTypeNormal, eventCreateOrUpdatedAtAiven, "about to create instance at aiven")

	annotationsBeforeCreate := maps.Clone(obj.GetAnnotations())

	res, err := controller.Create(ctx, obj)
	if err != nil {
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
		meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionCreateOrUpdate, err))
		r.setSyncedErrorCondition(obj, err)
		return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
	}

	if !maps.Equal(annotationsBeforeCreate, obj.GetAnnotations()) {
		if err := r.updateObject(ctx, obj); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
			meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionCreateOrUpdate, err))
			r.setSyncedErrorCondition(obj, err)
			return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
		}
	}

	if err := r.publishSecretDetails(ctx, obj, res.SecretDetails); err != nil {
		r.setSyncedErrorCondition(obj, err)
		return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
	}

	meta.SetStatusCondition(obj.Conditions(), getInitializedCondition("Creating", "creation requested at Aiven"))
	r.setSyncedSuccessCondition(obj)
	r.Recorder.Event(obj, corev1.EventTypeNormal, eventCreatedOrUpdatedAtAiven, "instance was created at aiven but may not be running yet")
	return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
}

func (r *Reconciler[T]) updateResource(ctx context.Context, controller AivenController[T], obj T) (ctrl.Result, error) {
	r.Recorder.Event(obj, corev1.EventTypeNormal, eventWaitingForTheInstanceToBeRunning, "waiting for the instance to be running")

	res, err := controller.Update(ctx, obj)
	if err != nil {
		if isNotFound(err) {
			// Keep this path status-only no-op: we intentionally don't persist status on transient update misses.
			return ctrl.Result{RequeueAfter: requeueTimeout}, nil
		}

		// Align with main/legacy behaviour: return reconcile error and let backoff handle retries.
		// We don't persist status here unless this branch starts mutating conditions in the future.
		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToWaitForInstanceToBeRunning, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to wait until instance is running: %w", err)
	}

	if err := r.publishSecretDetails(ctx, obj, res.SecretDetails); err != nil {
		r.setSyncedErrorCondition(obj, err)
		return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
	}

	metav1.SetMetaDataAnnotation(
		obj.GetObjectMeta(),
		processedGenerationAnnotation,
		strconv.FormatInt(obj.GetGeneration(), formatIntBaseDecimal),
	)
	if err := r.updateObject(ctx, obj); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}

		r.Recorder.Event(obj, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
		meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionCreateOrUpdate, err))
		r.setSyncedErrorCondition(obj, err)
		return ctrl.Result{Requeue: true}, r.updateStatus(ctx, obj)
	}

	if IsReadyToUse(obj) {
		r.setSyncedSuccessCondition(obj)
		return ctrl.Result{RequeueAfter: r.PollInterval}, r.updateStatus(ctx, obj)
	}

	// Many Aiven operations are asynchronous. After a successful API call the resource may still be
	// starting up, so requeue soon to check again instead of waiting for the normal periodic reconcile.
	r.setSyncedSuccessCondition(obj)
	return ctrl.Result{RequeueAfter: requeueTimeout}, r.updateStatus(ctx, obj)
}

func (r *Reconciler[T]) setSyncedErrorCondition(obj T, err error) {
	meta.SetStatusCondition(obj.Conditions(), getSyncedErrorCondition(err))
}

func (r *Reconciler[T]) setSyncedSuccessCondition(obj T) {
	meta.SetStatusCondition(obj.Conditions(), getSyncedSuccessCondition())
	meta.RemoveStatusCondition(obj.Conditions(), ConditionTypeError)
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

func (r *Reconciler[T]) finalize(ctx context.Context, obj T) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(obj, instanceDeletionFinalizer) {
		return ctrl.Result{}, nil
	}

	// Parse the annotations for the deletion policy. For simplicity, we only allow 'Orphan'.
	// If set will skip the deletion of the remote object. Disable by removing the annotation.
	if p, ok := obj.GetAnnotations()[deletionPolicyAnnotation]; ok {
		if p == deletionPolicyOrphan {
			logr.FromContextOrDiscard(ctx).Info("finalizing with Orphan deletion policy - Aiven resource will be preserved on Kubernetes resource deletion")
		} else {
			msg := fmt.Sprintf("invalid deletion policy %q, only %q is allowed", p, deletionPolicyOrphan)
			meta.SetStatusCondition(obj.Conditions(), getErrorCondition(errConditionDelete, errors.New(msg)))
			// This is a delete-path reconcile. Persisting the condition must not depend on external deps.
			// We also don't have the general persistence defer on the delete path.
			if err := r.updateStatus(ctx, obj); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, fmt.Errorf("unable to delete instance: %s", msg)
		}
	} else {
		if err := r.ensureAuthSecretFinalizer(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}

		avnGen, err := r.newAivenClient(ctx, obj)
		if err != nil {
			return ctrl.Result{}, err
		}
		controller := r.newController(avnGen)

		r.Recorder.Event(obj, corev1.EventTypeNormal, eventTryingToDeleteAtAiven, "trying to delete instance at aiven")
		if err := controller.Delete(ctx, obj); err != nil {
			if isInvalidTokenError(err) && !hasIsRunningAnnotation(obj) {
				logr.FromContextOrDiscard(ctx).Info("invalid token error on deletion, removing finalizer", "apiError", err)
			} else {
				res, err := r.handleDeleteError(ctx, obj, err)
				// This is a delete-path reconcile. Persisting the condition must not depend on external deps.
				// We also don't have the general persistence defer on the delete path.
				if err := r.updateStatus(ctx, obj); err != nil {
					return ctrl.Result{}, err
				}
				return res, err
			}
		}

		logr.FromContextOrDiscard(ctx).Info("instance was successfully deleted at Aiven, removing finalizer")
		r.Recorder.Event(obj, corev1.EventTypeNormal, eventSuccessfullyDeletedAtAiven, "instance is gone at aiven now")
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
	obj := r.newObj()
	b := ctrl.NewControllerManagedBy(mgr).For(obj, builder.WithPredicates(predicate.Or(
		deletionTimestampChanged(),
		predicate.GenerationChangedPredicate{},
		predicate.LabelChangedPredicate{},
		annotationsChangedExcluding(
			processedGenerationAnnotation,
			instanceIsRunningAnnotation,
		))))
	if _, ok := any(obj).(objWithSecret); ok {
		b = b.Owns(&corev1.Secret{})
	}

	if r.options != nil {
		b = b.WithOptions(*r.options)
	}

	return b.Complete(r)
}

// isInvalidTokenError checks if the error is related to invalid token
func isInvalidTokenError(err error) bool {
	// When an instance was created but pointing to an invalid API token
	// and no generation was ever processed, allow deleting such instance
	msg := err.Error()
	return strings.Contains(msg, "Invalid token") || strings.Contains(msg, "Missing (expired) db token")
}

func deletionTimestampChanged() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}

			oldDT := e.ObjectOld.GetDeletionTimestamp()
			newDT := e.ObjectNew.GetDeletionTimestamp()
			switch {
			case oldDT == nil && newDT == nil:
				return false
			case oldDT == nil || newDT == nil:
				return true
			default:
				return !oldDT.Time.Equal(newDT.Time)
			}
		},
	}
}

func annotationsChangedExcluding(keys ...string) predicate.Funcs {
	ignored := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		ignored[key] = struct{}{}
	}

	subsetEq := func(a, b map[string]string) bool {
		for k, v := range a {
			if _, skip := ignored[k]; skip {
				continue
			}

			bv, ok := b[k]
			if !ok || bv != v {
				return false
			}
		}
		return true
	}

	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}

			oldAnnotations := e.ObjectOld.GetAnnotations()
			newAnnotations := e.ObjectNew.GetAnnotations()
			return !subsetEq(oldAnnotations, newAnnotations) || !subsetEq(newAnnotations, oldAnnotations)
		},
	}
}
