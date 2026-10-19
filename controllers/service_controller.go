// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// errServiceCreatePoweredOff: the service is missing at Aiven and the spec has powered: false.
// ServiceCreateIn has no powered flag, so the resource cannot be reconciled until powered is true.
// An existing powered-off service is unaffected: it is adopted through Update.
var errServiceCreatePoweredOff = errors.New("service does not exist at Aiven and cannot be created powered off: set powered to true, then power it off once it is running")

// genericServiceController is the AivenController[T] implementation shared by all Aiven service resources.
// It embeds the legacy handler for the shared helpers until the legacy path is removed.
type genericServiceController[T v1alpha1.AivenManagedObject] struct {
	*genericServiceHandler
	avnGen avngen.Client
}

// newGenericServiceController builds a genericServiceController for a specific service type.
func newGenericServiceController[T v1alpha1.AivenManagedObject](
	c Controller,
	avnGen avngen.Client,
	fabric serviceAdapterFabric,
) *genericServiceController[T] {
	return &genericServiceController[T]{
		genericServiceHandler: &genericServiceHandler{
			fabric: fabric,
			log:    c.Log,
			k8s:    c.Client,
			rec:    c.Recorder,
		},
		avnGen: avnGen,
	}
}

func (c *genericServiceController[T]) Observe(ctx context.Context, obj T) (Observation, error) {
	o, err := c.fabric(obj)
	if err != nil {
		return Observation{}, err
	}

	spec := o.getServiceCommonSpec()

	// Powered defaults to true.
	isPowered := true
	if spec.Powered != nil {
		isPowered = *spec.Powered
	}

	// Fetch secrets only when they can be published this round.
	var query [][2]string
	if hasLatestGeneration(obj) && isPowered && !obj.NoSecret() {
		query = append(query, service.ServiceGetIncludeSecrets(true))
	}
	avnService, err := c.avnGen.ServiceGet(ctx, spec.Project, o.getObjectMeta().Name, query...)
	exists := err == nil
	if err != nil && !isNotFound(err) {
		return Observation{}, fmt.Errorf("failed to get service from Aiven: %w", err)
	}

	// An applied service is gone at Aiven: revoke readiness.
	if !exists && wasEverApplied(obj) {
		markInstanceNotReconciled(obj)
		status := o.getServiceStatus()
		status.State = ""
		status.Version = ""
		// The migration ran on the deleted service: the recreated one needs the credentials again.
		meta.RemoveStatusCondition(&status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
	}

	if !exists && !isPowered {
		return Observation{}, errServiceCreatePoweredOff
	}

	if !exists || !hasLatestGeneration(obj) {
		if ok, err := c.checkPreconditions(ctx, c.avnGen, obj); err != nil {
			return Observation{}, err
		} else if !ok {
			return Observation{}, errPreconditionNotMet
		}
	}

	if !exists {
		return Observation{ResourceExists: false}, nil
	}

	status := o.getServiceStatus()
	status.State = avnService.State
	status.Version = serviceVersion(o.getServiceType(), avnService.Metadata)

	// Unprocessed generation.
	if !hasLatestGeneration(obj) {
		return Observation{ResourceExists: true, ResourceUpToDate: false}, nil
	}

	// Not in its target state yet: keep polling without touching the running marker.
	if (!isPowered || status.State != service.ServiceStateTypeRunning) && (isPowered || status.State != service.ServiceStateTypePoweroff) {
		return Observation{ResourceExists: true, ResourceUpToDate: true}, nil
	}

	// A migration in progress is polled at requeueTimeout instead of PollInterval.
	// TODO: move this block and resolveUserConfig into a per-kind migration component
	// once kinds are defined instead of adapters.
	var requeueAfter time.Duration
	if mp, ok := o.(migrationSecretProvider); ok && mp.getMigrationSecretSource() != nil {
		// A completed migration is not polled again: ServiceGetMigrationStatus would 404.
		migrationDone := meta.IsStatusConditionTrue(status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		if !migrationDone {
			if err := c.updateMigrationStatus(ctx, c.avnGen, o, spec); err != nil {
				return Observation{}, err
			}
			if hasPendingMigration(obj) {
				requeueAfter = requeueTimeout
			}
		}
		if err := c.maybeDeleteMigrationSecret(ctx, o, mp); err != nil {
			return Observation{}, err
		}
	}

	// Powered off: no connection secret.
	if !isPowered {
		markInstancePoweredOff(obj)
		return Observation{ResourceExists: true, ResourceUpToDate: true, RequeueAfter: requeueAfter}, nil
	}

	details, err := c.buildSecretDetails(ctx, o, obj, avnService, spec.Project)
	if err != nil {
		return Observation{}, err
	}

	markInstanceRunning(obj)
	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: true,
		SecretDetails:    details,
		PruneSecretKeys:  true, // The details are the whole secret: drop keys no longer produced.
		RequeueAfter:     requeueAfter,
	}, nil
}

// buildSecretDetails returns the adapter's connection secret as publishable details, with the
// project CA certificate added for the kinds that expose one. Nil when the secret is disabled or
// the adapter produces none. A failed CA fetch is wrapped in errConnInfoSecret.
func (c *genericServiceController[T]) buildSecretDetails(
	ctx context.Context,
	o serviceAdapter,
	obj T,
	avnService *service.ServiceGetOut,
	project string,
) (map[string]string, error) {
	// The service was fetched without secrets, which the adapters dereference.
	if obj.NoSecret() {
		if GetIsRunningAnnotation(obj) != "true" {
			c.rec.Event(
				obj,
				corev1.EventTypeNormal,
				eventConnInfoSecretCreationDisabled,
				"connInfoSecretTargetDisabled is true, secret will not be created",
			)
		}
		return nil, nil
	}

	secret := o.newSecret(avnService)
	if secret == nil {
		return nil, nil
	}

	// Keys are already prefixed. The CA-cert kinds always populate StringData.
	details := maps.Clone(secret.StringData)

	switch o.getServiceType() {
	case serviceTypeKafka, serviceTypePostgreSQL, serviceTypeMySQL:
		// CA_CERT is only relevant for these service types.
		cert, err := c.avnGen.ProjectKmsGetCA(ctx, project)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot retrieve project CA certificate: %w", errConnInfoSecret, err)
		}

		prefix := getSecretPrefix(o)
		details[prefix+"CA_CERT"] = cert
		if o.getServiceType() == serviceTypeKafka {
			// todo: backward compatibility, remove in future releases
			details["CA_CERT"] = cert
		}
	}

	return details, nil
}

// Create implements AivenController[T]: ServiceCreate from the spec.
func (c *genericServiceController[T]) Create(ctx context.Context, obj T) (CreateResult, error) {
	o, err := c.fabric(obj)
	if err != nil {
		return CreateResult{}, err
	}

	spec := o.getServiceCommonSpec()
	ometa := o.getObjectMeta()

	userCfg, err := c.resolveUserConfig(ctx, o)
	if err != nil {
		return CreateResult{}, err
	}

	projectVPCID, err := c.resolveProjectVPCID(ctx, o, spec)
	if err != nil {
		return CreateResult{}, err
	}

	userConfig, err := CreateUserConfiguration(userCfg)
	if err != nil {
		return CreateResult{}, err
	}

	req := service.ServiceCreateIn{
		Cloud:                 NilIfZero(spec.CloudName),
		DiskSpaceMb:           NilIfZero(v1alpha1.ConvertDiskSpace(o.getDiskSpace())),
		Maintenance:           getMaintenanceWindow(spec.MaintenanceWindowDow, spec.MaintenanceWindowTime),
		Plan:                  spec.Plan,
		ProjectVpcId:          toOptionalStringPointer(projectVPCID),
		ServiceName:           ometa.Name,
		ServiceType:           string(o.getServiceType()),
		TerminationProtection: spec.TerminationProtection,
		UserConfig:            &userConfig,
		TechEmails:            buildTechEmails(spec),
	}

	// ServiceIntegrations are only applied at creation time (see ServiceCommonSpec docs).
	integrations := make([]service.ServiceIntegrationIn, 0, len(spec.ServiceIntegrations))
	for _, s := range spec.ServiceIntegrations {
		integrations = append(integrations, service.ServiceIntegrationIn{
			IntegrationType: s.IntegrationType,
			SourceService:   &s.SourceServiceName,
		})
	}
	if len(integrations) > 0 {
		req.ServiceIntegrations = &integrations
	}

	// 409: Observe saw a transient 404. Nothing was applied, so soft-requeue without claiming the generation.
	switch _, err := c.avnGen.ServiceCreate(ctx, spec.Project, &req); {
	case isAlreadyExists(err):
		return CreateResult{}, fmt.Errorf("%w: service already exists at Aiven, re-observing", errPreconditionNotMet)
	case err != nil:
		return CreateResult{}, fmt.Errorf("failed to create service: %w", err)
	case wasEverApplied(obj):
		c.rec.Event(
			obj,
			corev1.EventTypeWarning,
			eventInstanceMissingAtAiven,
			"service was not found at Aiven and has been recreated from the spec",
		)
		// Un-stamp the generation so a failure below is retried through Update.
		delete(obj.GetAnnotations(), processedGenerationAnnotation)
	}

	if err := c.replaceTags(ctx, spec.Project, ometa.Name, spec.Tags); err != nil {
		return CreateResult{}, err
	}

	// No previous service on the creation path.
	if err := o.createOrUpdateServiceSpecific(ctx, c.avnGen, nil); err != nil {
		return CreateResult{}, fmt.Errorf("failed to create or update service-specific: %w", err)
	}

	markServiceCreatedOrUpdated(obj)
	return CreateResult{}, nil
}

// Update implements AivenController[T]: the upgrade task, ServiceUpdate from the spec, then tags
// and the service-specific hook.
func (c *genericServiceController[T]) Update(ctx context.Context, obj T) (UpdateResult, error) {
	o, err := c.fabric(obj)
	if err != nil {
		return UpdateResult{}, err
	}

	spec := o.getServiceCommonSpec()
	ometa := o.getObjectMeta()

	userCfg, err := c.resolveUserConfig(ctx, o)
	if err != nil {
		return UpdateResult{}, err
	}

	projectVPCID, err := c.resolveProjectVPCID(ctx, o, spec)
	if err != nil {
		return UpdateResult{}, err
	}

	oldService, err := c.avnGen.ServiceGet(ctx, spec.Project, ometa.Name)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("failed to fetch service: %w", err)
	}

	// Disk space cannot be managed manually when the autoscaler integration owns it.
	diskSpace := v1alpha1.ConvertDiskSpace(o.getDiskSpace())
	if diskSpace > 0 {
		for _, v := range oldService.ServiceIntegrations {
			if v.IntegrationType == service.IntegrationTypeAutoscaler {
				return UpdateResult{}, fmt.Errorf("cannot set disk space for service with autoscaler integration enabled")
			}
		}
	}

	userConfig, err := UpdateUserConfiguration(userCfg)
	if err != nil {
		return UpdateResult{}, err
	}

	// The upgrade task (PostgreSQL only) must run before ServiceUpdate.
	if err := o.performUpgradeTaskIfNeeded(ctx, c.avnGen, oldService); err != nil {
		return UpdateResult{}, err
	}

	req := service.ServiceUpdateIn{
		Cloud:                 NilIfZero(spec.CloudName),
		DiskSpaceMb:           NilIfZero(diskSpace),
		Maintenance:           getMaintenanceWindow(spec.MaintenanceWindowDow, spec.MaintenanceWindowTime),
		Plan:                  NilIfZero(spec.Plan),
		Powered:               spec.Powered,
		ProjectVpcId:          NilIfZero(projectVPCID),
		TerminationProtection: spec.TerminationProtection,
		UserConfig:            &userConfig,
		TechEmails:            buildTechEmails(spec),
	}
	if _, err := c.avnGen.ServiceUpdate(ctx, spec.Project, ometa.Name, &req); err != nil {
		return UpdateResult{}, fmt.Errorf("failed to update service: %w", err)
	}

	if err := c.replaceTags(ctx, spec.Project, ometa.Name, spec.Tags); err != nil {
		return UpdateResult{}, err
	}

	if err := o.createOrUpdateServiceSpecific(ctx, c.avnGen, oldService); err != nil {
		return UpdateResult{}, fmt.Errorf("failed to create or update service-specific: %w", err)
	}

	markServiceCreatedOrUpdated(obj)
	return UpdateResult{}, nil
}

// resolveUserConfig returns the user config with the migration credentials from the referenced
// Secret merged in until the migration completes; Aiven persists the block after that.
func (c *genericServiceController[T]) resolveUserConfig(ctx context.Context, o serviceAdapter) (any, error) {
	userCfg := o.getUserConfig()
	mp, ok := o.(migrationSecretProvider)
	if !ok || mp.getMigrationSecretSource() == nil {
		return userCfg, nil
	}

	if meta.IsStatusConditionTrue(o.getServiceStatus().Conditions, v1alpha1.ConditionTypeMigrationComplete) {
		return userCfg, nil
	}

	data, err := readMigrationSecret(ctx, c.k8s, o.getObjectMeta().Namespace, mp.getMigrationSecretSource())
	if err != nil {
		return nil, fmt.Errorf("failed to read migration secret: %w", err)
	}
	resolved, err := mp.buildUserConfigWithMigration(data)
	if err != nil {
		return nil, fmt.Errorf("failed to build migration user config: %w", err)
	}
	return resolved, nil
}

// resolveProjectVPCID returns spec.projectVPCID or the status ID of the referenced ProjectVPC.
// The reconciler resolves refs only to gate readiness, so the lookup happens here.
func (c *genericServiceController[T]) resolveProjectVPCID(ctx context.Context, o serviceAdapter, spec *v1alpha1.ServiceCommonSpec) (string, error) {
	if spec.ProjectVPCID != "" {
		return spec.ProjectVPCID, nil
	}
	if spec.ProjectVPCRef == nil {
		return "", nil
	}

	ref := spec.ProjectVPCRef.ProjectVPC(o.getObjectMeta().Namespace)
	projectVPC := &v1alpha1.ProjectVPC{}
	if err := c.k8s.Get(ctx, ref.NamespacedName, projectVPC); err != nil {
		return "", fmt.Errorf("failed to get referenced ProjectVPC %q: %w", ref.NamespacedName, err)
	}
	return projectVPC.Status.ID, nil
}

// replaceTags replaces the service tags at Aiven; nil clears them.
func (c *genericServiceController[T]) replaceTags(ctx context.Context, project, serviceName string, tags map[string]string) error {
	req := service.ProjectServiceTagsReplaceIn{Tags: make(map[string]string)}
	if tags != nil {
		req.Tags = tags
	}
	if err := c.avnGen.ProjectServiceTagsReplace(ctx, project, serviceName, &req); err != nil {
		return fmt.Errorf("failed to update tags: %w", err)
	}
	return nil
}

// buildTechEmails converts the spec's technical emails; an empty spec is sent as an empty list.
func buildTechEmails(spec *v1alpha1.ServiceCommonSpec) *[]service.TechEmailIn {
	technicalEmails := make([]service.TechEmailIn, 0, len(spec.TechnicalEmails))
	for _, email := range spec.TechnicalEmails {
		technicalEmails = append(technicalEmails, service.TechEmailIn(email))
	}
	return &technicalEmails
}

// markServiceCreatedOrUpdated records the CreatedOrUpdated conditions and clears the running marker.
func markServiceCreatedOrUpdated(obj v1alpha1.AivenManagedObject) {
	delete(obj.GetAnnotations(), instanceIsRunningAnnotation)

	// We don't really know if an instance is created or updated.
	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(obj.Conditions(), getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(obj.Conditions(), getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))
}

// Delete implements AivenController[T].
func (c *genericServiceController[T]) Delete(ctx context.Context, obj T) error {
	o, err := c.fabric(obj)
	if err != nil {
		return err
	}

	spec := o.getServiceCommonSpec()
	if fromAnyPointer(spec.TerminationProtection) {
		// A hard error keeps the finalizer.
		return errTerminationProtectionOn
	}

	// Sync termination protection to Aiven before deletion.
	// The Kubernetes workqueue deduplicates events, when a user sets
	// spec.terminationProtection=false and deletes the resource immediately after,
	// the workqueue may collapse both events into a single reconciliation.
	// Without this call, Aiven may still have termination protection enabled.
	terminationProtection := false
	if _, err := c.avnGen.ServiceUpdate(ctx, spec.Project, o.getObjectMeta().Name, &service.ServiceUpdateIn{
		TerminationProtection: &terminationProtection,
	}); err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to disable termination protection before deletion: %w", err)
	}

	if err := c.avnGen.ServiceDelete(ctx, spec.Project, o.getObjectMeta().Name); err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete service in Aiven: %w", err)
	}
	return nil
}
