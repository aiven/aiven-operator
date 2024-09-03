package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
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

func (h *genericServiceHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	o, err := h.fabric(avn, obj)
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

	oldService, err := avnGen.ServiceGet(ctx, spec.Project, ometa.Name)
	exists := err == nil
	if !exists && !isNotFound(err) {
		return fmt.Errorf("failed to fetch service: %w", err)
	}

	technicalEmails := make([]aiven.ContactEmail, 0)
	for _, email := range spec.TechnicalEmails {
		technicalEmails = append(technicalEmails, aiven.ContactEmail(email))
	}

	// Creates if not exists or updates existing service
	var reason string
	if !exists {
		reason = "Created"
		userConfig, err := CreateUserConfiguration(o.getUserConfig())
		if err != nil {
			return err
		}

		req := aiven.CreateServiceRequest{
			Cloud:                 spec.CloudName,
			DiskSpaceMB:           v1alpha1.ConvertDiskSpace(o.getDiskSpace()),
			MaintenanceWindow:     getMaintenanceWindow(spec.MaintenanceWindowDow, spec.MaintenanceWindowTime),
			Plan:                  spec.Plan,
			ProjectVPCID:          toOptionalStringPointer(projectVPCID),
			ServiceIntegrations:   nil,
			ServiceName:           ometa.Name,
			ServiceType:           o.getServiceType(),
			TerminationProtection: fromAnyPointer(spec.TerminationProtection),
			UserConfig:            userConfig,
			TechnicalEmails:       &technicalEmails,
		}

		for _, s := range spec.ServiceIntegrations {
			i := aiven.NewServiceIntegration{
				IntegrationType: s.IntegrationType,
				SourceService:   &s.SourceServiceName,
			}
			req.ServiceIntegrations = append(req.ServiceIntegrations, i)
		}

		_, err = avn.Services.Create(ctx, spec.Project, req)
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	} else {
		reason = "Updated"
		userConfig, err := UpdateUserConfiguration(o.getUserConfig())
		if err != nil {
			return err
		}

		// Perform upgrade task if necessary (at the moment, this is relevant only for PostgreSQL)
		err = o.performUpgradeTaskIfNeeded(ctx, avnGen, oldService)
		if err != nil {
			return err
		}

		req := aiven.UpdateServiceRequest{
			Cloud:                 spec.CloudName,
			DiskSpaceMB:           v1alpha1.ConvertDiskSpace(o.getDiskSpace()),
			MaintenanceWindow:     getMaintenanceWindow(spec.MaintenanceWindowDow, spec.MaintenanceWindowTime),
			Plan:                  spec.Plan,
			Powered:               true,
			ProjectVPCID:          toOptionalStringPointer(projectVPCID),
			TerminationProtection: fromAnyPointer(spec.TerminationProtection),
			UserConfig:            userConfig,
			TechnicalEmails:       &technicalEmails,
		}
		_, err = avn.Services.Update(ctx, spec.Project, ometa.Name, req)
		if err != nil {
			return fmt.Errorf("failed to update service: %w", err)
		}
	}

	// Updates tags.
	// Four scenarios: service created/updated * with/without tags
	// By sending empty tags it clears existing list
	req := aiven.ServiceTagsRequest{
		Tags: make(map[string]string),
	}
	if spec.Tags != nil {
		req.Tags = spec.Tags
	}
	_, err = avn.ServiceTags.Set(ctx, spec.Project, ometa.Name, req)
	if err != nil {
		return fmt.Errorf("failed to update tags: %w", err)
	}

	status := o.getServiceStatus()
	meta.SetStatusCondition(&status.Conditions,
		getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Successfully created or updated the instance in Aiven, status remains unknown"))

	metav1.SetMetaDataAnnotation(
		o.getObjectMeta(),
		processedGenerationAnnotation,
		strconv.FormatInt(obj.GetGeneration(), formatIntBaseDecimal),
	)
	return nil
}

func (h *genericServiceHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	o, err := h.fabric(avn, obj)
	if err != nil {
		return false, err
	}

	spec := o.getServiceCommonSpec()
	if fromAnyPointer(spec.TerminationProtection) {
		return false, errTerminationProtectionOn
	}

	err = avn.Services.Delete(ctx, spec.Project, o.getObjectMeta().Name)
	if err == nil || isNotFound(err) {
		return true, nil
	}

	return false, fmt.Errorf("failed to delete service in Aiven: %w", err)
}

func (h *genericServiceHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	o, err := h.fabric(avn, obj)
	if err != nil {
		return nil, err
	}

	spec := o.getServiceCommonSpec()
	s, err := avnGen.ServiceGet(ctx, spec.Project, o.getObjectMeta().Name, service.ServiceGetIncludeSecrets(true))
	if err != nil {
		return nil, fmt.Errorf("failed to get service from Aiven: %w", err)
	}

	status := o.getServiceStatus()
	if !serviceIsRunning(s.State) {
		status.State = s.State
		return nil, nil
	}

	status.State = "RUNNING" // overrides REBALANCING
	meta.SetStatusCondition(&status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(o.getObjectMeta(), instanceIsRunningAnnotation, "true")

	// Some services get secrets after they are running only,
	// like ip addresses (hosts)
	secret, err := o.newSecret(ctx, s)
	if err != nil || secret == nil {
		return secret, err
	}

	switch o.getServiceType() {
	case "kafka", "pg", "mysql", "cassandra":
		// CA_CERT can be used with these service types only
	default:
		return secret, nil
	}

	cert, err := avnGen.ProjectKmsGetCA(ctx, spec.Project)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve project CA certificate: %w", err)
	}

	// We don't expect the StringData map to be empty, it must panic.
	prefix := getSecretPrefix(o)
	secret.StringData[prefix+"CA_CERT"] = cert
	if o.getServiceType() == "kafka" {
		// todo: backward compatibility, remove in future releases
		secret.StringData["CA_CERT"] = cert
	}
	return secret, nil
}

// checkPreconditions not required for now by services to be implemented
func (h *genericServiceHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	o, err := h.fabric(avn, obj)
	if err != nil {
		return false, err
	}

	spec := o.getServiceCommonSpec()
	for _, s := range spec.ServiceIntegrations {
		// Validates that read_replica is running
		// If not, the wrapper controller will try later
		if s.IntegrationType == "read_replica" {
			r, err := checkServiceIsOperational(ctx, avnGen, spec.Project, s.SourceServiceName)
			if !r || err != nil {
				return false, err
			}

			// Covers error "No valid backups for service"
			list, err := avnGen.ServiceBackupsGet(ctx, spec.Project, s.SourceServiceName)
			if len(list) == 0 || err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// serviceAdapterFabric returns serviceAdapter for specific service, like MySQL
type serviceAdapterFabric func(*aiven.Client, client.Object) (serviceAdapter, error)

// serviceAdapter turns client.Object into a generic thing
type serviceAdapter interface {
	objWithSecret
	getObjectMeta() *metav1.ObjectMeta
	getServiceStatus() *v1alpha1.ServiceStatus
	getServiceCommonSpec() *v1alpha1.ServiceCommonSpec
	getServiceType() string
	getDiskSpace() string
	getUserConfig() any
	newSecret(ctx context.Context, s *service.ServiceGetOut) (*corev1.Secret, error)
	performUpgradeTaskIfNeeded(ctx context.Context, avn avngen.Client, old *service.ServiceGetOut) error
}
