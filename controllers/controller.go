package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
)

const (
	// requeueTimeout sets timeout to requeue controller
	requeueTimeout = 10 * time.Second

	// processedGeneration annotations key that holds value of processed generation
	processedGeneration = "processed"

	// isRunning annotations key which is set when resource is running on Aiven side
	isRunning = "running"

	// requeueAfterTimeout set the timeout for reconciler when to requeue
	requeueAfterTimeout = 10 * time.Second

	// formatIntBaseDecimal it is a base to format int64 to string
	formatIntBaseDecimal = 10
)

var (
	requeueResult = ctrl.Result{Requeue: true, RequeueAfter: requeueTimeout}
)

type (
	// Controller reconciles the Aiven objects
	Controller struct {
		client.Client
		Log    logr.Logger
		Scheme *runtime.Scheme
	}

	// Handlers represents Aiven API handlers
	// It intended to be a layer between Kubernetes and Aiven API that handles all aspects
	// of the Aiven services lifecycle.
	Handlers interface {
		// create or updates an instance on the Aiven side.
		createOrUpdate(client.Object) (client.Object, error)

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

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (c *Controller) reconcileInstance(ctx context.Context, h Handlers, o client.Object) (ctrl.Result, error) {
	log := c.initLog(o)
	log.Info("reconciling instance")

	// handle finalizer logic

	controllerFinalizer := c.getFinalizerName(o)
	if o.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(o, controllerFinalizer) {
			log.Info("adding finalizer", "finalizer", controllerFinalizer)

			if err := c.addFinalizer(ctx, o, controllerFinalizer); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to register finalizer: %w", err)
			}
		}

	} else {
		if controllerutil.ContainsFinalizer(o, controllerFinalizer) {
			log.Info("running finalizer", "finalizer", controllerFinalizer)

			if deleted, err := h.delete(o); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to delete object: %w", err)
			} else if !deleted {
				return requeueResult, nil
			}
			if err := c.removeFinalizer(ctx, o, controllerFinalizer); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to remove finalizer: %w", err)
			}

			log.Info("instance was successfully finalized")
			return ctrl.Result{}, nil
		}
	}

	// handle reconcile logic
	//
	// TODO: writeup logic here

	log.Info("checking preconditions")
	if !h.checkPreconditions(o) {
		log.Info("preconditions are not met, requeue")
		return requeueResult, nil
	}

	log.Info("checking if generation was processed")
	if !c.generationAlreadyProcessed(o) {
		log.Info("generation wasn't processed yet, creating/updating instance on aiven side")

		newObj, err := h.createOrUpdate(o)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to create or update object: %w", err)
		}
		log.Info("object was created/updated setting annotations",
			"generation", newObj.GetGeneration(),
			"annotation", newObj.GetAnnotations(),
		)
		if err = c.updateStatusAndState(ctx, newObj); err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to update status and state: %w", err)
		}
		return requeueResult, nil
	} else {
		log.Info("generation was already processed, checking for readyness")

		curObj, secret, err := h.get(o)
		if err != nil {
			if aiven.IsNotFound(err) {
				log.Info("instance is not yet found")
				return requeueResult, nil
			}
			return ctrl.Result{}, fmt.Errorf("unable to get object: %w", err)
		}
		if curObj != nil {
			if err = c.updateStatusAndState(ctx, curObj); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to update status and state: %w", err)
			}
		}
		if secret != nil {
			if err = c.manageSecret(ctx, curObj, secret); err != nil {
				return ctrl.Result{}, fmt.Errorf("unable to manage secret: %w", err)
			}
		}
		if !c.isRunning(curObj) {
			log.Info("instance is not yet running")
			return requeueResult, nil
		}
	}

	log.Info("instance is successfully reconciled")
	return ctrl.Result{}, nil
}

func (c *Controller) getFinalizerName(o client.Object) string {
	return fmt.Sprintf("%s-finalizer.aiven.io", o.GetObjectKind().GroupVersionKind().Kind)
}

func (c *Controller) addFinalizer(ctx context.Context, o client.Object, f string) error {
	controllerutil.AddFinalizer(o, f)
	if err := c.Client.Update(ctx, o); err != nil {
		return fmt.Errorf("unable to update instance: %w", err)
	}
	return nil
}

func (c *Controller) removeFinalizer(ctx context.Context, o client.Object, f string) error {
	controllerutil.RemoveFinalizer(o, f)
	if err := c.Client.Update(ctx, o); err != nil {
		return fmt.Errorf("unable to update instance: %w", err)
	}
	return nil
}

func (c *Controller) updateStatusAndState(ctx context.Context, obj client.Object) error {
	if err := c.Status().Update(ctx, obj); err != nil {
		return fmt.Errorf("unable to update status: %w", err)
	}
	if err := c.Update(ctx, obj); err != nil {
		return fmt.Errorf("unable to update state: %w", err)
	}
	return nil
}

func (c *Controller) initLog(o client.Object) logr.Logger {
	a := make(map[string]string)
	if r, ok := o.GetAnnotations()[isRunning]; ok {
		a[isRunning] = r
	}

	if g, ok := o.GetAnnotations()[processedGeneration]; ok {
		a[processedGeneration] = g
	}

	return c.Log.WithValues(
		strings.ToLower(o.GetObjectKind().GroupVersionKind().Kind),
		types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()},
		"annotations", a)
}

func (c *Controller) generationAlreadyProcessed(o client.Object) bool {
	return o.GetAnnotations()[processedGeneration] == strconv.FormatInt(o.GetGeneration(), formatIntBaseDecimal)
}

func (c *Controller) isRunning(o client.Object) bool {
	_, found := o.GetAnnotations()[isRunning]
	return found
}

func (c *Controller) manageSecret(ctx context.Context, o client.Object, s *corev1.Secret) error {
	var createdSecret *corev1.Secret

	err := c.Get(ctx, types.NamespacedName{Name: s.Name, Namespace: s.Namespace}, createdSecret)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("unable to get secret: %w", err)
	}
	if err = ctrl.SetControllerReference(o, s, c.Scheme); err != nil {
		return fmt.Errorf("unable to set controller reference in secret: %w", err)
	}

	if len(createdSecret.Data) != 0 {
		if err := c.Update(ctx, s); err != nil {
			return fmt.Errorf("unable to update secret: %w", err)
		}
	} else {
		if err := c.Create(ctx, s); err != nil {
			return fmt.Errorf("unable to create secret: %w", err)
		}
	}

	return nil
}

// InitAivenClient retrieves an Aiven client
func (c *Controller) InitAivenClient(ctx context.Context, req ctrl.Request, secretRef v1alpha1.AuthSecretReference) (*aiven.Client, error) {
	// Check if aiven-token secret exists
	var secret *corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: req.Namespace}, secret); err != nil {
		return nil, fmt.Errorf("cannot get %q secret: %w", secretRef.Name, err)
	}

	var token string
	if v, ok := secret.Data[secretRef.Key]; ok {
		token = string(v)
	} else {
		return nil, fmt.Errorf("cannot initialize aiven client, kubernetes secret has no %q key", secretRef.Key)
	}
	if len(token) == 0 {
		return nil, fmt.Errorf("cannot initialize aiven client, %q key in a secret is empty", secretRef.Key)
	}

	con, err := aiven.NewTokenClient(token, "k8s-operator/")
	if err != nil {
		return nil, fmt.Errorf("cannot createOrUpdate an aiven client: %w", err)
	}
	return con, nil
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
