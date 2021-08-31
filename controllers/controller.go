package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
)

type (
	// Controller reconciles the Aiven objects
	Controller struct {
		client.Client

		Log      logr.Logger
		Scheme   *runtime.Scheme
		Recorder record.EventRecorder
	}

	// handlers handle all CRD specific logic
	Handlers interface {
		// create or updates an instance on the Aiven side.
		createOrUpdate(*aiven.Client, client.Object) error

		// fetches the resources that is expected to own this resource
		// it is not expected that all owners can be found as k8s objects since there may be
		// resources that were manually created so "not-found" errors should be ignored
		fetchOwners(context.Context, client.Object) ([]client.Object, error)

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

		AuthSecretRef() v1alpha1.AuthSecretReference
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
	eventWaitingforPreconditions            = "WaitingForPreconditions"
	eventUnableToWaitForPreconditions       = "UnableToWaitForPreconditions"
	eventPreconditionsAreMet                = "PreconditionsAreMet"
	eventAddOwnerReferences                 = "AddOwnerReferences"
	eventUnableToAddOwnerReferences         = "UnableToAddOwnerReferences"
	eventAddedOwnerReferences               = "AddedOwnerReferences"
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

	clientAuthSecret := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Name: o.AuthSecretRef().Name, Namespace: req.Namespace}, clientAuthSecret); err != nil {
		c.Recorder.Eventf(o, corev1.EventTypeWarning, eventUnableToGetAuthSecret, err.Error())
		return ctrl.Result{}, fmt.Errorf("cannot get secret %q: %w", o.AuthSecretRef().Name, err)
	}
	avn, err := aiven.NewTokenClient(string(clientAuthSecret.Data[o.AuthSecretRef().Key]), "k8s-operator/")
	if err != nil {
		c.Recorder.Event(o, corev1.EventTypeWarning, eventUnableToCreateClient, err.Error())
		return ctrl.Result{}, fmt.Errorf("cannot initialize aiven client: %w", err)
	}

	return instanceReconcilerHelper{
		avn:  avn,
		k8s:  c.Client,
		hnd:  h,
		log:  instanceLogger,
		scrt: clientAuthSecret,
		rec:  c.Recorder,
	}.reconcileInstance(ctx, o)
}

// a helper that closes over all instance specific fields
// to make reconciliation a little more ergonomic
type instanceReconcilerHelper struct {
	k8s client.Client

	// aiven client that is authorized with the instance token
	avn *aiven.Client
	// instance specific handler implementation
	hnd Handlers
	// secret that contains the aiven token for the instance
	scrt *corev1.Secret

	// logger setup with structured fields for the instance
	log logr.Logger
	// recprder to record events for the object
	rec record.EventRecorder
}

func (ir instanceReconcilerHelper) reconcileInstance(ctx context.Context, o client.Object) (ctrl.Result, error) {
	ir.log.Info("reconciling instance")
	ir.rec.Event(o, corev1.EventTypeNormal, eventReconciliationStarted, "starting reconciliation")

	ir.log.Info("handling finalizers")

	if markedForDeletion(o) {
		if controllerutil.ContainsFinalizer(o, instanceDeletionFinalizer) {
			ir.rec.Event(o, corev1.EventTypeNormal, eventTryingToDeleteAtAiven, "trying to delete instance at aiven")
			if err := ir.deleteSync(o); err != nil {
				ir.rec.Event(o, corev1.EventTypeWarning, eventUnableToDeleteAtAiven, err.Error())
				return ctrl.Result{}, fmt.Errorf("unable to delete instance at aiven: %w", err)
			}
			ir.log.Info("instance was successfully deleted at aiven, removing finalizer")
			ir.rec.Event(o, corev1.EventTypeNormal, eventSuccessfullyDeletedAtAiven, "instance is gone at aiven now")
			if err := removeFinalizer(ctx, ir.k8s, o, instanceDeletionFinalizer); err != nil {
				ir.rec.Event(o, corev1.EventTypeWarning, eventUnableToDeleteFinalizer, err.Error())
				return ctrl.Result{}, fmt.Errorf("unable to remove finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	} else {
		if !controllerutil.ContainsFinalizer(ir.scrt, secretProtectionFinalizer) {
			ir.log.Info("adding finalizer to secret")
			if err := addFinalizer(ctx, ir.k8s, ir.scrt, secretProtectionFinalizer); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to add finalizer to secret: %w", err)
			}
		}
		if !controllerutil.ContainsFinalizer(o, instanceDeletionFinalizer) {
			ir.log.Info("adding finalizer to instance")
			if err := addFinalizer(ctx, ir.k8s, o, instanceDeletionFinalizer); err != nil {
				ir.rec.Eventf(o, corev1.EventTypeWarning, eventUnableToAddFinalizer, err.Error())
				return ctrl.Result{}, fmt.Errorf("unable to add finalizer to instance: %w", err)
			}
			ir.rec.Event(o, corev1.EventTypeNormal, eventAddedFinalizer, "instance finalizer added")
		}
	}
	ir.log.Info("handling service update/creation")

	// wait for preconditions
	ir.rec.Event(o, corev1.EventTypeNormal, eventWaitingforPreconditions, "waiting for preconditions of the instance")
	if err := ir.waitForPreconditions(o); err != nil {
		ir.rec.Event(o, corev1.EventTypeWarning, eventUnableToWaitForPreconditions, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to wait for preconditions: %w", err)
	}
	ir.rec.Event(o, corev1.EventTypeNormal, eventPreconditionsAreMet, "preconditions are met, proceeding to create or update")

	// add owner references
	ir.rec.Event(o, corev1.EventTypeNormal, eventAddOwnerReferences, "adding owner references")
	if err := ir.addOwnerReferences(ctx, o); err != nil {
		ir.rec.Event(o, corev1.EventTypeWarning, eventUnableToAddOwnerReferences, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to add owner references: %w", err)
	}
	ir.rec.Event(o, corev1.EventTypeNormal, eventAddedOwnerReferences, "instance is properly owned now")

	// create or update
	ir.rec.Event(o, corev1.EventTypeNormal, eventCreateOrUpdatedAtAiven, "about to create instance at aiven")
	if err := ir.createOrUpdateInstance(o); err != nil {
		ir.rec.Event(o, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to create or update instance at aiven: %w", err)
	}
	ir.rec.Event(o, corev1.EventTypeNormal, eventCreatedOrUpdatedAtAiven, "instance was created at aiven but may not be running yet")

	// wait for the service to be running
	ir.rec.Event(o, corev1.EventTypeNormal, eventWaitingForTheInstanceToBeRunning, "waiting for the instance to be running")
	if err := ir.updateInstanceStateAndSecretUntilRunning(ctx, o); err != nil {
		ir.rec.Event(o, corev1.EventTypeWarning, eventUnableToWaitForInstanceToBeRunning, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to wait until instance is running: %w", err)
	}
	ir.rec.Event(o, corev1.EventTypeNormal, eventInstanceIsRunning, "instance is in a RUNNING state")

	ir.log.Info("instance was successfully reconciled")
	return ctrl.Result{}, nil
}

func (ir instanceReconcilerHelper) deleteSync(o client.Object) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (bool, error) {
		ir.log.Info("trying to delete aiven instance")
		return ir.hnd.delete(ir.avn, o)
	})
}

func (ir instanceReconcilerHelper) waitForPreconditions(o client.Object) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (bool, error) {
		ir.log.Info("checking preconditions")
		return ir.hnd.checkPreconditions(ir.avn, o)
	})
}

func (ir instanceReconcilerHelper) addOwnerReferences(ctx context.Context, o client.Object) error {
	owners, err := ir.hnd.fetchOwners(ctx, o)
	if err != nil {
		return fmt.Errorf("unable to fetch owners: %w", err)
	}
	for i := range owners {
		if err := controllerutil.SetOwnerReference(owners[i], o, ir.k8s.Scheme()); err != nil {
			return fmt.Errorf("unable to set owner reference to '%s': %w", owners[i].GetName(), err)
		}
	}
	return nil
}

func (ir instanceReconcilerHelper) createOrUpdateInstance(o client.Object) error {
	ir.log.Info("checking if generation was processed")

	if isAlreadyProcessed(o) {
		return nil
	}

	ir.log.Info("generation wasn't processed, creation or updating instance on aiven side")
	a := o.GetAnnotations()
	delete(a, processedGenerationAnnotation)
	delete(a, instanceIsRunningAnnotation)

	if err := ir.hnd.createOrUpdate(ir.avn, o); err != nil {
		return fmt.Errorf("unable to create or update aiven instance: %w", err)
	}
	ir.log.Info(
		"processed instance, updating annotations",
		"generation", o.GetGeneration(),
		"annotations", o.GetAnnotations(),
	)
	return nil
}

func (ir instanceReconcilerHelper) updateInstanceStateAndSecretUntilRunning(ctx context.Context, o client.Object) (err error) {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (bool, error) {
		ir.log.Info("checking if instance is ready")

		defer func() {
			err = multierror.Append(err, ir.k8s.Status().Update(ctx, o))
			err = multierror.Append(err, ir.k8s.Update(ctx, o))
			err = err.(*multierror.Error).ErrorOrNil()
		}()

		serviceSecret, err := ir.hnd.get(ir.avn, o)
		if err != nil {
			if aiven.IsNotFound(err) {
				return false, nil
			}
			return false, fmt.Errorf("unable to get aiven instance: %w", err)
		} else if serviceSecret != nil {
			if err = ir.createOrUpdateSecret(ctx, o, serviceSecret); err != nil {
				return false, fmt.Errorf("unable to create or update aiven secret: %w", err)
			}
		}
		return isAlreadyRunning(o), nil
	})
}

func (ir instanceReconcilerHelper) createOrUpdateSecret(ctx context.Context, owner client.Object, want *corev1.Secret) error {
	_, err := controllerutil.CreateOrUpdate(ctx, ir.k8s, want, func() error {
		return ctrl.SetControllerReference(owner, want, ir.k8s.Scheme())
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

func ensureSecretDataIsNotEmpty(log logr.Logger, s *corev1.Secret) *corev1.Secret {
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
