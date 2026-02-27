// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/avast/retry-go"
	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func newServiceIntegrationReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.ServiceIntegration] {
			return &ServiceIntegrationController{
				Client: c.Client,
				avnGen: avnGen,
			}
		},
		nil,
	)
}

//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations/finalizers,verbs=get;create;update
//+kubebuilder:rbac:groups=aiven.io,resources=serviceintegrationendpoints,verbs=get;list;watch

// ServiceIntegrationController reconciles a ServiceIntegration object
type ServiceIntegrationController struct {
	client.Client
	avnGen avngen.Client
}

func (r *ServiceIntegrationController) Observe(ctx context.Context, si *v1alpha1.ServiceIntegration) (Observation, error) {
	if err := r.checkPreconditions(ctx, si); err != nil {
		return Observation{}, err
	}

	if si.Status.ID == "" {
		destEndpointID, err := r.resolveDestinationEndpointID(ctx, si)
		if err != nil {
			return Observation{}, err
		}

		existingIntegration, err := r.findExistingIntegration(ctx, si, destEndpointID)
		if err != nil {
			return Observation{}, fmt.Errorf("checking for existing integration: %w", err)
		}
		if existingIntegration == nil {
			return Observation{ResourceExists: false}, nil
		}

		// Adopt existing. Mark as not up-to-date so Update() runs and applies user config (if any) and sets READY metadata.
		si.Status.ID = existingIntegration.ServiceIntegrationId
		return Observation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	_, err := r.avnGen.ServiceIntegrationGet(ctx, si.Spec.Project, si.Status.ID)
	if err != nil {
		if isNotFound(err) {
			si.Status.ID = ""
			return Observation{ResourceExists: false}, nil
		}
		return Observation{}, fmt.Errorf("getting service integration: %w", err)
	}

	meta.SetStatusCondition(&si.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&si.ObjectMeta, instanceIsRunningAnnotation, "true")

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: IsReadyToUse(si),
	}, nil
}

func (r *ServiceIntegrationController) Create(ctx context.Context, si *v1alpha1.ServiceIntegration) (CreateResult, error) {
	userConfig, err := si.GetUserConfig()
	if err != nil {
		return CreateResult{}, err
	}

	destEndpointID, err := r.resolveDestinationEndpointID(ctx, si)
	if err != nil {
		return CreateResult{}, err
	}

	userConfigMap, err := CreateUserConfiguration(userConfig)
	if err != nil {
		return CreateResult{}, err
	}

	integration, err := r.avnGen.ServiceIntegrationCreate(
		ctx,
		si.Spec.Project,
		&service.ServiceIntegrationCreateIn{
			DestEndpointId:   NilIfZero(destEndpointID),
			DestService:      NilIfZero(si.Spec.DestinationServiceName),
			DestProject:      NilIfZero(si.Spec.DestinationProjectName),
			IntegrationType:  si.Spec.IntegrationType,
			SourceEndpointId: NilIfZero(si.Spec.SourceEndpointID),
			SourceService:    NilIfZero(si.Spec.SourceServiceName),
			SourceProject:    NilIfZero(si.Spec.SourceProjectName),
			UserConfig:       &userConfigMap,
		},
	)
	if err != nil {
		return CreateResult{}, fmt.Errorf("creating service integration: %w", err)
	}

	si.Status.ID = integration.ServiceIntegrationId
	meta.SetStatusCondition(&si.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&si.ObjectMeta, instanceIsRunningAnnotation, "true")

	return CreateResult{}, nil
}

func (r *ServiceIntegrationController) Update(ctx context.Context, si *v1alpha1.ServiceIntegration) (UpdateResult, error) {
	if !si.HasUserConfig() {
		meta.SetStatusCondition(&si.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
		metav1.SetMetaDataAnnotation(&si.ObjectMeta, instanceIsRunningAnnotation, "true")
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

	var updatedIntegration *service.ServiceIntegrationUpdateOut
	err = retry.Do(
		func() error {
			var updateErr error
			updatedIntegration, updateErr = r.avnGen.ServiceIntegrationUpdate(
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
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "user config not changed") {
			meta.SetStatusCondition(&si.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
			metav1.SetMetaDataAnnotation(&si.ObjectMeta, instanceIsRunningAnnotation, "true")
			return UpdateResult{}, nil
		}
		return UpdateResult{}, err
	}

	if updatedIntegration != nil && updatedIntegration.ServiceIntegrationId != "" {
		si.Status.ID = updatedIntegration.ServiceIntegrationId
	}

	meta.SetStatusCondition(&si.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&si.ObjectMeta, instanceIsRunningAnnotation, "true")
	return UpdateResult{}, nil
}

func (r *ServiceIntegrationController) Delete(ctx context.Context, si *v1alpha1.ServiceIntegration) error {
	if si.Status.ID == "" {
		return nil
	}

	err := r.avnGen.ServiceIntegrationDelete(ctx, si.Spec.Project, si.Status.ID)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting service integration: %w", err)
	}

	return nil
}

func (r *ServiceIntegrationController) checkPreconditions(ctx context.Context, si *v1alpha1.ServiceIntegration) error {
	// todo: validate SourceEndpointID, DestinationEndpointID when ServiceIntegrationEndpoint kind released

	if si.Spec.DestinationEndpointID != "" && si.Spec.DestinationEndpointRef != nil {
		return fmt.Errorf("destinationEndpointId and destinationEndpointRef are mutually exclusive")
	}

	if si.Spec.SourceServiceName != "" {
		project := si.Spec.SourceProjectName
		if project == "" {
			project = si.Spec.Project
		}

		if err := r.checkService(ctx, project, si.Spec.SourceServiceName); err != nil {
			return err
		}
	}

	if si.Spec.DestinationServiceName != "" {
		project := si.Spec.DestinationProjectName
		if project == "" {
			project = si.Spec.Project
		}

		if err := r.checkService(ctx, project, si.Spec.DestinationServiceName); err != nil {
			return err
		}
	}

	if _, err := r.resolveDestinationEndpointID(ctx, si); err != nil {
		return err
	}

	return nil
}

func (r *ServiceIntegrationController) checkService(ctx context.Context, project, serviceName string) error {
	on, err := checkServiceIsOperational(ctx, r.avnGen, project, serviceName)
	if err != nil {
		return err
	}
	if !on {
		return fmt.Errorf("%w: service %s/%s is not yet operational", errPreconditionNotMet, project, serviceName)
	}
	return nil
}

// resolveDestinationEndpointID resolves the endpoint ID from destinationEndpointId or destinationEndpointRef.
func (r *ServiceIntegrationController) resolveDestinationEndpointID(ctx context.Context, si *v1alpha1.ServiceIntegration) (string, error) {
	if si.Spec.DestinationEndpointID != "" {
		return si.Spec.DestinationEndpointID, nil
	}

	ref := si.Spec.DestinationEndpointRef
	if ref == nil {
		return "", nil
	}
	if strings.Contains(ref.Name, "/") {
		return "", fmt.Errorf("destinationEndpointRef.name must not contain a namespace")
	}

	endpoint := &v1alpha1.ServiceIntegrationEndpoint{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: si.GetNamespace()}, endpoint); err != nil {
		if apierrors.IsNotFound(err) {
			return "", fmt.Errorf("%w: destination endpoint %s is not yet available", errPreconditionNotMet, ref.Name)
		}
		return "", fmt.Errorf("getting destination endpoint: %w", err)
	}

	if endpoint.Spec.Project != si.Spec.Project {
		return "", fmt.Errorf("destination endpoint %s has project %q, expected %q", endpoint.GetName(), endpoint.Spec.Project, si.Spec.Project)
	}

	if !IsReadyToUse(endpoint) {
		return "", fmt.Errorf("%w: destination endpoint %s is not ready", errPreconditionNotMet, endpoint.GetName())
	}

	if endpoint.Status.ID == "" {
		return "", fmt.Errorf("%w: destination endpoint %s has no status.id yet", errPreconditionNotMet, endpoint.GetName())
	}

	return endpoint.Status.ID, nil
}

// findExistingIntegration checks if an integration with matching configuration already exists on Aiven.
func (r *ServiceIntegrationController) findExistingIntegration(ctx context.Context, si *v1alpha1.ServiceIntegration, destEndpointID string) (*service.ServiceIntegrationOut, error) {
	if si.Spec.SourceServiceName == "" {
		return nil, nil // integration with only endpoints, cannot list integrations
	}

	matchCandidate := si
	if destEndpointID != "" && si.Spec.DestinationEndpointID == "" {
		matchCandidate = si.DeepCopy()
		matchCandidate.Spec.DestinationEndpointID = destEndpointID
	}

	sourceProject := si.Spec.SourceProjectName
	if sourceProject == "" {
		sourceProject = si.Spec.Project
	}

	svc, err := r.avnGen.ServiceGet(ctx, sourceProject, si.Spec.SourceServiceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s/%s: %w", sourceProject, si.Spec.SourceServiceName, err)
	}

	for _, integration := range svc.ServiceIntegrations {
		if r.integrationMatches(&integration, matchCandidate) {
			return &integration, nil
		}
	}

	return nil, nil
}

type integrationKey struct {
	IntegrationType  service.IntegrationType
	SourceService    string
	SourceProject    string
	SourceEndpointID *string
	DestService      *string
	DestProject      string
	DestEndpointID   *string
}

func (r *ServiceIntegrationController) integrationMatches(existing *service.ServiceIntegrationOut, desired *v1alpha1.ServiceIntegration) bool {
	sourceProject := desired.Spec.SourceProjectName
	if sourceProject == "" {
		sourceProject = desired.Spec.Project
	}

	destProject := desired.Spec.DestinationProjectName
	if destProject == "" {
		destProject = desired.Spec.Project
	}

	existingKey := integrationKey{
		IntegrationType:  existing.IntegrationType,
		SourceService:    existing.SourceService,
		SourceProject:    existing.SourceProject,
		SourceEndpointID: existing.SourceEndpointId,
		DestService:      existing.DestService,
		DestProject:      existing.DestProject,
		DestEndpointID:   existing.DestEndpointId,
	}

	desiredKey := integrationKey{
		IntegrationType:  desired.Spec.IntegrationType,
		SourceService:    desired.Spec.SourceServiceName,
		SourceProject:    sourceProject,
		SourceEndpointID: NilIfZero(desired.Spec.SourceEndpointID),
		DestService:      NilIfZero(desired.Spec.DestinationServiceName),
		DestProject:      destProject,
		DestEndpointID:   NilIfZero(desired.Spec.DestinationEndpointID),
	}

	return cmp.Equal(existingKey, desiredKey)
}
