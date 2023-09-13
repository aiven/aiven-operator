package controllers

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/liip/sheriff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-go-client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const (
	conditionTypeRunning     = "Running"
	conditionTypeInitialized = "Initialized"

	secretProtectionFinalizer = "finalizers.aiven.io/needed-to-delete-services"
	instanceDeletionFinalizer = "finalizers.aiven.io/delete-remote-resource"

	processedGenerationAnnotation = "controllers.aiven.io/generation-was-processed"
	instanceIsRunningAnnotation   = "controllers.aiven.io/instance-is-running"
)

var (
	version                    = "dev"
	errTerminationProtectionOn = errors.New("termination protection is on")
)

func checkServiceIsRunning(c *aiven.Client, project, serviceName string) (bool, error) {
	s, err := c.Services.Get(project, serviceName)
	if err != nil {
		// if service is not found, it is not running
		if aiven.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return s.State == "RUNNING", nil
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

func isMarkedForDeletion(o client.Object) bool {
	return !o.GetDeletionTimestamp().IsZero()
}

func addFinalizer(ctx context.Context, client client.Client, o client.Object, f string) error {
	controllerutil.AddFinalizer(o, f)
	return client.Update(ctx, o)
}

func removeFinalizer(ctx context.Context, client client.Client, o client.Object, f string) error {
	controllerutil.RemoveFinalizer(o, f)
	return client.Update(ctx, o)
}

func isAlreadyProcessed(o client.Object) bool {
	return o.GetAnnotations()[processedGenerationAnnotation] == strconv.FormatInt(o.GetGeneration(), formatIntBaseDecimal)
}

// IsAlreadyRunning returns true if object is ready to use
func IsAlreadyRunning(o client.Object) bool {
	_, found := o.GetAnnotations()[instanceIsRunningAnnotation]
	return found
}

func optionalStringPointer(u string) *string {
	if len(u) == 0 {
		return nil
	}

	return &u
}

func isAivenServerError(err error) bool {
	e, ok := err.(aiven.Error)
	return ok && e.Status >= http.StatusInternalServerError
}

// NewAivenClient returns Aiven client
func NewAivenClient(token string) (*aiven.Client, error) {
	return aiven.NewTokenClient(token, "k8s-operator/"+version)
}

func fromAnyPointer[T any](v *T) T {
	if v != nil {
		return *v
	}
	var t T
	return t
}

func anyOptional[T comparable](v T) *T {
	var zero T
	if zero == v {
		return nil
	}
	return &v
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
		return nil, nil
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
		return nil, nil
	}

	return m, nil
}

func CreateUserConfiguration(userConfig any) (map[string]any, error) {
	return userConfigurationToAPI(userConfig, "create", "update")
}

func UpdateUserConfiguration(userConfig any) (map[string]any, error) {
	return userConfigurationToAPI(userConfig, "update")
}
