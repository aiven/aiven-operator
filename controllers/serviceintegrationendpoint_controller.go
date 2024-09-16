// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aiven/aiven-go-client/v2"
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

func (h ServiceIntegrationEndpointHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
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
			return fmt.Errorf("cannot createOrUpdate service integration: %w", err)
		}

		reason = "Created"
		si.Status.ID = endpoint.EndpointId
	} else {
		if !si.HasUserConfig() {
			return nil
		}

		userConfigMap, err := UpdateUserConfiguration(userConfig)
		if err != nil {
			return err
		}

		updatedIntegration, err := avnGen.ServiceIntegrationEndpointUpdate(
			ctx,
			si.Spec.Project,
			si.Status.ID,
			&service.ServiceIntegrationEndpointUpdateIn{
				UserConfig: userConfigMap,
			},
		)
		reason = "Updated"
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "user config not changed") {
				return nil
			}
			return err
		}
		si.Status.ID = updatedIntegration.EndpointId
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

func (h ServiceIntegrationEndpointHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
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

func (h ServiceIntegrationEndpointHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
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

func (h ServiceIntegrationEndpointHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
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
