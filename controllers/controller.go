package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const requeueTimeout = 10 * time.Second
const processedGeneration = "processed"
const isRunning = "running"

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
		// create or updates an instance on the Aiven side.
		createOrUpdate(client.Object) (error error)

		// delete removes an instance on Aiven side.
		// If an object is already deleted and cannot be found, it should not be an error. For other deletion
		// errors, return an error.
		delete(client.Object) (bool, error)

		// get retrieve an obejct and a secret (for example, connection credentials) that is generated on the
		// fly based on data from Aiven API.  When not applicable to service, it should return nil.
		get(client.Object) (client.Object, *corev1.Secret, error)

		// checkPreconditions check whether all preconditions for creating (or updating) the resource are in place.
		// For example, it is applicable when a service needs to be running before this resource can be created.
		checkPreconditions(client.Object) bool
	}
)

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;createOrUpdate;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;createOrUpdate;update;patch;delete

func (c *Controller) reconcileInstance(ctx context.Context, req ctrl.Request, h Handlers, o client.Object) (ctrl.Result, error) {
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
		if contains(o.GetFinalizers(), c.getFinalizerName(o)) {
			// Run finalization logic for finalizer. If the
			// finalization logic fails, don't remove the finalize so
			// that we can retry during the next reconciliation.
			// When applicable, it retrieves an associated object that
			// has to be deleted from Kubernetes, and it could be a
			// secret associated with an instance.
			finalised, err := h.delete(o)
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
			controllerutil.RemoveFinalizer(o, c.getFinalizerName(o))
			err = c.Update(ctx, o)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(o.GetFinalizers(), c.getFinalizerName(o)) {
		if err := c.addFinalizer(o, c.getFinalizerName(o)); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check preconditions
	if !h.checkPreconditions(o) {
		return ctrl.Result{Requeue: true, RequeueAfter: requeueTimeout}, nil
	}

	if !c.processed(o) {
		// If instance does not exist, createOrUpdate a new one
		err := h.createOrUpdate(o)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	obj, s, err := h.get(o)
	if err != nil {
		if aiven.IsNotFound(err) {
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: 10 * time.Second,
			}, nil
		}
		return ctrl.Result{}, err
	}

	if obj != nil {
		err = c.Status().Update(ctx, obj)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if s != nil {
		err = c.manageSecret(ctx, obj, s)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if !c.isRunning(o) {
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	return ctrl.Result{}, nil
}

func (c *Controller) getFinalizerName(o client.Object) string {
	return fmt.Sprintf("%s-finalizer.aiven.io", o.GetName())
}

func (c *Controller) processed(o client.Object) bool {
	for k, v := range o.GetAnnotations() {
		if processedGeneration == k && v == strconv.FormatInt(o.GetGeneration(), 10) {
			return true
		}
	}
	return false
}

func (c *Controller) markAsProcessed(o client.Object) error {
	ann := o.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string)
	}

	ann[processedGeneration] = strconv.FormatInt(o.GetGeneration(), 10)

	return c.Client.Update(context.Background(), o)
}

func (c *Controller) isRunning(o client.Object) bool {
	_, found := o.GetAnnotations()[isRunning]
	return found
}

func (c *Controller) manageSecret(ctx context.Context, o client.Object, s *corev1.Secret) error {
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
func (c *Controller) InitAivenClient(ctx context.Context, req ctrl.Request, secretRef v1alpha1.AuthSecretReference) error {
	if c.AivenClient != nil {
		return nil
	}

	// Check if aiven-token secret exists
	var token string
	secret := &corev1.Secret{}
	if secretRef.Name == "" || secretRef.Key == "" {
		return fmt.Errorf("secret ref  key or secret is empty, cannot createOrUpdate an aiven client")
	}

	err := c.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: req.Namespace}, secret)
	if err != nil {
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

	c.AivenClient, err = aiven.NewTokenClient(token, "k8s-operator/")
	if err != nil {
		return fmt.Errorf("cannot createOrUpdate an aiven client: %w", err)
	}

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
