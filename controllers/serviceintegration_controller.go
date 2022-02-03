// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ServiceIntegrationReconciler reconciles a ServiceIntegration object
type ServiceIntegrationReconciler struct {
	Controller
}

type ServiceIntegrationHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations/status,verbs=get;update;patch

func (r *ServiceIntegrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ServiceIntegrationHandler{}, &v1alpha1.ServiceIntegration{})
}

func (r *ServiceIntegrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceIntegration{}).
		Complete(r)
}

func (h ServiceIntegrationHandler) createOrUpdate(avn *aiven.Client, i client.Object) error {
	si, err := h.convert(i)
	if err != nil {
		return err
	}

	var integration *aiven.ServiceIntegration

	var reason string
	if si.Status.ID == "" {
		integration, err = avn.ServiceIntegrations.Create(
			si.Spec.Project,
			aiven.CreateServiceIntegrationRequest{
				DestinationEndpointID: toOptionalStringPointer(si.Spec.DestinationEndpointID),
				DestinationService:    toOptionalStringPointer(si.Spec.DestinationServiceName),
				IntegrationType:       si.Spec.IntegrationType,
				SourceEndpointID:      toOptionalStringPointer(si.Spec.SourceEndpointID),
				SourceService:         toOptionalStringPointer(si.Spec.SourceServiceName),
				UserConfig:            h.getUserConfig(si),
			},
		)
		if err != nil {
			return fmt.Errorf("cannot createOrUpdate service integration: %w", err)
		}

		reason = "Created"
	} else {
		integration, err = avn.ServiceIntegrations.Update(
			si.Spec.Project,
			si.Status.ID,
			aiven.UpdateServiceIntegrationRequest{
				UserConfig: h.getUserConfig(si),
			},
		)
		reason = "Updated"
		if err != nil {
			if strings.Contains(err.Error(), "user config not changed") {
				return nil
			}
			return err
		}
	}

	si.Status.ID = integration.ServiceIntegrationID

	meta.SetStatusCondition(&si.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&si.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&si.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(si.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h ServiceIntegrationHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	si, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = avn.ServiceIntegrations.Delete(si.Spec.Project, si.Status.ID)
	if err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("aiven client delete service ingtegration error: %w", err)
	}

	return true, nil
}

func (h ServiceIntegrationHandler) get(_ *aiven.Client, i client.Object) (*corev1.Secret, error) {
	si, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&si.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&si.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h ServiceIntegrationHandler) checkPreconditions(avn *aiven.Client, i client.Object) (bool, error) {
	si, err := h.convert(i)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&si.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	sourceCheck, err := checkServiceIsRunning(avn, si.Spec.Project, si.Spec.SourceServiceName)
	if err != nil {
		return false, err
	}

	destinationCheck, err := checkServiceIsRunning(avn, si.Spec.Project, si.Spec.DestinationServiceName)
	if err != nil {
		return false, err
	}

	return sourceCheck && destinationCheck, nil
}

func (h ServiceIntegrationHandler) convert(i client.Object) (*v1alpha1.ServiceIntegration, error) {
	si, ok := i.(*v1alpha1.ServiceIntegration)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ServiceIntegration")
	}

	return si, nil
}

func (h ServiceIntegrationHandler) getUserConfig(int *v1alpha1.ServiceIntegration) map[string]interface{} {
	if int.Spec.IntegrationType == "datadog" {
		return UserConfigurationToAPI(int.Spec.DatadogUserConfig).(map[string]interface{})
	}
	if int.Spec.IntegrationType == "kafka_connect" {
		return UserConfigurationToAPI(int.Spec.KafkaConnectUserConfig).(map[string]interface{})
	}
	if int.Spec.IntegrationType == "kafka_logs" {
		return UserConfigurationToAPI(int.Spec.KafkaLogsUserConfig).(map[string]interface{})
	}
	if int.Spec.IntegrationType == "metrics" {
		return UserConfigurationToAPI(int.Spec.MetricsUserConfig).(map[string]interface{})
	}

	return nil
}
