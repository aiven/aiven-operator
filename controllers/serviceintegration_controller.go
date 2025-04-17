// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/avast/retry-go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ServiceIntegrationReconciler reconciles a ServiceIntegration object
type ServiceIntegrationReconciler struct {
	Controller
}

func newServiceIntegrationReconciler(c Controller) reconcilerType {
	return &ServiceIntegrationReconciler{Controller: c}
}

type ServiceIntegrationHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations/finalizers,verbs=get;create;update

func (r *ServiceIntegrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ServiceIntegrationHandler{}, &v1alpha1.ServiceIntegration{})
}

func (r *ServiceIntegrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceIntegration{}).
		Complete(r)
}

func (h ServiceIntegrationHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	si, err := h.convert(obj)
	if err != nil {
		return err
	}

	userConfig, err := si.GetUserConfig()
	if err != nil {
		return err
	}

	var reason string
	if si.Status.ID == "" {
		userConfigMap, err := CreateUserConfiguration(userConfig)
		if err != nil {
			return err
		}

		integration, err := avnGen.ServiceIntegrationCreate(
			ctx,
			si.Spec.Project,
			&service.ServiceIntegrationCreateIn{
				DestEndpointId:   toPtr(si.Spec.DestinationEndpointID),
				DestService:      toPtr(si.Spec.DestinationServiceName),
				DestProject:      toPtr(si.Spec.DestinationProjectName),
				IntegrationType:  si.Spec.IntegrationType,
				SourceEndpointId: toPtr(si.Spec.SourceEndpointID),
				SourceService:    toPtr(si.Spec.SourceServiceName),
				SourceProject:    toPtr(si.Spec.SourceProjectName),
				UserConfig:       &userConfigMap,
			},
		)
		if err != nil {
			return fmt.Errorf("cannot createOrUpdate service integration: %w", err)
		}

		reason = "Created"
		si.Status.ID = integration.ServiceIntegrationId
	} else {
		if !si.HasUserConfig() {
			return nil
		}

		userConfigMap, err := UpdateUserConfiguration(userConfig)
		if err != nil {
			return err
		}

		var updatedIntegration *service.ServiceIntegrationUpdateOut
		err = retry.Do(
			func() error {
				var updateErr error
				updatedIntegration, updateErr = avnGen.ServiceIntegrationUpdate(
					ctx,
					si.Spec.Project,
					si.Status.ID,
					&service.ServiceIntegrationUpdateIn{
						UserConfig: userConfigMap,
					},
				)

				return updateErr
			},
			retry.RetryIf(isNotFound),
			retry.Attempts(3), //nolint:mnd
			retry.Delay(1*time.Second),
		)

		reason = "Updated"
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "user config not changed") {
				return nil
			}
			return err
		}
		si.Status.ID = updatedIntegration.ServiceIntegrationId
	}

	meta.SetStatusCondition(&si.Status.Conditions,
		getInitializedCondition(reason,
			"Successfully created or updated the instance in Aiven"))

	meta.SetStatusCondition(&si.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Successfully created or updated the instance in Aiven, status remains unknown"))

	metav1.SetMetaDataAnnotation(&si.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(si.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h ServiceIntegrationHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	si, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	if si.Status.ID == "" {
		return true, nil
	}

	err = avnGen.ServiceIntegrationDelete(ctx, si.Spec.Project, si.Status.ID)
	if err != nil && !isNotFound(err) {
		return false, fmt.Errorf("aiven client delete service integration error: %w", err)
	}

	return true, nil
}

func (h ServiceIntegrationHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	si, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&si.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&si.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h ServiceIntegrationHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	si, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&si.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	// todo: validate SourceEndpointID, DestinationEndpointID when ServiceIntegrationEndpoint kind released

	if si.Spec.SourceServiceName != "" {
		project := si.Spec.SourceProjectName
		if project == "" {
			project = si.Spec.Project
		}
		running, err := checkServiceIsOperational(ctx, avnGen, project, si.Spec.SourceServiceName)
		if !running || err != nil {
			return false, err
		}
	}

	if si.Spec.DestinationServiceName != "" {
		project := si.Spec.DestinationProjectName
		if project == "" {
			project = si.Spec.Project
		}
		running, err := checkServiceIsOperational(ctx, avnGen, project, si.Spec.DestinationServiceName)
		if !running || err != nil {
			return false, err
		}
	}

	return true, nil
}

func (h ServiceIntegrationHandler) convert(i client.Object) (*v1alpha1.ServiceIntegration, error) {
	si, ok := i.(*v1alpha1.ServiceIntegration)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ServiceIntegration")
	}

	return si, nil
}
