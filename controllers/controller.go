package controllers

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-multierror"
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

// requeueTimeout sets timeout to requeue controller
const requeueTimeout = 10 * time.Second

// processedGeneration annotations key that holds value of processed generation
const processedGeneration = "processed"

// isRunning annotations key which is set when resource is running on Aiven side
const isRunning = "running"

// requeueAfterTimeout set the timeout for reconciler when to requeue
const requeueAfterTimeout = 10 * time.Second

// formatIntBaseDecimal it is a base to format int64 to string
const formatIntBaseDecimal = 10

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
		createOrUpdate(client.Object) error

		// delete removes an instance on Aiven side.
		// If an object is already deleted and cannot be found, it should not be an error. For other deletion
		// errors, return an error.
		delete(client.Object) (bool, error)

		// get retrieve an object and a secret (for example, connection credentials) that is generated on the
		// fly based on data from Aiven API.  When not applicable to service, it should return nil.
		get(client.Object) (*corev1.Secret, error)

		// checkPreconditions check whether all preconditions for creating (or updating) the resource are in place.
		// For example, it is applicable when a service needs to be running before this resource can be created.
		checkPreconditions(client.Object) (bool, error)
	}
)

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (c *Controller) reconcileInstance(ctx context.Context, h Handlers, o client.Object) (_ ctrl.Result, rErr error) {
	log := c.initLog(o)

	log.Info("reconciling instance")
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

			log.Info("instance was successfully finalized")

			// Checking if instance was finalized, if not triggering a requeue
			if !finalised {
				return ctrl.Result{
					Requeue:      true,
					RequeueAfter: requeueTimeout,
				}, nil
			}

			// Remove finalizer. Once all finalizers have been
			// removed, the object will be deleted.
			log.Info("removing finalizer from instance")
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
		log.Info("add finalizer")
		if err := c.addFinalizer(o, c.getFinalizerName(o)); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check preconditions
	log.Info("checking preconditions")
	check, err := h.checkPreconditions(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !check {
		log.Info("preconditions are not met, requeue")
		return ctrl.Result{Requeue: true, RequeueAfter: requeueTimeout}, nil
	}

	defer func() {
		var result error

		err := c.Status().Update(ctx, o)
		if err != nil {
			log.Error(err, "cannot update CR status")
			result = multierror.Append(result, err)
		}

		err = c.Update(ctx, o)
		if err != nil {
			log.Error(err, "cannot update CR")
			result = multierror.Append(result, err)
		}

		rErr = multierror.Append(rErr, result)
	}()

	log.Info("checking if generation was processed")
	if !c.processed(o) {
		log.Info("generation wasn't processed, creation or updating instance on aiven side")
		c.resetAnnotations(o)
		err := h.createOrUpdate(o)
		if err != nil {
			return ctrl.Result{}, err
		}

		log.Info(fmt.Sprintf("generation %q was processed, setting annotations: %+v",
			o.GetGeneration(), o.GetAnnotations()))
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: requeueAfterTimeout,
		}, nil
	}

	log.Info("getting an instance")
	s, err := h.get(o)
	if err != nil {
		if aiven.IsNotFound(err) {
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: requeueAfterTimeout,
			}, nil
		}
		return ctrl.Result{}, err
	}

	if s != nil {
		err = c.manageSecret(ctx, o, s)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("checking if instance is running")
	if !c.isRunning(o) {
		log.Info("instance is not yet running, triggering requeue")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: requeueAfterTimeout,
		}, nil
	}

	log.Info("instance is successfully reconciled")
	return ctrl.Result{}, nil
}

func (c *Controller) resetAnnotations(o client.Object) {
	a := o.GetAnnotations()
	delete(a, processedGeneration)
	delete(a, isRunning)
}

func (c *Controller) initLog(o client.Object) logr.Logger {
	a := make(map[string]string)
	if r, ok := o.GetAnnotations()[isRunning]; ok {
		a[isRunning] = r
	}

	if g, ok := o.GetAnnotations()[processedGeneration]; ok {
		a[processedGeneration] = g
	}

	return c.Log.WithValues(strings.ToLower(o.GetObjectKind().GroupVersionKind().Kind),
		types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()},
		"annotations", a)
}

func (c *Controller) getFinalizerName(o client.Object) string {
	return fmt.Sprintf("%s-finalizer.aiven.io", o.GetObjectKind().GroupVersionKind().Kind)
}

func (c *Controller) processed(o client.Object) bool {
	for k, v := range o.GetAnnotations() {
		if processedGeneration == k && v == strconv.FormatInt(o.GetGeneration(), formatIntBaseDecimal) {
			return true
		}
	}
	return false
}

func (c *Controller) isRunning(o client.Object) bool {
	_, found := o.GetAnnotations()[isRunning]
	return found
}

func (c *Controller) manageSecret(ctx context.Context, o client.Object, s *corev1.Secret) error {
	createdSecret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: s.Name, Namespace: s.Namespace}, createdSecret)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	err = ctrl.SetControllerReference(o, s, c.Scheme)
	if err != nil {
		return err
	}

	if len(createdSecret.Data) != 0 {
		if err := c.Update(ctx, s); err != nil {
			return err
		}
	} else {
		if err := c.Create(ctx, s); err != nil {
			return err
		}
	}

	return nil
}

// InitAivenClient retrieves an Aiven client
func (c *Controller) InitAivenClient(ctx context.Context, req ctrl.Request, secretRef v1alpha1.AuthSecretReference) (*aiven.Client, error) {
	// Check if aiven-token secret exists
	var token string
	secret := &corev1.Secret{}
	if secretRef.Name == "" || secretRef.Key == "" {
		return nil, fmt.Errorf("secret ref  key or secret is empty, cannot createOrUpdate an aiven client")
	}

	err := c.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: req.Namespace}, secret)
	if err != nil {
		return nil, fmt.Errorf("cannot get %q secret: %w", secretRef.Name, err)
	}

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

func (c *Controller) addFinalizer(o client.Object, f string) error {
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
