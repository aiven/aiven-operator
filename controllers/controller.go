package controllers

import (
	"fmt"
	"strings"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// formatIntBaseDecimal it is a base to format int64 to string
const formatIntBaseDecimal = 10

var errNoTokenProvided = fmt.Errorf("no Aiven API token available: authSecretRef is not set and no DEFAULT_AIVEN_TOKEN configured")

type (
	// Controller reconciles the Aiven objects
	Controller struct {
		client.Client

		Log             logr.Logger
		Scheme          *runtime.Scheme
		Recorder        record.EventRecorder
		DefaultToken    string
		KubeVersion     string
		OperatorVersion string
		PollInterval    time.Duration
		newAivenClient  func(token, kubeVersion, operatorVersion string) (avngen.Client, error)

		// EnabledKinds are the kinds selected by --controllers.
		EnabledKinds kindSet
	}
)

const (
	// Lifecycle event types we expose to the user
	eventUnableToGetAuthSecret              = "UnableToGetAuthSecret"
	eventUnableToCreateClient               = "UnableToCreateClient"
	eventTryingToDeleteAtAiven              = "TryingToDeleteAtAiven"
	eventUnableToDeleteAtAiven              = "UnableToDeleteAtAiven"
	eventUnableToDeleteFinalizer            = "UnableToDeleteFinalizer"
	eventUnableToDelete                     = "UnableToDelete"
	eventSuccessfullyDeletedAtAiven         = "SuccessfullyDeletedAtAiven"
	eventAddedFinalizer                     = "InstanceFinalizerAdded"
	eventWaitingForPreconditions            = "WaitingForPreconditions"
	eventUnableToWaitForPreconditions       = "UnableToWaitForPreconditions"
	eventPreconditionsNotMet                = "PreconditionsNotMet"
	eventUnableToCreateOrUpdateAtAiven      = "UnableToCreateOrUpdateAtAiven"
	eventCreateOrUpdatedAtAiven             = "CreateOrUpdatedAtAiven"
	eventCreatedOrUpdatedAtAiven            = "CreatedOrUpdatedAtAiven"
	eventWaitingForTheInstanceToBeRunning   = "WaitingForInstanceToBeRunning"
	eventUnableToWaitForInstanceToBeRunning = "UnableToWaitForInstanceToBeRunning"
	eventConnInfoSecretCreationDisabled     = "ConnInfoSecretCreationDisabled"
	eventCannotPublishConnectionDetails     = "CannotPublishConnectionDetails"
	eventInstanceMissingAtAiven             = "InstanceMissingAtAiven"
	eventSpecChangeIgnored                  = "SpecChangeIgnored"
	eventManagementPolicyNotSupported       = "ManagementPolicyNotSupported"
	eventDriftDetected                      = "DriftDetected"
	eventDiskSpaceNotApplied                = "DiskSpaceNotApplied"
)

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// hasPendingMigration returns true when migration is explicitly in progress.
func hasPendingMigration(o v1alpha1.AivenManagedObject) bool {
	cond := meta.FindStatusCondition(*o.Conditions(), v1alpha1.ConditionTypeMigrationComplete)
	return cond != nil && cond.Reason == v1alpha1.MigrationReasonInProgress
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
