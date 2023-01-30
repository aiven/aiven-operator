package controllers

import (
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func newGenericServiceHandler(fabric serviceAdapterFabric) Handlers {
	return &genericServiceHandler{fabric: fabric}
}

// genericServiceHandler provides common CRUD management for all service types using serviceAdapter,
// which turns specific service (mysql, redis) into a generic.
type genericServiceHandler struct {
	fabric serviceAdapterFabric
}

func (h *genericServiceHandler) createOrUpdate(a *aiven.Client, object client.Object, refs []client.Object) error {
	o, err := h.fabric(object)
	if err != nil {
		return err
	}

	spec := o.getServiceCommonSpec()
	ometa := o.getObjectMeta()

	// Project id reference
	// Could be right in spec or referenced (has ref)
	projectVPCID := spec.ProjectVPCID
	if projectVPCID == "" {
		if p := v1alpha1.FindProjectVPC(refs); p != nil {
			projectVPCID = p.Status.ID
		}
	}

	_, err = a.Services.Get(spec.Project, ometa.Name)
	exists := err == nil
	if !exists && !aiven.IsNotFound(err) {
		return fmt.Errorf("failed to fetch service: %w", err)
	}

	// Creates if not exists or updates existing service
	var reason string
	if !exists {
		reason = "Created"
		userConfig, err := UserConfigurationToAPIV2(o.getUserConfig(), []string{"create", "update"})
		if err != nil {
			return err
		}

		req := aiven.CreateServiceRequest{
			Cloud:                 spec.CloudName,
			DiskSpaceMB:           v1alpha1.ConvertDiscSpace(o.getDiskSpace()),
			MaintenanceWindow:     getMaintenanceWindow(spec.MaintenanceWindowDow, spec.MaintenanceWindowTime),
			Plan:                  spec.Plan,
			ProjectVPCID:          toOptionalStringPointer(projectVPCID),
			ServiceIntegrations:   nil,
			ServiceName:           ometa.Name,
			ServiceType:           o.getServiceType(),
			TerminationProtection: spec.TerminationProtection,
			UserConfig:            userConfig,
		}

		_, err = a.Services.Create(spec.Project, req)
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	} else {
		reason = "Updated"
		userConfig, err := UserConfigurationToAPIV2(o.getUserConfig(), []string{"update"})
		if err != nil {
			return err
		}

		req := aiven.UpdateServiceRequest{
			Cloud:                 spec.CloudName,
			DiskSpaceMB:           v1alpha1.ConvertDiscSpace(o.getDiskSpace()),
			MaintenanceWindow:     getMaintenanceWindow(spec.MaintenanceWindowDow, spec.MaintenanceWindowTime),
			Plan:                  spec.Plan,
			Powered:               true,
			ProjectVPCID:          toOptionalStringPointer(projectVPCID),
			TerminationProtection: spec.TerminationProtection,
			UserConfig:            userConfig,
		}
		_, err = a.Services.Update(spec.Project, ometa.Name, req)
		if err != nil {
			return fmt.Errorf("failed to update service: %w", err)
		}
	}

	status := o.getServiceStatus()
	meta.SetStatusCondition(&status.Conditions,
		getInitializedCondition(reason, "Instance was created or update on Aiven side"))
	meta.SetStatusCondition(&status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason, "Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(
		o.getObjectMeta(),
		processedGenerationAnnotation,
		strconv.FormatInt(object.GetGeneration(), formatIntBaseDecimal),
	)
	return nil
}

func (h *genericServiceHandler) delete(a *aiven.Client, object client.Object) (bool, error) {
	o, err := h.fabric(object)
	if err != nil {
		return false, err
	}

	err = a.Services.Delete(o.getServiceCommonSpec().Project, o.getObjectMeta().Name)
	if err == nil || aiven.IsNotFound(err) {
		return true, nil
	}

	return false, fmt.Errorf("failed to delete service in Aiven: %w", err)
}

func (h *genericServiceHandler) get(a *aiven.Client, object client.Object) (*corev1.Secret, error) {
	o, err := h.fabric(object)
	if err != nil {
		return nil, err
	}

	s, err := a.Services.Get(o.getServiceCommonSpec().Project, o.getObjectMeta().Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get service from Aiven: %w", err)
	}

	status := o.getServiceStatus()
	status.State = s.State
	if s.State == "RUNNING" {
		meta.SetStatusCondition(&status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(o.getObjectMeta(), instanceIsRunningAnnotation, "true")

		// Some services get secrets after they are running only,
		// like ip addresses (hosts)
		return o.newSecret(s), nil
	}
	return nil, nil
}

// checkPreconditions not required for now by services to be implemented
func (h *genericServiceHandler) checkPreconditions(a *aiven.Client, object client.Object) (bool, error) {
	return true, nil
}

// serviceAdapterFabric returns serviceAdapter for specific service, like MySQL
type serviceAdapterFabric func(client.Object) (serviceAdapter, error)

// serviceAdapter turns client.Object into a generic thing
type serviceAdapter interface {
	getObjectMeta() *metav1.ObjectMeta
	getServiceStatus() *v1alpha1.ServiceStatus
	getServiceCommonSpec() *v1alpha1.ServiceCommonSpec
	getServiceType() string
	getDiskSpace() string
	getUserConfig() any
	newSecret(*aiven.Service) *corev1.Secret
}
