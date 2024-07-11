package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// formatIntBaseDecimal it is a base to format int64 to string
const formatIntBaseDecimal = 10

// requeueTimeout sets timeout to requeue controller
const requeueTimeout = 10 * time.Second

var errNoTokenProvided = fmt.Errorf("authSecretReference is not set and no default token provided")

type (
	// Controller reconciles the Aiven objects
	Controller struct {
		client.Client

		Log             logr.Logger
		Scheme          *runtime.Scheme
		Recorder        record.EventRecorder
		DefaultToken    string
		AvGenClient     avngen.Client
		KubeVersion     string
		OperatorVersion string
	}

	// Handlers represents Aiven API handlers
	// It intended to be a layer between Kubernetes and Aiven API that handles all aspects
	// of the Aiven services lifecycle.
	Handlers interface {
		// create or updates an instance on the Aiven side.
		createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error

		// delete removes an instance on Aiven side.
		// If an object is already deleted and cannot be found, it should not be an error. For other deletion
		// errors, return an error.
		delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error)

		// get retrieve an object and a secret (for example, connection credentials) that is generated on the
		// fly based on data from Aiven API.  When not applicable to service, it should return nil.
		get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error)

		// checkPreconditions check whether all preconditions for creating (or updating) the resource are in place.
		// For example, it is applicable when a service needs to be running before this resource can be created.
		checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error)
	}

	// refsObject returns references to dependent resources
	refsObject interface {
		client.Object

		GetRefs() []*v1alpha1.ResourceReferenceObject
	}
)

const (
	// Lifecycle event types we expose to the user
	eventUnableToGetAuthSecret              = "UnableToGetAuthSecret"
	eventUnableToCreateClient               = "UnableToCreateClient"
	eventReconciliationStarted              = "ReconcilationStarted"
	eventTryingToDeleteAtAiven              = "TryingToDeleteAtAiven"
	eventUnableToDeleteAtAiven              = "UnableToDeleteAtAiven"
	eventUnableToDeleteFinalizer            = "UnableToDeleteFinalizer"
	eventUnableToDelete                     = "UnableToDelete"
	eventSuccessfullyDeletedAtAiven         = "SuccessfullyDeletedAtAiven"
	eventAddedFinalizer                     = "InstanceFinalizerAdded"
	eventWaitingForPreconditions            = "WaitingForPreconditions"
	eventUnableToWaitForPreconditions       = "UnableToWaitForPreconditions"
	eventPreconditionsAreMet                = "PreconditionsAreMet"
	eventPreconditionsNotMet                = "PreconditionsNotMet"
	eventUnableToCreateOrUpdateAtAiven      = "UnableToCreateOrUpdateAtAiven"
	eventCreateOrUpdatedAtAiven             = "CreateOrUpdatedAtAiven"
	eventCreatedOrUpdatedAtAiven            = "CreatedOrUpdatedAtAiven"
	eventWaitingForTheInstanceToBeRunning   = "WaitingForInstanceToBeRunning"
	eventUnableToWaitForInstanceToBeRunning = "UnableToWaitForInstanceToBeRunning"
	eventInstanceIsRunning                  = "InstanceIsRunning"
	eventConnInfoSecretCreationDisabled     = "ConnInfoSecretCreationDisabled"
)

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (c *Controller) reconcileInstance(ctx context.Context, req ctrl.Request, h Handlers, o v1alpha1.AivenManagedObject) (ctrl.Result, error) {
	if err := c.Get(ctx, req.NamespacedName, o); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	instanceLogger := setupLogger(c.Log, o)
	instanceLogger.Info("setting up aiven client with instance secret")

	var token string
	var clientAuthSecret *corev1.Secret
	if len(c.DefaultToken) > 0 {
		token = c.DefaultToken
	} else if auth := o.AuthSecretRef(); auth != nil {
		clientAuthSecret = &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{Name: auth.Name, Namespace: req.Namespace}, clientAuthSecret); err != nil {
			c.Recorder.Eventf(o, corev1.EventTypeWarning, eventUnableToGetAuthSecret, err.Error())
			return ctrl.Result{}, fmt.Errorf("cannot get secret %q: %w", auth.Name, err)
		}
		token = string(clientAuthSecret.Data[auth.Key])
	} else {
		return ctrl.Result{}, errNoTokenProvided
	}

	avn, err := NewAivenClient(token, c.KubeVersion, c.OperatorVersion)
	if err != nil {
		c.Recorder.Event(o, corev1.EventTypeWarning, eventUnableToCreateClient, err.Error())
		return ctrl.Result{}, fmt.Errorf("cannot initialize aiven client: %w", err)
	}

	avnGen, err := NewAivenGeneratedClient(token, c.KubeVersion, c.OperatorVersion)
	if err != nil {
		c.Recorder.Event(o, corev1.EventTypeWarning, eventUnableToCreateClient, err.Error())
		return ctrl.Result{}, fmt.Errorf("cannot initialize aiven generated client: %w", err)
	}

	helper := instanceReconcilerHelper{
		avn:    avn,
		avnGen: avnGen,
		k8s:    c.Client,
		h:      h,
		log:    instanceLogger,
		s:      clientAuthSecret,
		rec:    c.Recorder,
	}

	requeue, err := helper.reconcile(ctx, o)
	result := ctrl.Result{Requeue: requeue}
	if requeue {
		result.RequeueAfter = requeueTimeout
	}
	return result, err
}

// a helper that closes over all instance specific fields
// to make reconciliation a little more ergonomic
type instanceReconcilerHelper struct {
	k8s client.Client

	// avn, Aiven client that is authorized with the instance token
	avn *aiven.Client

	// avnGen, Aiven client that is authorized with the instance token
	avnGen avngen.Client

	// h, instance specific handler implementation
	h Handlers

	// s, secret that contains the aiven token for the instance
	s *corev1.Secret

	// log, logger setup with structured fields for the instance
	log logr.Logger

	// rec, recorder to record events for the object
	rec record.EventRecorder
}

func (i *instanceReconcilerHelper) reconcile(ctx context.Context, o v1alpha1.AivenManagedObject) (bool, error) {
	// Deletion
	if isMarkedForDeletion(o) {
		if controllerutil.ContainsFinalizer(o, instanceDeletionFinalizer) {
			return i.finalize(ctx, o)
		}
		return false, nil
	}

	if isAlreadyProcessed(o) && IsAlreadyRunning(o) {
		return false, nil
	}

	// Create or update.
	// Even if reconcile fails, we need to update the object in kube
	// to save conditions and other data.
	// So we don't exit on error.
	orig := o.DeepCopyObject().(v1alpha1.AivenManagedObject)
	requeue, err := i.reconcileInstance(ctx, o)
	if equality.Semantic.DeepEqual(orig, o) {
		return requeue, err
	}

	// Order matters.
	// First need to update the object, and then update the status.
	// So dependent resources won't see READY before it has been updated with new values

	// Now we can update the status
	errUpdate := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// When updated, object status is vanished.
		// So we waste a copy for that,
		// while the original object must already have all the fields updated in runtime
		// Additionally, it gets the "latest version" to resolve optimistic concurrency control conflict
		latest := o.DeepCopyObject().(client.Object)
		err = i.k8s.Get(ctx, types.NamespacedName{
			Name:      latest.GetName(),
			Namespace: latest.GetNamespace(),
		}, latest)
		if err != nil {
			return err
		}

		updated := o.DeepCopyObject().(client.Object)
		updated.SetResourceVersion(latest.GetResourceVersion())
		err := i.k8s.Update(ctx, updated)
		if err != nil {
			return err
		}

		o.SetResourceVersion(updated.GetResourceVersion())
		return i.k8s.Status().Update(ctx, o)
	})

	errMerged := errors.Join(err, errUpdate)
	return requeue || errMerged != nil, errMerged
}

func (i *instanceReconcilerHelper) reconcileInstance(ctx context.Context, o v1alpha1.AivenManagedObject) (bool, error) {
	i.log.Info("reconciling instance")
	i.rec.Event(o, corev1.EventTypeNormal, eventReconciliationStarted, "starting reconciliation")

	// Add finalizers to an instance and associated secret, only if they haven't
	// been added in the previous reconciliation loops
	if i.s != nil {
		if !controllerutil.ContainsFinalizer(i.s, secretProtectionFinalizer) {
			i.log.Info("adding finalizer to secret")
			if err := addFinalizer(ctx, i.k8s, i.s, secretProtectionFinalizer); err != nil {
				return false, fmt.Errorf("unable to add finalizer to secret: %w", err)
			}
		}
	}

	if !controllerutil.ContainsFinalizer(o, instanceDeletionFinalizer) {
		// Adds finalizer. The commit is performed in the outer function
		i.log.Info("adding finalizer to instance")
		controllerutil.AddFinalizer(o, instanceDeletionFinalizer)
		i.rec.Event(o, corev1.EventTypeNormal, eventAddedFinalizer, "instance finalizer added")
	}

	// check instance preconditions, if not met - requeue
	i.log.Info("handling service update/creation")
	refs, err := i.getObjectRefs(ctx, o)
	if err != nil {
		i.log.Info(fmt.Sprintf("one or more references can't be found yet: %s", err))
		return true, nil
	}

	requeue, err := i.checkPreconditions(ctx, o, refs)
	if requeue {
		return true, err
	}

	if err != nil {
		meta.SetStatusCondition(o.Conditions(), getErrorCondition(errConditionPreconditions, err))
		return false, err
	}

	if !isAlreadyProcessed(o) {
		i.rec.Event(o, corev1.EventTypeNormal, eventCreateOrUpdatedAtAiven, "about to create instance at aiven")
		if err := i.createOrUpdateInstance(ctx, o, refs); err != nil {
			i.rec.Event(o, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
			return false, fmt.Errorf("unable to create or update instance at aiven: %w", err)
		}

		i.rec.Event(o, corev1.EventTypeNormal, eventCreatedOrUpdatedAtAiven, "instance was created at aiven but may not be running yet")
	}

	i.rec.Event(o, corev1.EventTypeNormal, eventWaitingForTheInstanceToBeRunning, "waiting for the instance to be running")
	err = i.updateInstanceStateAndSecretUntilRunning(ctx, o)
	if err != nil {
		if isNotFound(err) {
			return true, nil
		}

		i.rec.Event(o, corev1.EventTypeWarning, eventUnableToWaitForInstanceToBeRunning, err.Error())
		return false, fmt.Errorf("unable to wait until instance is running: %w", err)
	}

	if !IsAlreadyRunning(o) {
		i.log.Info("instance is not yet running, triggering requeue")
		return true, nil
	}

	i.rec.Event(o, corev1.EventTypeNormal, eventInstanceIsRunning, "instance is in a RUNNING state")
	i.log.Info("instance was successfully reconciled")
	return false, nil
}

func (i *instanceReconcilerHelper) checkPreconditions(ctx context.Context, o client.Object, refs []client.Object) (bool, error) {
	i.rec.Event(o, corev1.EventTypeNormal, eventWaitingForPreconditions, "waiting for preconditions of the instance")

	// Checks references
	if len(refs) > 0 {
		for _, r := range refs {
			if !(isAlreadyProcessed(r) && IsAlreadyRunning(r)) {
				i.log.Info("references are in progress")
				return true, nil
			}
		}
		i.log.Info("all references are good")
	}

	check, err := i.h.checkPreconditions(ctx, i.avn, i.avnGen, o)
	if err != nil {
		i.rec.Event(o, corev1.EventTypeWarning, eventUnableToWaitForPreconditions, err.Error())
		return false, fmt.Errorf("unable to wait for preconditions: %w", err)
	}

	if !check {
		i.rec.Event(o, corev1.EventTypeNormal, eventPreconditionsNotMet, "preconditions are not met, requeue")
		i.log.Info("preconditions are not met, requeue")
		return true, nil
	}

	i.rec.Event(o, corev1.EventTypeNormal, eventPreconditionsAreMet, "preconditions are met, proceeding to create or update")
	return false, nil
}

func (i *instanceReconcilerHelper) getObjectRefs(ctx context.Context, o client.Object) ([]client.Object, error) {
	refsObj, ok := o.(refsObject)
	if !ok {
		return nil, nil
	}

	refs := refsObj.GetRefs()
	if len(refs) == 0 {
		return nil, nil
	}

	schema := i.k8s.Scheme()
	objs := make([]client.Object, 0, len(refs))
	for _, r := range refs {
		runtimeObj, err := schema.New(r.GroupVersionKind)
		if err != nil {
			return nil, fmt.Errorf("unknown GroupVersionKind %s: %w", r.GroupVersionKind, err)
		}

		obj, ok := runtimeObj.(client.Object)
		if !ok {
			return nil, fmt.Errorf("gvk %s is not client.Object", r.GroupVersionKind)
		}

		err = i.k8s.Get(ctx, r.NamespacedName, obj)
		if err != nil {
			return nil, fmt.Errorf("cannot get client obj %+v: %w", r, err)
		}
		objs = append(objs, obj)
	}

	return objs, nil
}

// finalize runs finalization logic. If the finalization logic fails, don't remove the finalizer so
// that we can retry during the next reconciliation. When applicable, it retrieves an associated object that
// has to be deleted from Kubernetes, and it could be a secret associated with an instance.
func (i *instanceReconcilerHelper) finalize(ctx context.Context, o v1alpha1.AivenManagedObject) (bool, error) {
	i.rec.Event(o, corev1.EventTypeNormal, eventTryingToDeleteAtAiven, "trying to delete instance at aiven")

	var err error
	finalised := true
	deletionPolicy := deletionPolicyDelete

	// Parse the annotations for the deletion policy. For simplicity, we only allow 'Orphan'.
	// If set will skip the deletion of the remote object. Disable by removing the annotation.
	if p, ok := o.GetAnnotations()[deletionPolicyAnnotation]; ok {
		deletionPolicy = deletionPolicyOrphan
		if p != deletionPolicyOrphan {
			i.log.Info(fmt.Sprintf("Invalid deletion policy! Only '%s' is allowed.", deletionPolicyOrphan))
			finalised = false
		}
	}

	if deletionPolicy == deletionPolicyDelete {
		finalised, err = i.h.delete(ctx, i.avn, i.avnGen, o)
		if err != nil {
			meta.SetStatusCondition(o.Conditions(), getErrorCondition(errConditionDelete, err))
		}
	}

	// There are dependencies on Aiven side, resets error, so it goes for requeue
	// Handlers does not have logger, it goes here
	if errors.Is(err, v1alpha1.ErrDeleteDependencies) {
		i.log.Info("object has dependencies", "apiError", err)
		err = nil
	}

	// If the deletion failed, don't remove the finalizer so that we can retry during the next reconciliation.
	// Unless the error is invalid token and resource is not running, in that case we remove the finalizer
	// and let the instance be deleted.
	if err != nil {
		if i.isInvalidTokenError(err) && !IsAlreadyRunning(o) {
			i.log.Info("invalid token error on deletion, removing finalizer", "apiError", err)
			finalised = true
		} else if isNotFound(err) {
			i.rec.Event(o, corev1.EventTypeWarning, eventUnableToDeleteAtAiven, err.Error())
			return false, fmt.Errorf("unable to delete instance at aiven: %w", err)
		} else if isAivenServerError(err) {
			// If failed to delete, retries
			i.log.Info(fmt.Sprintf("unable to delete instance at aiven: %s", err))
			err = nil
		} else {
			i.rec.Event(o, corev1.EventTypeWarning, eventUnableToDelete, err.Error())
			return false, fmt.Errorf("unable to delete instance: %w", err)
		}
	}

	// checking if instance was finalized, if not triggering a requeue
	if !finalised {
		i.log.Info("instance is not yet deleted at aiven, triggering requeue")
		return true, nil
	}

	i.log.Info("instance was successfully deleted at aiven, removing finalizer")
	i.rec.Event(o, corev1.EventTypeNormal, eventSuccessfullyDeletedAtAiven, "instance is gone at aiven now")

	// remove finalizer, once all finalizers have been removed, the object will be deleted.
	if err := removeFinalizer(ctx, i.k8s, o, instanceDeletionFinalizer); err != nil {
		i.rec.Event(o, corev1.EventTypeWarning, eventUnableToDeleteFinalizer, err.Error())
		return false, fmt.Errorf("unable to remove finalizer: %w", err)
	}

	i.log.Info("finalizer was removed, instance is deleted")
	return false, nil
}

// isInvalidTokenError checks if the error is related to invalid token
func (i *instanceReconcilerHelper) isInvalidTokenError(err error) bool {
	// When an instance was created but pointing to an invalid API token
	// and no generation was ever processed, allow deleting such instance
	msg := err.Error()
	return strings.Contains(msg, "Invalid token") || strings.Contains(msg, "Missing (expired) db token")
}

func (i *instanceReconcilerHelper) createOrUpdateInstance(ctx context.Context, o v1alpha1.AivenManagedObject, refs []client.Object) error {
	i.log.Info("generation wasn't processed, creation or updating instance on aiven side")
	a := o.GetAnnotations()
	delete(a, processedGenerationAnnotation)
	delete(a, instanceIsRunningAnnotation)

	if err := i.h.createOrUpdate(ctx, i.avn, i.avnGen, o, refs); err != nil {
		meta.SetStatusCondition(o.Conditions(), getErrorCondition(errConditionCreateOrUpdate, err))
		return fmt.Errorf("unable to create or update aiven instance: %w", err)
	}

	i.log.Info(
		"processed instance, updating annotations",
		"generation", o.GetGeneration(),
		"annotations", o.GetAnnotations(),
	)
	return nil
}

func (i *instanceReconcilerHelper) updateInstanceStateAndSecretUntilRunning(ctx context.Context, o v1alpha1.AivenManagedObject) error {
	i.log.Info("checking if instance is ready")

	// Needs to be before o.NoSecret() check because `get` mutates the object's metadata annotations.
	// It set the instanceIsRunningAnnotation annotation when the instance is running on Aiven's side.
	secret, err := i.h.get(ctx, i.avn, i.avnGen, o)
	if secret == nil || err != nil {
		return err
	}

	if o.NoSecret() {
		i.rec.Event(o, corev1.EventTypeNormal, eventConnInfoSecretCreationDisabled, "connInfoSecretTargetDisabled is true, secret will not be created")
		return nil
	}

	_, err = controllerutil.CreateOrUpdate(ctx, i.k8s, secret, func() error {
		return controllerutil.SetControllerReference(o, secret, i.k8s.Scheme())
	})

	return err
}

func setupLogger(log logr.Logger, o client.Object) logr.Logger {
	a := make(map[string]string)
	if r, ok := o.GetAnnotations()[instanceIsRunningAnnotation]; ok {
		a[instanceIsRunningAnnotation] = r
	}

	if g, ok := o.GetAnnotations()[processedGenerationAnnotation]; ok {
		a[processedGenerationAnnotation] = g
	}
	kind := strings.ToLower(o.GetObjectKind().GroupVersionKind().Kind)
	name := types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}

	return log.WithValues("kind", kind, "name", name, "annotations", a)
}

func toOptionalStringPointer(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

func getMaintenanceWindow(dow, time string) *aiven.MaintenanceWindow {
	if dow != "" || time != "" {
		return &aiven.MaintenanceWindow{
			DayOfWeek: dow,
			TimeOfDay: time,
		}
	}

	return nil
}
