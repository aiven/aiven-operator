package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"
)

const requeueTimeout = 10 * time.Second

type (
	// Controller reconciles the Aiven objects
	Controller struct {
		client.Client
		Log         logr.Logger
		Scheme      *runtime.Scheme
		AivenClient *aiven.Client //TODO: remove it, left here for backwards compatibility
	}

	// Handlers represents Aiven API handlers
	// It intended to be a layer between Kubernetes and Aiven API that handles all aspects
	// of the Aiven services lifecycle.
	Handlers interface {
		// create an instance on the Aiven side.
		// If the entity already exists, it should not be an error, but if it impossible to create it by any
		// other reason, it should return an error
		create(logr.Logger, client.Object) (createdObj client.Object, error error)

		// remove an instance on Aiven side.
		// If an object is already deleted and cannot be found, it should not be an error. For other deletion
		// errors, return an error.
		// The return value is any other object that should be deleted along with this one.  For example, if
		// there is a secret associated with an instance, it should return it to have it deleted by controller.
		// When deletion requires further runs, the bool parameter 'isDeleted' should be false and only when an
		// entity was successfully deleted on the Aiven side should it be true.
		delete(logr.Logger, client.Object) (objToBeDeleted client.Object, isDeleted bool, error error)

		// checks if an instance already exists on the Aiven side.
		exists(logr.Logger, client.Object) (exists bool, error error)

		// update an instance on the Aiven side, assuming it was previously created.
		// Should return the updated object, if the update was successful.
		update(logr.Logger, client.Object) (updatedObj client.Object, error error)

		// retrieve a secret (for example, connection credentials) that is generated on the fly based on data
		// from Aiven API.  When not applicable to service, it should return nil.
		getSecret(logr.Logger, client.Object) (secret *corev1.Secret, error error)

		// check whether all preconditions for creating (or updating) the resource are in place.
		// For example, it is applicable when a service needs to be running before this resource can be created.
		checkPreconditions(logr.Logger, client.Object) bool

		// checks if an instance is ready for use on the Aiven side. Applicable for services that have multiple
		// states and start in a transition state. When a service reaches a target state, return true.
		isActive(logr.Logger, client.Object) (bool, error)
	}
)

var aivenClient *aiven.Client

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (c *Controller) reconcileInstance(h Handlers, ctx context.Context, log logr.Logger, req ctrl.Request, o client.Object, finalizerName string) (ctrl.Result, error) {
	// Initiating Aiven client, secret with a token is required
	if err := c.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}
	c.AivenClient = aivenClient //TODO: remove it, left here for backwards compatibility

	// Fetch an instance
	err := c.Get(ctx, req.NamespacedName, o)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			c.Log.Info("Instance resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		c.Log.Error(err, "Failed to get Instance")
		return ctrl.Result{}, err
	}

	// Check if the instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isInstanceMarkedToBeDeleted := o.GetDeletionTimestamp() != nil
	if isInstanceMarkedToBeDeleted {
		if contains(o.GetFinalizers(), finalizerName) {
			// Run finalization logic for finalizer. If the
			// finalization logic fails, don't remove the finalize so
			// that we can retry during the next reconciliation.
			// When applicable, it retrieves an associated object that
			// has to be deleted from Kubernetes, and it could be a
			// secret associated with an instance.
			associatedObjToBeDeleted, finalised, err := h.delete(log, o)
			if err != nil {
				return ctrl.Result{}, err
			}

			// Checking if instance was finalized, if not triggering a requeue
			if !finalised {
				return ctrl.Result{
					Requeue:      true,
					RequeueAfter: requeueTimeout,
				}, nil
			}

			// Delete a K8s secret if handler finalized things on Aiven side
			if associatedObjToBeDeleted != nil {
				if err := c.Delete(ctx, associatedObjToBeDeleted); err != nil {
					return ctrl.Result{}, fmt.Errorf("cannot delete object: %w", err)
				}
			}

			// Remove finalizer. Once all finalizers have been
			// removed, the object will be deleted.
			c.Log.Info("Removing finalizer from instance ...")
			controllerutil.RemoveFinalizer(o, finalizerName)
			err = c.Update(ctx, o)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(o.GetFinalizers(), finalizerName) {
		if err := c.addFinalizer(o, finalizerName); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check preconditions
	if !h.checkPreconditions(log, o) {
		return ctrl.Result{Requeue: true, RequeueAfter: requeueTimeout}, nil
	}

	// Check if instance already exists on Aiven side
	exists, err := h.exists(log, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !exists {
		// If instance does not exist, create a new one
		obj, err := h.create(log, o)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Get a secret if available
		s, err := h.getSecret(log, o)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Creating a secret if available
		if s != nil {
			if err := c.Create(ctx, s); err != nil {
				return ctrl.Result{}, err
			}

			err = controllerutil.SetControllerReference(o, s, c.Scheme)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("k8s set controller reference error %w", err)
			}
		}

		return ctrl.Result{}, c.Status().Update(ctx, obj)
	}

	// Check if instance is already active
	isActive, err := h.isActive(log, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	// If instance is not yet active wait and try again
	if !isActive {
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	// Updating instance on the Aiven side
	obj, err := h.update(log, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	if obj != nil { // If object was updated
		// Updating a secret if available
		s, err := h.getSecret(log, o)
		if err != nil {
			return ctrl.Result{}, err
		}

		if s != nil {
			if err := c.Update(ctx, s); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, c.Status().Update(ctx, obj)
	}

	return ctrl.Result{}, nil
}

// InitAivenClient retrieves an Aiven client
func (c *Controller) InitAivenClient(req ctrl.Request, ctx context.Context, log logr.Logger) error {
	if aivenClient != nil {
		return nil
	}
	log.Info("Initializing an Aiven Client ...")

	// Check if aiven-token secret exists
	var token string
	secret := &corev1.Secret{}
	log.Info("Getting an `aiven-token` secret from the namespace")
	err := c.Get(ctx, types.NamespacedName{Name: "aiven-token", Namespace: req.Namespace}, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "aiven-token secret is missing, it is required by the Aiven client")
		}
		return fmt.Errorf("cannot get `aiven-token` secret: %w", err)
	}

	if v, ok := secret.Data["token"]; ok {
		token = string(v)
	} else {
		return fmt.Errorf("cannot initialize Aiven client, kubernetes secret has no `token` key")
	}

	if len(token) == 0 {
		return fmt.Errorf("cannot initialize Aiven client, `token` key in a secret is empty")
	}

	log.Info("Creating an Aiven Client ...")
	aivenClient, err = aiven.NewTokenClient(token, "k8s-operator/")
	if err != nil {
		return fmt.Errorf("cannot create an Aiven Client: %w", err)
	}

	log.Info("Aiven Client was successfully initialized")
	return nil
}

func (c *Controller) addFinalizer(o client.Object, f string) error {
	c.Log.Info("Adding Finalizer for the instance")
	controllerutil.AddFinalizer(o, f)

	// Update CR
	err := c.Client.Update(context.Background(), o)
	if err != nil {
		c.Log.Error(err, "Failed to update instance with finalize")
		return err
	}

	return nil
}

// contains checks if string slice contains an element
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
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

func stringPointerToString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
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
