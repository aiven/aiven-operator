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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
)

// formatIntBaseDecimal it is a base to format int64 to string
const formatIntBaseDecimal = 10

type (
	// Controller reconciles the Aiven objects
	Controller struct {
		client.Client

		Log      logr.Logger
		Scheme   *runtime.Scheme
		Recorder record.EventRecorder
	}

	// Handlers represents Aiven API handlers
	// It intended to be a layer between Kubernetes and Aiven API that handles all aspects
	// of the Aiven services lifecycle.
	Handlers interface {
		// create or updates an instance on the Aiven side.
		createOrUpdate(*aiven.Client, client.Object) error

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
	eventReconciliationStarted               = "ReconcilationStarted"
	eventBadAuthSecretRef                    = "BadAuthSecretRef"
	eventUnableToCreateClient                = "UnableToCreateClient"
	eventNotYetDeletedAtAiven                = "NotYetDeletedAtAiven"
	eventUnableToDeleteAtAiven               = "UnableToDeleteAtAiven"
	eventSuccessfullyDeletedAtAiven          = "SuccessfullyDeletedAtAiven"
	eventAddedFinalizer                      = "InstanceFinalizerAdded"
	eventUnableToAddFinalizer                = "UnableToAddFinalizer"
	eventUnableToCheckPreconditions          = "UnableToCheckPreconditions"
	eventPreconditionsAreNotMet              = "PreconditionsAreNotMet"
	eventUnableToCreateOrUpdateAtAiven       = "UnableToCreateOrUpdateAtAiven"
	eventCreatedOrUpdatedAtAiven             = "CreatedOrUpdatedAtAiven"
	eventInstanceNotFoundYetAtAiven          = "InstanceWasNotFoundYetAtAiven"
	eventUnableToGetInstanceAtAiven          = "UnableToGetInstanceAtAiven"
	eventUnableToCreateOrUpdateServiceSecret = "UnableToCreateOrUpdateServiceSecret"
	eventCreatedOrUpdatedServiceSecret       = "CreatedOrUpdatedServiceSecret"
	eventNotYetRunning                       = "NotYetRunning"
	eventRunning                             = "Running"
)

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (c *Controller) reconcileInstance(ctx context.Context, req ctrl.Request, h Handlers, o aivenManagedObject) (_ ctrl.Result, rErr error) {
	if err := c.Get(ctx, req.NamespacedName, o); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log := c.loggerForInstance(o)
	log.Info("reconciling instance")
	c.Recorder.Event(o, corev1.EventTypeNormal, eventReconciliationStarted, "starting reconciliation")

	clientAuthSecret := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Name: o.AuthSecretRef().Name, Namespace: req.Namespace}, clientAuthSecret); err != nil {
		c.Recorder.Eventf(o, corev1.EventTypeWarning, eventBadAuthSecretRef, "aiven client secret '%s' in namespace '%s' was not found.", o.AuthSecretRef().Name, req.Namespace)
		return ctrl.Result{}, fmt.Errorf("cannot get secret %q: %w", o.AuthSecretRef().Name, err)
	}
	avn, err := aiven.NewTokenClient(string(clientAuthSecret.Data[o.AuthSecretRef().Key]), "k8s-operator/")
	if err != nil {
		c.Recorder.Event(o, corev1.EventTypeWarning, eventUnableToCreateClient, err.Error())
		return ctrl.Result{}, fmt.Errorf("cannot initialize aiven client: %w", err)
	}

	log.Info("handling finalizers")
	if markedForDeletion(o) {
		if controllerutil.ContainsFinalizer(o, instanceDeletionFinalizer) {
			log.Info("deleting instance at aiven")

			if deleted, err := h.delete(avn, o); err != nil {
				c.Recorder.Event(o, corev1.EventTypeWarning, eventUnableToDeleteAtAiven, err.Error())
				return ctrl.Result{}, err
			} else if !deleted {
				log.Info("instance was not deleted, requeue")
				c.Recorder.Event(o, corev1.EventTypeNormal, eventNotYetDeletedAtAiven, "instance deletion may take some time")
				return requeueCtrlResult(), nil
			}
			log.Info("instance was successfully deleted at aiven, removing finalizer")
			c.Recorder.Event(o, corev1.EventTypeNormal, eventSuccessfullyDeletedAtAiven, "instance is gone now")
			if err = removeFinalizer(ctx, c.Client, o, instanceDeletionFinalizer); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to remove finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	} else {
		if !controllerutil.ContainsFinalizer(clientAuthSecret, secretProtectionFinalizer) {
			log.Info("adding finalizer to secret")
			if err := addFinalizer(ctx, c.Client, clientAuthSecret, secretProtectionFinalizer); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to add finalizer to secret: %w", err)
			}
		}
		if !controllerutil.ContainsFinalizer(o, instanceDeletionFinalizer) {
			log.Info("adding finalizer to instance")
			if err := addFinalizer(ctx, c.Client, o, instanceDeletionFinalizer); err != nil {
				c.Recorder.Eventf(o, corev1.EventTypeWarning, eventUnableToAddFinalizer, err.Error())
				return ctrl.Result{}, fmt.Errorf("unable to add finalizer to instance: %w", err)
			}
			c.Recorder.Event(o, corev1.EventTypeNormal, eventAddedFinalizer, "instance finalizer added")
		}
	}

	log.Info("handling service update/creation")
	log.Info("checking preconditions")
	if check, err := h.checkPreconditions(avn, o); err != nil {
		c.Recorder.Event(o, corev1.EventTypeWarning, eventUnableToCheckPreconditions, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to check preconditions: %w", err)
	} else if !check {
		log.Info("preconditions are not met, requeue")
		c.Recorder.Event(o, corev1.EventTypeNormal, eventPreconditionsAreNotMet, "preconditions for this service are not met yet")
		return requeueCtrlResult(), nil
	}

	defer func() {
		rErr = multierror.Append(rErr, c.Status().Update(ctx, o))
		rErr = multierror.Append(rErr, c.Update(ctx, o))
		rErr = rErr.(*multierror.Error).ErrorOrNil()
	}()

	log.Info("checking if generation was processed")
	if !isAlreadyProcessed(o) {
		log.Info("generation wasn't processed, creation or updating instance on aiven side")
		c.resetAnnotations(o)
		if err := h.createOrUpdate(avn, o); err != nil {
			c.Recorder.Event(o, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
			return ctrl.Result{}, fmt.Errorf("unable to create or update aiven instance: %w", err)
		}
		log.Info("processed instance, updating annotations", "generation", o.GetGeneration(), "annotations", o.GetAnnotations())
		c.Recorder.Event(o, corev1.EventTypeNormal, eventCreatedOrUpdatedAtAiven, "instance was created at aiven but may not be running yet")
		return requeueCtrlResult(), nil
	}

	log.Info("managing instance secret")
	if serviceSecret, err := h.get(avn, o); err != nil {
		if aiven.IsNotFound(err) {
			log.Info("instance not found, requeue")
			c.Recorder.Event(o, corev1.EventTypeNormal, eventInstanceNotFoundYetAtAiven, "instance was created but is not yet found via the api")
			return requeueCtrlResult(), nil
		}
		c.Recorder.Event(o, corev1.EventTypeWarning, eventUnableToGetInstanceAtAiven, err.Error())
		return ctrl.Result{}, fmt.Errorf("unable to get aiven instance: %w", err)
	} else if serviceSecret != nil {
		if err = c.createOrUpdateSecret(ctx, o, serviceSecret); err != nil {
			c.Recorder.Event(o, corev1.EventTypeWarning, eventUnableToCreateOrUpdateServiceSecret, err.Error())
			return ctrl.Result{}, fmt.Errorf("unable to manage aiven secret: %w", err)
		}
		c.Recorder.Eventf(o, corev1.EventTypeNormal, eventCreatedOrUpdatedServiceSecret, "%s/%s", serviceSecret.GetNamespace(), serviceSecret.GetName())
	}

	log.Info("checking if instance is running")
	if !isAlreadyRunning(o) {
		log.Info("instance is not yet running, requeue")
		c.Recorder.Event(o, corev1.EventTypeNormal, eventNotYetRunning, "instance is not yet running")
		return requeueCtrlResult(), nil
	}

	log.Info("instance was successfully reconciled")
	c.Recorder.Event(o, corev1.EventTypeNormal, eventRunning, "instance is running")
	return ctrl.Result{}, nil
}

func (c *Controller) resetAnnotations(o client.Object) {
	a := o.GetAnnotations()
	delete(a, processedGenerationAnnotation)
	delete(a, instanceIsRunningAnnotation)
}

func (c *Controller) loggerForInstance(o client.Object) logr.Logger {
	a := make(map[string]string)
	if r, ok := o.GetAnnotations()[instanceIsRunningAnnotation]; ok {
		a[instanceIsRunningAnnotation] = r
	}

	if g, ok := o.GetAnnotations()[processedGenerationAnnotation]; ok {
		a[processedGenerationAnnotation] = g
	}
	kind := strings.ToLower(o.GetObjectKind().GroupVersionKind().Kind)
	name := types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}

	return c.Log.WithValues("kind", kind, "name", name, "annotations", a)
}

func (c *Controller) createOrUpdateSecret(ctx context.Context, owner client.Object, want *corev1.Secret) error {
	_, err := controllerutil.CreateOrUpdate(ctx, c.Client, want, func() error {
		return ctrl.SetControllerReference(owner, want, c.Scheme)
	})
	return err
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

func checkServiceIsRunning(c *aiven.Client, project, serviceName string) (bool, error) {
	s, err := c.Services.Get(project, serviceName)
	if err != nil {
		return false, err
	}

	return s.State == "RUNNING", nil
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
