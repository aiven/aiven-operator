// Copyright (c) 2025 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const (
	defaultMaxConcurrent = 1
	defaultRequeueDelay  = 10 * time.Second
	defaultResyncPeriod  = 1 * time.Hour

	// Condition reasons for service states
	reasonServiceRunning    = "ServiceRunning"
	reasonServicePoweredOff = "ServicePoweredOff"
	reasonCreatedOrUpdated  = "CreatedOrUpdated"
)

type KafkaControllerV2 struct {
	client.Client

	avnClient avngen.Client
	handler   Handlers

	log      logr.Logger
	recorder record.EventRecorder

	conditionEvaluator *ConditionEvaluator

	defaultToken    string
	kubeVersion     string
	operatorVersion string

	maxConcurrent int
}

func NewKafkaControllerV2(mgr ctrl.Manager, defaultToken, kubeVersion, operatorVersion string) (*KafkaControllerV2, error) {
	var (
		log       = ctrl.Log.WithName("kafka-controller")
		avnClient avngen.Client
		err       error
	)

	if defaultToken != "" {
		avnClient, err = NewAivenGeneratedClient(defaultToken, kubeVersion, operatorVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to create Aiven client with default token: %w", err)
		}
		log.Info("initialized controller with default Aiven client")
	}

	return &KafkaControllerV2{
		Client:    mgr.GetClient(),
		avnClient: avnClient, // nil if no default token
		handler:   newGenericServiceHandler(newKafkaAdapter, log),
		log:       log,
		recorder:  mgr.GetEventRecorderFor("kafka-v2"),
		conditionEvaluator: NewConditionEvaluator(
			CheckGenerationChanged,
			CheckRunningStatus,
			CheckTriggerAnnotations,
		),
		defaultToken:    defaultToken,
		kubeVersion:     kubeVersion,
		operatorVersion: operatorVersion,
		maxConcurrent:   defaultMaxConcurrent,
	}, nil
}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkas/finalizers,verbs=get;create;update

func (r *KafkaControllerV2) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	kafka := &v1alpha1.Kafka{}
	if err := r.Get(ctx, req.NamespacedName, kafka); err != nil {
		if client.IgnoreNotFound(err) == nil {
			r.log.Info("kafka resource not found, skipping reconciliation", "resource", req.NamespacedName)
			return ctrl.Result{}, nil
		}

		r.log.Error(err, "failed to fetch kafka resource", "resource", req.NamespacedName)

		return ctrl.Result{}, fmt.Errorf("failed to fetch kafka resource %s: %w", req.NamespacedName, err)
	}

	r.log.Info("reconciling Kafka", "kafka", req.NamespacedName)

	statusUpdate := NewStatusUpdate()

	// handle deletion
	if kafka.GetDeletionTimestamp() != nil {
		r.log.Info("handling resource deletion", "kafka", kafka.Name)
		return r.finalize(ctx, kafka)
	}

	// check if we may skip reconciliation
	skip, reason, condErr := r.conditionEvaluator.ShouldSkip(kafka)
	if condErr != nil {
		r.log.Error(condErr, "failed to evaluate conditions", "kafka", kafka.Name, "reason", reason)

		return requeue(defaultRequeueDelay), fmt.Errorf("failed to evaluate conditions for kafka %s: %w", kafka.Name, condErr)
	}

	if skip {
		r.log.Info(
			"skipping reconciliation",
			"kafka",
			kafka.Name,
			"reason",
			reason,
			"generation",
			kafka.Generation,
		)

		return requeue(defaultResyncPeriod), nil
	}

	r.log.Info("reconciliation needed", "kafka", kafka.Name, "reason", reason)

	avnClient, err := r.getAvnClient(ctx, kafka)
	if err != nil {
		r.log.Error(err, "failed to get Aiven client", "kafka", kafka.Name)
		r.recorder.Event(kafka, corev1.EventTypeWarning, eventUnableToCreateClient, err.Error())
		return requeue(defaultRequeueDelay), fmt.Errorf("failed to get Aiven client: %w", err)
	}

	// prevent premature deletion
	if err = addDeletionProtection(ctx, r.Client, kafka); err != nil {
		r.log.Error(err, "failed to add finalizer", "kafka", kafka.Name)
		return requeue(defaultRequeueDelay), fmt.Errorf("failed to add finalizer: %w", err)
	}

	// check prerequisites: service integrations, power-off validation, etc.
	canProceed, err := r.handler.checkPreconditions(ctx, avnClient, kafka)
	if err != nil {
		r.log.Error(err, "precondition check failed", "kafka", kafka.Name)
		statusUpdate.SetCondition(getErrorCondition(errConditionPreconditions, err))
		if commitErr := CommitStatusUpdates(ctx, r.Client, kafka, statusUpdate); commitErr != nil {
			r.log.Error(commitErr, "failed to commit status updates for precondition error", "kafka", kafka.Name)
		}
		return requeue(defaultRequeueDelay), fmt.Errorf("precondition check failed: %w", err)
	}

	if !canProceed {
		r.log.Info("preconditions not met, requeuing", "kafka", kafka.Name)
		return requeue(defaultRequeueDelay), nil
	}

	// for now only call createOrUpdate if generation changed to avoid redundant calls during provisioning.
	// can be extended to check drift detection, etc. in the future.
	if !hasLatestGeneration(kafka) {
		r.log.Info("generation changed, creating or updating service", "kafka", kafka.Name,
			"generation", kafka.GetGeneration(), "processedGeneration", kafka.GetAnnotations()[processedGenerationAnnotation])

		if err = r.handler.createOrUpdate(ctx, avnClient, kafka, nil); err != nil {
			r.log.Error(err, "failed to create or update service", "kafka", kafka.Name)
			r.recorder.Event(kafka, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
			statusUpdate.SetCondition(getErrorCondition(errConditionCreateOrUpdate, err))
			if commitErr := CommitStatusUpdates(ctx, r.Client, kafka, statusUpdate); commitErr != nil {
				r.log.Error(commitErr, "failed to commit status updates for create/update error", "kafka", kafka.Name)
			}
			return requeue(defaultRequeueDelay), fmt.Errorf("failed to create or update service: %w", err)
		}

		statusUpdate.SetCondition(getInitializedCondition(reasonCreatedOrUpdated, "Successfully created or updated the instance in Aiven"))
		statusUpdate.SetCondition(getRunningCondition(metav1.ConditionUnknown, reasonCreatedOrUpdated, "Successfully created or updated the instance in Aiven, status remains unknown"))
	}

	secret, err := r.handler.get(ctx, avnClient, kafka)
	if err != nil {
		r.log.Error(err, "failed to get service status", "kafka", kafka.Name)
		return requeue(defaultRequeueDelay), fmt.Errorf("failed to get service status: %w", err)
	}

	// if secret is nil, service is still provisioning or powered off
	if secret == nil {
		return r.handleProvisioning(ctx, kafka, statusUpdate)
	}

	if !kafka.NoSecret() {
		if err = r.createOrUpdateSecret(ctx, kafka, secret); err != nil {
			r.log.Error(err, "failed to create or update connection secret", "kafka", kafka.Name)
			return requeue(defaultRequeueDelay), fmt.Errorf("failed to create or update connection secret: %w", err)
		}
	}

	// extract annotations and status fields that were set by handler.get()
	runningAnnotationValue := kafka.GetAnnotations()[instanceIsRunningAnnotation]
	if runningAnnotationValue != "" {
		statusUpdate.SetAnnotation(instanceIsRunningAnnotation, runningAnnotationValue)
	}

	if kafka.Status.State != "" {
		statusUpdate.SetStatusField("state", kafka.Status.State)
	}

	switch runningAnnotationValue {
	case "true":
		statusUpdate.SetCondition(getRunningCondition(metav1.ConditionTrue, reasonServiceRunning, "Service is running in Aiven"))
	case "false":
		statusUpdate.SetCondition(getRunningCondition(metav1.ConditionTrue, reasonServicePoweredOff, "Service is powered off in Aiven"))
	}

	statusUpdate.SetProcessedGeneration(kafka.GetGeneration())

	if err = CommitStatusUpdates(ctx, r.Client, kafka, statusUpdate); err != nil {
		r.log.Error(err, "failed to commit status updates", "kafka", kafka.Name)
		return requeue(defaultRequeueDelay), fmt.Errorf("failed to commit status updates: %w", err)
	}

	r.log.Info("reconciliation completed successfully", "kafka", kafka.Name)

	return requeue(defaultResyncPeriod), nil
}

func (r *KafkaControllerV2) getAvnClient(ctx context.Context, kafka *v1alpha1.Kafka) (avngen.Client, error) {
	if r.avnClient != nil && kafka.AuthSecretRef() == nil {
		return r.avnClient, nil
	}

	return createAivenClient(ctx, kafka, r.Client, r.defaultToken, r.kubeVersion, r.operatorVersion)
}

// finalize handles resource deletion
func (r *KafkaControllerV2) finalize(ctx context.Context, kafka *v1alpha1.Kafka) (ctrl.Result, error) {
	if policy, exists := kafka.GetAnnotations()[deletionPolicyAnnotation]; exists && policy == deletionPolicyOrphan {
		r.log.Info("orphan deletion policy, skipping Aiven cleanup", "kafka", kafka.Name)
		return r.removeDeletionProtection(ctx, kafka)
	}

	avnClient, err := r.getAvnClient(ctx, kafka)
	if err != nil {
		return requeue(defaultRequeueDelay), fmt.Errorf("failed to get Aiven client for deletion: %w", err)
	}

	err = avnClient.ServiceDelete(ctx, kafka.Spec.Project, kafka.GetName())
	if err != nil && !isNotFound(err) {
		r.log.Error(err, "failed to delete service from Aiven", "kafka", kafka.Name)
		r.recorder.Event(kafka, corev1.EventTypeWarning, eventUnableToDeleteAtAiven, err.Error())

		statusUpdate := NewStatusUpdate()
		statusUpdate.SetCondition(getErrorCondition(errConditionDelete, err))
		if commitErr := CommitStatusUpdates(ctx, r.Client, kafka, statusUpdate); commitErr != nil {
			r.log.Error(commitErr, "failed to commit status updates for deletion error", "kafka", kafka.Name)
		}

		return requeue(defaultRequeueDelay), fmt.Errorf("failed to delete service: %w", err)
	}

	r.log.Info("service deleted from Aiven", "kafka", kafka.Name)

	return r.removeDeletionProtection(ctx, kafka)
}

// removeDeletionProtection removes the deletion finalizer
func (r *KafkaControllerV2) removeDeletionProtection(ctx context.Context, kafka *v1alpha1.Kafka) (ctrl.Result, error) {
	if err := removeDeletionProtection(ctx, r.Client, kafka); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
	}

	r.recorder.Event(kafka, corev1.EventTypeNormal, eventSuccessfullyDeletedAtAiven, "Kafka service deleted successfully")
	r.log.Info("finalizer removed, deletion completed", "kafka", kafka.Name)

	return ctrl.Result{}, nil
}

// createOrUpdateSecret upserts the connection secret for Kafka
func (r *KafkaControllerV2) createOrUpdateSecret(ctx context.Context, kafka *v1alpha1.Kafka, secret *corev1.Secret) error {
	if secret == nil {
		return nil
	}

	if err := controllerutil.SetControllerReference(kafka, secret, r.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	if err := r.Create(ctx, secret); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}

		existing := &corev1.Secret{}
		if err = r.Get(ctx, client.ObjectKeyFromObject(secret), existing); err != nil {
			return fmt.Errorf("failed to get existing secret: %w", err)
		}

		existing.Data = secret.Data
		existing.StringData = secret.StringData
		if err = r.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update secret: %w", err)
		}

		return nil
	}

	r.log.Info("created connection secret", "kafka", kafka.Name, "secret", secret.Name)

	return nil
}

// handleProvisioning handles the case where service is still provisioning or powered off
func (r *KafkaControllerV2) handleProvisioning(ctx context.Context, kafka *v1alpha1.Kafka, statusUpdate *StatusUpdate) (ctrl.Result, error) {
	// check if service is explicitly powered off
	runningAnnotationValue := kafka.GetAnnotations()[instanceIsRunningAnnotation]
	if runningAnnotationValue == "false" {
		r.log.Info("service is powered off", "kafka", kafka.Name)

		statusUpdate.SetProcessedGeneration(kafka.GetGeneration())
		statusUpdate.SetAnnotation(instanceIsRunningAnnotation, "false")
		statusUpdate.SetCondition(getRunningCondition(metav1.ConditionTrue, reasonServicePoweredOff, "Service is powered off in Aiven"))

		if err := CommitStatusUpdates(ctx, r.Client, kafka, statusUpdate); err != nil {
			r.log.Error(err, "failed to commit status updates for powered-off service", "kafka", kafka.Name)
			return requeue(defaultRequeueDelay), fmt.Errorf("failed to commit status updates: %w", err)
		}

		return ctrl.Result{RequeueAfter: defaultResyncPeriod}, nil
	}

	r.log.Info("service still provisioning, will requeue",
		"kafka", kafka.Name, "generation", kafka.GetGeneration(), "state", kafka.Status.State)

	statusUpdate.SetProcessedGeneration(kafka.GetGeneration())

	if err := CommitStatusUpdates(ctx, r.Client, kafka, statusUpdate); err != nil {
		r.log.Error(err, "failed to commit status updates during provisioning", "kafka", kafka.Name)
		return requeue(defaultRequeueDelay), fmt.Errorf("failed to commit status updates: %w", err)
	}

	// requeue to check service status again
	return requeue(defaultRequeueDelay), nil
}

// SetupWithManager sets up the controller with the manager
func (r *KafkaControllerV2) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Kafka{}).
		Owns(&corev1.Secret{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.maxConcurrent,
		}).
		Complete(r)
}
