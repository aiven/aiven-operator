// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ServiceIntegrationEndpointReconciler reconciles a ServiceIntegrationEndpoint object
type ServiceIntegrationEndpointReconciler struct {
	Controller
}

func newServiceIntegrationEndpointReconciler(c Controller) reconcilerType {
	return &ServiceIntegrationEndpointReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrationendpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrationendpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrationendpoints/finalizers,verbs=get;create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ServiceIntegrationEndpointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ServiceIntegrationEndpointHandler{}, &v1alpha1.ServiceIntegrationEndpoint{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceIntegrationEndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceIntegrationEndpoint{}).
		Complete(r)
}

type ServiceIntegrationEndpointHandler struct{}

func (h ServiceIntegrationEndpointHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) (bool, error) {
	si, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	userConfig, err := si.GetUserConfig()
	if err != nil {
		return false, err
	}

	exists := si.Status.ID != ""
	if !exists {
		userConfigMap, err := CreateUserConfiguration(userConfig)
		if err != nil {
			return false, err
		}

		endpoint, err := avnGen.ServiceIntegrationEndpointCreate(
			ctx,
			si.Spec.Project,
			&service.ServiceIntegrationEndpointCreateIn{
				EndpointName: si.Spec.EndpointName,
				EndpointType: service.EndpointType(si.Spec.EndpointType),
				UserConfig:   userConfigMap,
			},
		)
		if err != nil {
			return false, fmt.Errorf("cannot service integration: %w", err)
		}

		si.Status.ID = endpoint.EndpointId
	} else {
		if !si.HasUserConfig() {
			return false, nil
		}

		userConfigMap, err := UpdateUserConfiguration(userConfig)
		if err != nil {
			return false, err
		}

		updatedIntegration, err := avnGen.ServiceIntegrationEndpointUpdate(
			ctx,
			si.Spec.Project,
			si.Status.ID,
			&service.ServiceIntegrationEndpointUpdateIn{
				UserConfig: userConfigMap,
			},
		)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "user config not changed") {
				return false, nil
			}
			return false, err
		}
		si.Status.ID = updatedIntegration.EndpointId
	}

	return !exists, nil
}

func (h ServiceIntegrationEndpointHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	si, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	if si.Status.ID == "" {
		return true, nil
	}

	err = avnGen.ServiceIntegrationEndpointDelete(ctx, si.Spec.Project, si.Status.ID)
	if err != nil && !isNotFound(err) {
		return false, fmt.Errorf("aiven client delete service integration endpoint error: %w", err)
	}

	return true, nil
}

func (h ServiceIntegrationEndpointHandler) get(_ context.Context, _ avngen.Client, obj client.Object) (*corev1.Secret, error) {
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

func (h ServiceIntegrationEndpointHandler) checkPreconditions(_ context.Context, _ avngen.Client, obj client.Object) (bool, error) {
	si, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&si.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return true, nil
}

func (h ServiceIntegrationEndpointHandler) convert(i client.Object) (*v1alpha1.ServiceIntegrationEndpoint, error) {
	si, ok := i.(*v1alpha1.ServiceIntegrationEndpoint)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ServiceIntegrationEndpoint")
	}

	return si, nil
}
