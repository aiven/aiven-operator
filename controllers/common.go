package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/liip/sheriff"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const (
	conditionTypeRunning     = "Running"
	conditionTypeInitialized = "Initialized"
	ConditionTypeError       = "Error"

	secretProtectionFinalizer = "finalizers.aiven.io/needed-to-delete-services"
	instanceDeletionFinalizer = "finalizers.aiven.io/delete-remote-resource"

	processedGenerationAnnotation = "controllers.aiven.io/generation-was-processed"
	instanceIsRunningAnnotation   = "controllers.aiven.io/instance-is-running"
	secretSourceUpdatedAnnotation = "controllers.aiven.io/secret-source-updated"

	deletionPolicyAnnotation = "controllers.aiven.io/deletion-policy"
	deletionPolicyOrphan     = "Orphan"
	deletionPolicyDelete     = "Delete"
)

type errCondition string

const (
	errConditionDelete         errCondition = "Delete"
	errConditionPreconditions  errCondition = "Preconditions"
	errConditionCreateOrUpdate errCondition = "CreateOrUpdate"
)

var (
	errTerminationProtectionOn = errors.New("termination protection is on")
	errServicePoweredOff       = errors.New("service is powered off")
)

// ErrRequeueNeeded is an error type that indicates that the reconciliation should be requeued.
// It is used to handle errors that are expected to be resolved on a subsequent retries.
type ErrRequeueNeeded struct {
	OriginalError error
}

func (e ErrRequeueNeeded) Error() string {
	return fmt.Sprintf("requeue needed: %s", e.OriginalError.Error())
}

func (e ErrRequeueNeeded) Unwrap() error {
	return e.OriginalError
}

// checkServiceIsOperational checks if a service is in operational state, i.e., can create databases, users, etc.
// Returns errServicePoweredOff if the service is powered off.
func checkServiceIsOperational(ctx context.Context, avnGen avngen.Client, project, serviceName string) (bool, error) {
	s, err := avnGen.ServiceGet(ctx, project, serviceName)
	if isNotFound(err) {
		// Service not found indicates it hasn't started running.
		// We ignore not found errors since they could mean either:
		// 1. The service doesn't exist yet
		// 2. The project doesn't exist yet (may be created by operator)
		return false, nil
	}

	if err != nil {
		return false, err
	}

	switch s.State {
	case service.ServiceStateTypeRebalancing, service.ServiceStateTypeRunning:
		// Running means the service is fully operational.
		// Rebalancing doesn't block most of the operations.
		// But depending on the service type and the operation, additional checks may be needed.
		return true, nil
	case service.ServiceStateTypePoweroff:
		// If the service is powered off, returns an error,
		// so that Kube won't infinitely retry the Aiven API.
		return false, fmt.Errorf("%w: %s/%s", errServicePoweredOff, project, serviceName)
	}

	// Must be an intermediate state, e.g. rebuilding, etc.
	return false, nil
}

// checkServiceIsOperational checks if a service is in operational state, i.e., can create databases, users, etc.
// Returns errServicePoweredOff if the service is powered off.
func checkServiceIsOperational2(ctx context.Context, avnGen avngen.Client, project, serviceName string) error {
	s, err := avnGen.ServiceGet(ctx, project, serviceName)
	if isNotFound(err) {
		// Service not found indicates it hasn't started running.
		// We ignore not found errors since they could mean either:
		// 1. The service doesn't exist yet
		// 2. The project doesn't exist yet (may be created by operator)
		return fmt.Errorf("%w: %w", errPreconditionNotMet, err)
	}

	if err != nil {
		// Preserve original error semantics (including 5xx) so that handleObserveError can classify retryable Aiven errors.
		return err
	}

	switch s.State {
	case service.ServiceStateTypeRebalancing, service.ServiceStateTypeRunning:
		// Running means the service is fully operational.
		// Rebalancing doesn't block most of the operations.
		// But depending on the service type and the operation, additional checks may be needed.
		return nil
	case service.ServiceStateTypePoweroff:
		// If the service is powered off, returns an error,
		// so that Kube won't infinitely retry the Aiven API.
		return fmt.Errorf("%w: %s/%s", errServicePoweredOff, project, serviceName)
	}

	// Must be an intermediate state, e.g. rebuilding, etc.
	return fmt.Errorf("%w: service %s/%s is not yet operational", errPreconditionNotMet, project, serviceName)
}

func getInitializedCondition(reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionTypeInitialized,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	}
}

func getRunningCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionTypeRunning,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

func getErrorCondition(reason errCondition, err error) metav1.Condition {
	return metav1.Condition{
		Type:    ConditionTypeError,
		Status:  metav1.ConditionUnknown,
		Reason:  string(reason),
		Message: err.Error(),
	}
}

func isMarkedForDeletion(o client.Object) bool {
	return !o.GetDeletionTimestamp().IsZero()
}

func addFinalizer(ctx context.Context, client client.Client, o client.Object, f string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		key := types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}
		if err := client.Get(ctx, key, o); err != nil {
			return err
		}

		controllerutil.AddFinalizer(o, f)

		return client.Update(ctx, o)
	})
}

func removeFinalizer(ctx context.Context, client client.Client, o client.Object, f string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		key := types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}
		if err := client.Get(ctx, key, o); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}

			return err
		}

		controllerutil.RemoveFinalizer(o, f)

		return client.Update(ctx, o)
	})
}

// hasLatestGeneration returns true if the client.Object's controller has processed the latest generation of the object.
// Note: This only indicates that the controller has seen and started processing the latest changes.
// It does not mean the object is ready to use, as the controller may still be polling the Aiven API
// waiting for the requested changes to take effect.
func hasLatestGeneration(o client.Object) bool {
	return o.GetAnnotations()[processedGenerationAnnotation] == strconv.FormatInt(o.GetGeneration(), formatIntBaseDecimal)
}

// hasIsRunningAnnotation means the client.Object is running/rebalancing in Aiven or powered-off (for services).
func hasIsRunningAnnotation(o client.Object) bool {
	_, ok := o.GetAnnotations()[instanceIsRunningAnnotation]
	return ok
}

// GetIsRunningAnnotation returns "true" for running/rebalancing resources,
// and "false" for powered-off resources.
func GetIsRunningAnnotation(o client.Object) string {
	return o.GetAnnotations()[instanceIsRunningAnnotation]
}

// IsReadyToUse returns true when the client.Object's controller has processed the latest manifest changes
// and the resource is in a running state in Aiven. For services, this includes both running and powered-off states.
// This indicates the resource is ready for use and has reached its desired state.
func IsReadyToUse(o client.Object) bool {
	return hasIsRunningAnnotation(o) && hasLatestGeneration(o)
}

// NilIfZero returns a pointer to the value, or nil if the value equals its zero value
func NilIfZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}

	return &v
}

// userAgent is a helper function to create a User-Agent string used for the Go client.
func userAgent(kubeVersion, operatorVersion string) string {
	// Remove the leading "v" from the version strings, if present.
	// This is to unify the version format across the Terraform Provider and the Kubernetes Operator.
	//
	// Cf. terraform-provider-aiven/A.B.C/X.Y.Z vs. k8s-operator/vA.B.C/vX.Y.Z.
	kubeVersion = strings.TrimPrefix(kubeVersion, "v")
	operatorVersion = strings.TrimPrefix(operatorVersion, "v")

	return fmt.Sprintf("k8s-operator/%s/%s", kubeVersion, operatorVersion)
}

// NewAivenGeneratedClient returns Aiven generated client client (aiven/go-client-codegen)
func NewAivenGeneratedClient(token, kubeVersion, operatorVersion string) (avngen.Client, error) {
	return avngen.NewClient(avngen.TokenOpt(token), avngen.UserAgentOpt(userAgent(kubeVersion, operatorVersion)))
}

func fromAnyPointer[T any](v *T) T {
	if v != nil {
		return *v
	}
	var t T
	return t
}

type objWithSecret interface {
	GetName() string
	GetNamespace() string
	GetObjectKind() schema.ObjectKind
	GetConnInfoSecretTarget() v1alpha1.ConnInfoSecretTarget
}

func newSecret(o objWithSecret, stringData map[string]string, addPrefix bool) *corev1.Secret {
	target := o.GetConnInfoSecretTarget()
	meta := metav1.ObjectMeta{
		Name:        o.GetName(),
		Namespace:   o.GetNamespace(),
		Annotations: target.Annotations,
		Labels:      target.Labels,
	}

	if target.Name != "" {
		meta.Name = target.Name
	}

	// fixme: set this as default behaviour
	//  when legacy secrets removed
	if addPrefix {
		prefix := getSecretPrefix(o)
		for k, v := range stringData {
			delete(stringData, k)
			stringData[prefix+k] = v
		}
	}

	return &corev1.Secret{
		ObjectMeta: meta,
		StringData: stringData,
	}
}

// getSecretPrefix returns user's prefix or kind name
func getSecretPrefix(o objWithSecret) string {
	target := o.GetConnInfoSecretTarget()
	if target.Prefix != "" {
		return target.Prefix
	}
	kind := o.GetObjectKind()
	return strings.ToUpper(kind.GroupVersionKind().Kind) + "_"
}

// userConfigurationToAPI converts user config into a map
func userConfigurationToAPI(c any, groups ...string) (map[string]any, error) {
	if c == nil || (reflect.ValueOf(c).Kind() == reflect.Ptr && reflect.ValueOf(c).IsNil()) {
		return map[string]any{}, nil
	}

	o := &sheriff.Options{
		Groups: groups,
	}

	i, err := sheriff.Marshal(o, c)
	if err != nil {
		return nil, err
	}

	m, ok := i.(map[string]interface{})
	if !ok {
		// It is an empty pointer
		// sheriff just returned the very same object
		return map[string]any{}, nil
	}

	return m, nil
}

func CreateUserConfiguration(userConfig any) (map[string]any, error) {
	return userConfigurationToAPI(userConfig, "create", "update")
}

func UpdateUserConfiguration(userConfig any) (map[string]any, error) {
	return userConfigurationToAPI(userConfig, "update")
}

// isNotFound works both for old and new client errors
func isNotFound(err error) bool {
	return avngen.IsNotFound(err)
}

// isAlreadyExists works both for old and new client errors
func isAlreadyExists(err error) bool {
	return avngen.IsAlreadyExists(err)
}

func NewNotFound(msg string) error {
	return avngen.Error{Status: http.StatusNotFound, Message: msg}
}

func isDeleted(err error) (bool, error) {
	if isNotFound(err) {
		return true, nil
	}
	return err == nil, err
}

// isAivenError returns true if the error comes from the old or new client and has given http code
func isAivenError(err error, code int) bool {
	var e avngen.Error
	if errors.As(err, &e) {
		return e.Status == code
	}

	return false
}

func isServerError(err error) bool {
	var e avngen.Error
	if errors.As(err, &e) {
		return e.Status >= http.StatusInternalServerError && e.Status < 600
	}
	return false
}

// isRetryableAivenError returns true if the error represents a transient Aiven API failure that should be retried by the reconciler.
//
// Current policy:
// - 404: resource may not be visible yet (eventual consistency).
// - 5xx: server-side issues are considered transient.
// - 403: eventual consistency in IAM / permissions.
func isRetryableAivenError(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case isNotFound(err):
		return true
	case isServerError(err):
		return true
	case isAivenError(err, http.StatusForbidden):
		return true
	default:
		return false
	}
}
