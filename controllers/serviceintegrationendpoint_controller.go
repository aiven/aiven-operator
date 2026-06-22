// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func newServiceIntegrationEndpointReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.ServiceIntegrationEndpoint] {
			return &ServiceIntegrationEndpointController{
				Client: c.Client,
				avnGen: avnGen,
			}
		},
		nil,
	)
}

//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrationendpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrationendpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrationendpoints/finalizers,verbs=get;create;update

// ServiceIntegrationEndpointController reconciles a ServiceIntegrationEndpoint object
type ServiceIntegrationEndpointController struct {
	client.Client
	avnGen avngen.Client
}

func (r *ServiceIntegrationEndpointController) Observe(ctx context.Context, si *v1alpha1.ServiceIntegrationEndpoint) (Observation, error) {
	if si.Status.ID == "" {
		// Status.ID may be empty because the endpoint was never created, or a prior Create status write was lost.
		existing, err := r.findEndpoint(ctx, si)
		if err != nil {
			return Observation{}, err
		}
		if existing == nil {
			return Observation{ResourceExists: false}, nil
		}

		// adoption
		si.Status.ID = existing.EndpointId
		return Observation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	_, err := r.avnGen.ServiceIntegrationEndpointGet(ctx, si.Spec.Project, si.Status.ID)
	if err != nil {
		if isNotFound(err) {
			si.Status.ID = ""
			return Observation{ResourceExists: false}, nil
		}
		return Observation{}, fmt.Errorf("getting service integration endpoint: %w", err)
	}

	markInstanceRunning(si)

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: IsReadyToUse(si),
	}, nil
}

func (r *ServiceIntegrationEndpointController) Create(ctx context.Context, si *v1alpha1.ServiceIntegrationEndpoint) (CreateResult, error) {
	delete(si.GetAnnotations(), instanceIsRunningAnnotation)

	userConfig, err := si.GetUserConfig()
	if err != nil {
		return CreateResult{}, err
	}

	userConfigMap, err := CreateUserConfiguration(userConfig)
	if err != nil {
		return CreateResult{}, err
	}

	endpoint, err := r.avnGen.ServiceIntegrationEndpointCreate(
		ctx,
		si.Spec.Project,
		&service.ServiceIntegrationEndpointCreateIn{
			EndpointName: si.Spec.EndpointName,
			EndpointType: service.EndpointType(si.Spec.EndpointType),
			UserConfig:   userConfigMap,
		},
	)
	if err != nil {
		return CreateResult{}, fmt.Errorf("creating service integration endpoint: %w", err)
	}

	si.Status.ID = endpoint.EndpointId
	markInstanceRunning(si)

	return CreateResult{}, nil
}

func (r *ServiceIntegrationEndpointController) Update(ctx context.Context, si *v1alpha1.ServiceIntegrationEndpoint) (UpdateResult, error) {
	delete(si.GetAnnotations(), instanceIsRunningAnnotation)

	if !si.HasUserConfig() {
		markInstanceRunning(si)
		return UpdateResult{}, nil
	}

	userConfig, err := si.GetUserConfig()
	if err != nil {
		return UpdateResult{}, err
	}

	userConfigMap, err := UpdateUserConfiguration(userConfig)
	if err != nil {
		return UpdateResult{}, err
	}

	updatedEndpoint, err := r.avnGen.ServiceIntegrationEndpointUpdate(
		ctx,
		si.Spec.Project,
		si.Status.ID,
		&service.ServiceIntegrationEndpointUpdateIn{
			UserConfig: userConfigMap,
		},
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "user config not changed") {
			markInstanceRunning(si)
			return UpdateResult{}, nil
		}
		return UpdateResult{}, err
	}

	if updatedEndpoint != nil && updatedEndpoint.EndpointId != "" {
		si.Status.ID = updatedEndpoint.EndpointId
	}
	markInstanceRunning(si)

	return UpdateResult{}, nil
}

func (r *ServiceIntegrationEndpointController) Delete(ctx context.Context, si *v1alpha1.ServiceIntegrationEndpoint) error {
	if si.Status.ID == "" {
		return nil
	}

	err := r.avnGen.ServiceIntegrationEndpointDelete(ctx, si.Spec.Project, si.Status.ID)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting service integration endpoint: %w", err)
	}

	return nil
}

// findEndpoint looks up an endpoint matching the desired spec.
// Aiven guarantees uniqueness on (project, endpoint_name, endpoint_type).
func (r *ServiceIntegrationEndpointController) findEndpoint(
	ctx context.Context,
	si *v1alpha1.ServiceIntegrationEndpoint,
) (*service.ServiceIntegrationEndpointOut, error) {
	endpoints, err := r.avnGen.ServiceIntegrationEndpointList(ctx, si.Spec.Project)
	if err != nil {
		return nil, fmt.Errorf("listing service integration endpoints: %w", err)
	}

	for i := range endpoints {
		e := &endpoints[i]
		if e.EndpointName == si.Spec.EndpointName && e.EndpointType == service.EndpointType(si.Spec.EndpointType) {
			return e, nil
		}
	}

	return nil, nil
}
