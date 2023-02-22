package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/aiven/aiven-go-client"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/liip/sheriff"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
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

		Log          logr.Logger
		Scheme       *runtime.Scheme
		Recorder     record.EventRecorder
		DefaultToken string
	}

	// Handlers represents Aiven API handlers
	// It intended to be a layer between Kubernetes and Aiven API that handles all aspects
	// of the Aiven services lifecycle.
	Handlers interface {
		// create or updates an instance on the Aiven side.
		createOrUpdate(*aiven.Client, client.Object, []client.Object) error

		// delete removes an instance on Aiven side.
		// If an object is already deleted and cannot be found, it should not be an error. For other deletion
		// errors, return an error.
		delete(*aiven.Client, client.Object) (bool, error)

		// get retrieve an object and a secret (for example, connection credentials) that is generated on the
		// fly based on data from Aiven API.  When not applicable to service, it should return nil.
		get(*aiven.Client, client.Object) (*corev1.Secret, error)

		// checkPreconditions check whether all preconditions for creating (or updating) the resource are in place.
		// For example, it is applicable when a service needs to be running before this resource can be created.
		checkPreconditions(*aiven.Client, client.Object) (bool, error)
	}

	aivenManagedObject interface {
		client.Object

		AuthSecretRef() *v1alpha1.AuthSecretReference
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
	eventSuccessfullyDeletedAtAiven         = "SuccessfullyDeletedAtAiven"
	eventAddedFinalizer                     = "InstanceFinalizerAdded"
	eventUnableToAddFinalizer               = "UnableToAddFinalizer"
	eventWaitingForPreconditions            = "WaitingForPreconditions"
	eventUnableToWaitForPreconditions       = "UnableToWaitForPreconditions"
	eventPreconditionsAreMet                = "PreconditionsAreMet"
	eventUnableToCreateOrUpdateAtAiven      = "UnableToCreateOrUpdateAtAiven"
	eventCreateOrUpdatedAtAiven             = "CreateOrUpdatedAtAiven"
	eventCreatedOrUpdatedAtAiven            = "CreatedOrUpdatedAtAiven"
	eventWaitingForTheInstanceToBeRunning   = "WaitingForInstanceToBeRunning"
	eventUnableToWaitForInstanceToBeRunning = "UnableToWaitForInstanceToBeRunning"
	eventInstanceIsRunning                  = "InstanceIsRunning"
)

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (c *Controller) reconcileInstance(ctx context.Context, req ctrl.Request, h Handlers, o aivenManagedObject) (ctrl.Result, error) {
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

	avn, err := aiven.NewTokenClient(token, operatorUserAgent)
	if err != nil {
		c.Recorder.Event(o, corev1.EventTypeWarning, eventUnableToCreateClient, err.Error())
		return ctrl.Result{}, fmt.Errorf("cannot initialize aiven client: %w", err)
	}

	return instanceReconcilerHelper{
		avn: avn,
		k8s: c.Client,
		h:   h,
		log: instanceLogger,
		s:   clientAuthSecret,
		rec: c.Recorder,
	}.reconcileInstance(ctx, o)
}

// a helper that closes over all instance specific fields
// to make reconciliation a little more ergonomic
type instanceReconcilerHelper struct {
	k8s client.Client

	// avn, Aiven client that is authorized with the instance token
	avn *aiven.Client

	// h, instance specific handler implementation
	h Handlers

	// s, secret that contains the aiven token for the instance
	s *corev1.Secret

	// log, logger setup with structured fields for the instance
	log logr.Logger

	// rec, recorder to record events for the object
	rec record.EventRecorder
}

func (i instanceReconcilerHelper) reconcileInstance(ctx context.Context, o client.Object) (ctrl.Result, error) {
	i.log.Info("reconciling instance")
	i.rec.Event(o, corev1.EventTypeNormal, eventReconciliationStarted, "starting reconciliation")

	if isMarkedForDeletion(o) {
		if controllerutil.ContainsFinalizer(o, instanceDeletionFinalizer) {
			return i.finalize(ctx, o)
		}
		return ctrl.Result{}, nil
	}

	// Add finalizers to an instance and associated secret, only if they haven't
	// been added in the previous reconciliation loops
	if i.s != nil {
		if !controllerutil.ContainsFinalizer(i.s, secretProtectionFinalizer) {
			i.log.Info("adding finalizer to secret")
			if err := addFinalizer(ctx, i.k8s, i.s, secretProtectionFinalizer); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to add finalizer to secret: %w", err)
			}
		}
	}
	if !controllerutil.ContainsFinalizer(o, instanceDeletionFinalizer) {
		i.log.Info("adding finalizer to instance")
		if err := addFinalizer(ctx, i.k8s, o, instanceDeletionFinalizer); err != nil {
			i.rec.Eventf(o, corev1.EventTypeWarning, eventUnableToAddFinalizer, err.Error())
			return ctrl.Result{}, fmt.Errorf("unable to add finalizer to instance: %w", err)
		}
		i.rec.Event(o, corev1.EventTypeNormal, eventAddedFinalizer, "instance finalizer added")
	}

	// check instance preconditions, if not met - requeue
	i.log.Info("handling service update/creation")
	refs, err := i.getObjectRefs(ctx, o)
	if err != nil {
		i.log.Info(fmt.Sprintf("one or more references can't be found yet: %s", err))
		return ctrl.Result{Requeue: true, RequeueAfter: requeueTimeout}, nil
	}

	requeue, err := i.checkPreconditions(ctx, o, refs)
	if requeue {
		// It must be possible to return requeue and error by design.
		// By the time this comment created, there is no such case in checkPreconditions()
		return ctrl.Result{Requeue: true, RequeueAfter: requeueTimeout}, err
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if !isAlreadyProcessed(o) {
		i.rec.Event(o, corev1.EventTypeNormal, eventCreateOrUpdatedAtAiven, "about to create instance at aiven")
		if err := i.createOrUpdateInstance(o, refs); err != nil {
			i.rec.Event(o, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
			return ctrl.Result{}, fmt.Errorf("unable to create or update instance at aiven: %w", err)
		}

		i.rec.Event(o, corev1.EventTypeNormal, eventCreatedOrUpdatedAtAiven, "instance was created at aiven but may not be running yet")
	}

	i.rec.Event(o, corev1.EventTypeNormal, eventWaitingForTheInstanceToBeRunning, "waiting for the instance to be running")
	isRunning, err := i.updateInstanceStateAndSecretUntilRunning(ctx, o)
	if err != nil {
		if aiven.IsNotFound(err) {
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: requeueTimeout,
			}, nil
		}

		i.rec.Event(o, corev1.EventTypeWarning, eventUnableToWaitForInstanceToBeRunning, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to wait until instance is running: %w", err)
	}

	if !isRunning {
		i.log.Info("instance is not yet running, triggering requeue")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: requeueTimeout,
		}, nil
	}

	i.rec.Event(o, corev1.EventTypeNormal, eventInstanceIsRunning, "instance is in a RUNNING state")
	i.log.Info("instance was successfully reconciled")

	return ctrl.Result{}, nil
}

func (i instanceReconcilerHelper) checkPreconditions(ctx context.Context, o client.Object, refs []client.Object) (bool, error) {
	i.rec.Event(o, corev1.EventTypeNormal, eventWaitingForPreconditions, "waiting for preconditions of the instance")

	// Checks references
	if len(refs) > 0 {
		for _, r := range refs {
			if !(isAlreadyProcessed(r) && isAlreadyRunning(r)) {
				i.log.Info("references are in progress")
				return true, nil
			}
		}
		i.log.Info("all references are good")
	}

	check, err := i.h.checkPreconditions(i.avn, o)
	if err != nil {
		i.rec.Event(o, corev1.EventTypeWarning, eventUnableToWaitForPreconditions, err.Error())
		return false, fmt.Errorf("unable to wait for preconditions: %w", err)
	}

	if !check {
		i.log.Info("preconditions are not met, requeue")
		return true, nil
	}

	i.rec.Event(o, corev1.EventTypeNormal, eventPreconditionsAreMet, "preconditions are met, proceeding to create or update")
	return false, nil
}

func (i instanceReconcilerHelper) getObjectRefs(ctx context.Context, o client.Object) ([]client.Object, error) {
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
func (i instanceReconcilerHelper) finalize(ctx context.Context, o client.Object) (ctrl.Result, error) {
	i.rec.Event(o, corev1.EventTypeNormal, eventTryingToDeleteAtAiven, "trying to delete instance at aiven")

	finalised, err := i.h.delete(i.avn, o)

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
		if i.isInvalidTokenError(err) && !isAlreadyRunning(o) {
			i.log.Info("invalid token error on deletion, removing finalizer", "apiError", err)
			finalised = true
		} else if aiven.IsNotFound(err) {
			i.rec.Event(o, corev1.EventTypeWarning, eventUnableToDeleteAtAiven, err.Error())
			return ctrl.Result{}, fmt.Errorf("unable to delete instance at aiven: %w", err)
		}
	}

	// checking if instance was finalized, if not triggering a requeue
	if !finalised {
		i.log.Info("instance is not yet deleted at aiven, triggering requeue")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: requeueTimeout,
		}, nil
	}

	i.log.Info("instance was successfully deleted at aiven, removing finalizer")
	i.rec.Event(o, corev1.EventTypeNormal, eventSuccessfullyDeletedAtAiven, "instance is gone at aiven now")

	// remove finalizer, once all finalizers have been removed, the object will be deleted.
	if err := removeFinalizer(ctx, i.k8s, o, instanceDeletionFinalizer); err != nil {
		i.rec.Event(o, corev1.EventTypeWarning, eventUnableToDeleteFinalizer, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to remove finalizer: %w", err)
	}

	i.log.Info("finalizer was removed, instance is deleted")
	return ctrl.Result{}, nil
}

// isInvalidTokenError checks if the error is related to invalid token
func (i instanceReconcilerHelper) isInvalidTokenError(err error) bool {
	// When an instance was created but pointing to an invalid API token
	// and no generation was ever processed, allow deleting such instance
	return strings.Contains(err.Error(), "Invalid token")
}

func (i instanceReconcilerHelper) createOrUpdateInstance(o client.Object, refs []client.Object) error {
	i.log.Info("generation wasn't processed, creation or updating instance on aiven side")
	a := o.GetAnnotations()
	delete(a, processedGenerationAnnotation)
	delete(a, instanceIsRunningAnnotation)

	if err := i.h.createOrUpdate(i.avn, o, refs); err != nil {
		return fmt.Errorf("unable to create or update aiven instance: %w", err)
	}

	i.log.Info(
		"processed instance, updating annotations",
		"generation", o.GetGeneration(),
		"annotations", o.GetAnnotations(),
	)
	return nil
}

func (i instanceReconcilerHelper) updateInstanceStateAndSecretUntilRunning(ctx context.Context, o client.Object) (bool, error) {
	var err error

	i.log.Info("checking if instance is ready")

	defer func() {
		// Order matters.
		// First need to update the object, and then update the status.
		// So dependent resources won't see READY before it has been updated with new values
		// Clone is used so update won't overwrite in-memory values
		clone := o.DeepCopyObject().(client.Object)
		err = multierror.Append(err, i.k8s.Update(ctx, clone))

		// Original object has been updated
		o.SetResourceVersion(clone.GetResourceVersion())

		// It's ready to cast its status
		err = multierror.Append(err, i.k8s.Status().Update(ctx, o))
		err = err.(*multierror.Error).ErrorOrNil()
	}()

	serviceSecret, err := i.h.get(i.avn, o)
	if err != nil {
		return false, err
	} else if serviceSecret != nil {
		if err = i.createOrUpdateSecret(ctx, o, serviceSecret); err != nil {
			return false, fmt.Errorf("unable to create or update aiven secret: %w", err)
		}
	}
	return isAlreadyRunning(o), nil

}

func (i instanceReconcilerHelper) createOrUpdateSecret(ctx context.Context, owner client.Object, want *corev1.Secret) error {
	_, err := controllerutil.CreateOrUpdate(ctx, i.k8s, want, func() error {
		return ctrl.SetControllerReference(owner, want, i.k8s.Scheme())
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

// UserConfigurationToAPI converts UserConfiguration options structure
// to Aiven API compatible map[string]interface{}
func UserConfigurationToAPI(c interface{}) interface{} {
	result := make(map[string]interface{})

	v := reflect.ValueOf(c)

	// if its a pointer, resolve its value
	if v.Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}

	if v.Kind() != reflect.Struct {
		switch v.Kind() {
		case reflect.Int64:
			return *c.(*int64)
		case reflect.Bool:
			return *c.(*bool)
		default:
			return c
		}
	}

	structType := v.Type()

	// convert UserConfig structure to a map
	for i := 0; i < structType.NumField(); i++ {
		name := strings.ReplaceAll(structType.Field(i).Tag.Get("json"), ",omitempty", "")

		if structType.Kind() == reflect.Struct {
			result[name] = UserConfigurationToAPI(v.Field(i).Interface())
		} else {
			result[name] = v.Elem().Field(i).Interface()
		}
	}

	// remove all the nil and empty map data
	for key, val := range result {
		if val == nil || isNil(val) || val == "" {
			delete(result, key)
		}

		if reflect.TypeOf(val).Kind() == reflect.Map {
			if len(val.(map[string]interface{})) == 0 {
				delete(result, key)
			}
		}
	}

	return result
}

// UserConfigurationToAPIV2 same as UserConfigurationToAPI but uses sheriff.Marshal
// which can subset fields from create or update operation
func UserConfigurationToAPIV2(userConfig interface{}, groups []string) (map[string]interface{}, error) {
	if userConfig == nil {
		return nil, nil
	}

	o := &sheriff.Options{
		Groups: groups,
	}

	i, err := sheriff.Marshal(o, userConfig)
	if err != nil {
		return nil, err
	}

	m, ok := i.(map[string]interface{})
	if !ok {
		// It is an empty pointer
		// sheriff just returned the very same object
		return nil, nil
	}

	return m, nil
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
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

func ensureSecretDataIsNotEmpty(log *logr.Logger, s *corev1.Secret) *corev1.Secret {
	if s == nil {
		return nil
	}

	for i, v := range s.StringData {
		if len(v) == 0 {
			if log != nil {
				log.Info("secret field is empty, deleting it from the secret",
					"field", v,
					"secret name", s.Name)
			}
			delete(s.StringData, i)
		}
	}

	return s
}
