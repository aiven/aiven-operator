package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-k8s-operator/api/v1alpha1"
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
		AivenClient *aiven.Client
	}

	// Handlers represents Aiven API handlers
	// It intended to be a layer between Kubernetes and Aiven API that handles all aspects
	// of the Aiven services lifecycle.
	Handlers interface {
		// create an instance on the Aiven side.
		// If the entity already exists, it should not be an error, but if it impossible to create it by any
		// other reason, it should return an error
		create(*aiven.Client, logr.Logger, client.Object) (createdObj client.Object, error error)

		// delete removes an instance on Aiven side.
		// If an object is already deleted and cannot be found, it should not be an error. For other deletion
		// errors, return an error.
		delete(*aiven.Client, logr.Logger, client.Object) (isDeleted bool, error error)

		// exists checks if an instance already exists on the Aiven side.
		exists(*aiven.Client, logr.Logger, client.Object) (exists bool, error error)

		// update an instance on the Aiven side, assuming it was previously created.
		// Should return the updated object, if the update was successful.
		update(*aiven.Client, logr.Logger, client.Object) (updatedObj client.Object, error error)

		// getSecret retrieve a secret (for example, connection credentials) that is generated on the fly based on data
		// from Aiven API.  When not applicable to service, it should return nil.
		getSecret(*aiven.Client, logr.Logger, client.Object) (secret *corev1.Secret, error error)

		// checkPreconditions check whether all preconditions for creating (or updating) the resource are in place.
		// For example, it is applicable when a service needs to be running before this resource can be created.
		checkPreconditions(*aiven.Client, logr.Logger, client.Object) bool

		// isActive checks if an instance is ready for use on the Aiven side. Applicable for services that have multiple
		// states and start in a transition state. When a service reaches a target state, return true.
		isActive(*aiven.Client, logr.Logger, client.Object) (bool, error)

		// getSecretReference retrieves a secret reference that contains name of the
		// secret and key where Aiven API token is located
		getSecretReference(client.Object) *v1alpha1.AuthSecretReference
	}
)

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (c *Controller) reconcileInstance(h Handlers, ctx context.Context, log logr.Logger, req ctrl.Request, o client.Object, finalizerName string) (ctrl.Result, error) {
	// Fetch an instance
	err := c.Get(ctx, req.NamespacedName, o)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			c.Log.Info("instance resource not found. ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		c.Log.Error(err, "failed to get instance")
		return ctrl.Result{}, err
	}

	// Initiating Aiven client, secret with a token is required
	if err := c.InitAivenClient(h, o, req, ctx, log); err != nil {
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
			finalised, err := h.delete(c.AivenClient, log, o)
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

			// Remove finalizer. Once all finalizers have been
			// removed, the object will be deleted.
			c.Log.Info("removing finalizer from instance")
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
	if !h.checkPreconditions(c.AivenClient, log, o) {
		return ctrl.Result{Requeue: true, RequeueAfter: requeueTimeout}, nil
	}

	// Check if instance already exists on Aiven side
	exists, err := h.exists(c.AivenClient, log, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !exists {
		// If instance does not exist, create a new one
		obj, err := h.create(c.AivenClient, log, o)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = c.manageSecret(log, ctx, h, o)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("managing secret %w", err)
		}

		return ctrl.Result{}, c.Status().Update(ctx, obj)
	}

	// Check if instance is already active
	isActive, err := h.isActive(c.AivenClient, log, o)
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
	obj, err := h.update(c.AivenClient, log, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	if obj != nil { // If object was updated
		// Updating a secret if available
		err = c.manageSecret(log, ctx, h, o)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("managing secret %w", err)
		}

		return ctrl.Result{}, c.Status().Update(ctx, obj)
	}

	return ctrl.Result{}, nil
}

func (c *Controller) manageSecret(log logr.Logger, ctx context.Context, h Handlers, o client.Object) error {
	// Get a secret if available
	s, err := h.getSecret(c.AivenClient, log, o)
	if err != nil {
		return err
	}

	// Creating or updating a secret if available
	if s != nil {
		createdSecret := &corev1.Secret{}
		err := c.Get(ctx, types.NamespacedName{Name: s.Name, Namespace: s.Namespace}, createdSecret)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}

		if len(createdSecret.Data) != 0 {
			if err := c.Update(ctx, s); err != nil {
				return err
			}
		} else {
			err = ctrl.SetControllerReference(o, s, c.Scheme)
			if err != nil {
				return err
			}

			if err := c.Create(ctx, s); err != nil {
				return err
			}
		}
	}

	return nil
}

// InitAivenClient retrieves an Aiven client
func (c *Controller) InitAivenClient(h Handlers, o client.Object, req ctrl.Request, ctx context.Context, log logr.Logger) error {
	if c.AivenClient != nil {
		return nil
	}
	log.Info("initializing an aiven client")

	// Check if aiven-token secret exists
	var token string
	secret := &corev1.Secret{}
	secretRef := h.getSecretReference(o)
	if secretRef == nil {
		return fmt.Errorf("secret ref is nil, cannot create an aiven client")
	}

	if secretRef.Name == "" || secretRef.Key == "" {
		return fmt.Errorf("secret ref  key or secret is empty, cannot create an aiven client")
	}

	err := c.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: req.Namespace}, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "secret is missing, it is required by the aiven client", secretRef.Name, "secretName")
		}
		return fmt.Errorf("cannot get %q secret: %w", secretRef.Name, err)
	}

	if v, ok := secret.Data[secretRef.Key]; ok {
		token = string(v)
	} else {
		return fmt.Errorf("cannot initialize aiven client, kubernetes secret has no %q key", secretRef.Key)
	}

	if len(token) == 0 {
		return fmt.Errorf("cannot initialize aiven client, %q key in a secret is empty", secretRef.Key)
	}

	log.Info("creating an aiven client")
	c.AivenClient, err = aiven.NewTokenClient(token, "k8s-operator/")
	if err != nil {
		return fmt.Errorf("cannot create an aiven client: %w", err)
	}

	log.Info("aiven client was successfully initialized")

	return nil
}

func (c *Controller) addFinalizer(o client.Object, f string) error {
	c.Log.Info("adding finalizer for the instance")
	controllerutil.AddFinalizer(o, f)

	// Update CR
	err := c.Client.Update(context.Background(), o)
	if err != nil {
		c.Log.Error(err, "failed to update instance with finalize")
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

func checkServiceIsRunning(c *aiven.Client, project, serviceName string) bool {
	s, err := c.Services.Get(project, serviceName)
	if err != nil {
		return false
	}

	return s.State == "RUNNING"
}
