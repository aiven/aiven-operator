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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// errServiceCreatePoweredOff: the service is missing at Aiven and the spec has powered: false.
// ServiceCreateIn has no powered flag, so the resource cannot be reconciled until powered is true.
// An existing powered-off service is unaffected: it is adopted through Update.
var errServiceCreatePoweredOff = errors.New("service does not exist at Aiven and cannot be created powered off: set powered to true, then power it off once it is running")

// serviceController is the AivenController[T] implementation shared by all Aiven service resources.
// Per-kind behavior comes from kind.
type serviceController[T v1alpha1.ServiceObject] struct {
	kind   serviceKind[T]
	avnGen avngen.Client
	k8s    client.Client
	rec    record.EventRecorder
	log    logr.Logger
}

// newServiceController builds a serviceController for a specific service kind.
func newServiceController[T v1alpha1.ServiceObject](
	c Controller,
	avnGen avngen.Client,
	kind serviceKind[T],
) *serviceController[T] {
	return &serviceController[T]{
		kind:   kind,
		avnGen: avnGen,
		k8s:    c.Client,
		rec:    c.Recorder,
		log:    c.Log,
	}
}

func (c *serviceController[T]) Observe(ctx context.Context, obj T) (Observation, error) {
	spec := obj.ServiceCommonSpec()
	powered := isPowered(spec)

	// Fetch secrets only when they can be published this round.
	var query [][2]string
	if hasLatestGeneration(obj) && powered && !obj.NoSecret() {
		query = append(query, service.ServiceGetIncludeSecrets(true))
	}
	avnService, err := c.avnGen.ServiceGet(ctx, spec.Project, obj.GetName(), query...)
	exists := err == nil
	if err != nil && !isNotFound(err) {
		return Observation{}, fmt.Errorf("failed to get service from Aiven: %w", err)
	}

	// An applied service is gone at Aiven: revoke readiness.
	if !exists && wasEverApplied(obj) {
		markInstanceNotReconciled(obj)
		status := obj.ServiceStatus()
		status.State = ""
		status.Version = ""
		// The migration ran on the deleted service: the recreated one needs the credentials again.
		meta.RemoveStatusCondition(&status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
	}

	// Refuse to create a powered-off service.
	if !exists && !powered {
		return Observation{}, errServiceCreatePoweredOff
	}

	if !exists || !hasLatestGeneration(obj) {
		if err := c.kind.checkPreconditions(ctx, c.avnGen, obj); err != nil {
			return Observation{}, err
		}
	}

	if !exists {
		return Observation{ResourceExists: false}, nil
	}

	status := obj.ServiceStatus()
	status.State = avnService.State
	status.Version = serviceVersion(c.kind.serviceType, avnService.Metadata)

	// Unprocessed generation.
	if !hasLatestGeneration(obj) {
		return Observation{ResourceExists: true, ResourceUpToDate: false}, nil
	}

	// Not in its target state yet: keep polling without touching the running marker.
	if (!powered || status.State != service.ServiceStateTypeRunning) && (powered || status.State != service.ServiceStateTypePoweroff) {
		return Observation{ResourceExists: true, ResourceUpToDate: true}, nil
	}

	// A migration in progress is polled at requeueTimeout instead of PollInterval.
	var requeueAfter time.Duration
	if src := c.kind.migrationSource(obj); src != nil {
		// A completed migration is not polled again: ServiceGetMigrationStatus would 404.
		migrationDone := meta.IsStatusConditionTrue(status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		if !migrationDone {
			if err := updateMigrationCondition(ctx, c.avnGen, spec.Project, obj.GetName(), obj.GetGeneration(), &status.Conditions); err != nil {
				return Observation{}, err
			}
			if hasPendingMigration(obj) {
				requeueAfter = requeueTimeout
			}
		}
		if err := c.maybeDeleteMigrationSecret(ctx, obj, src); err != nil {
			return Observation{}, err
		}
	}

	// Powered off: no connection secret.
	if !powered {
		markInstancePoweredOff(obj)
		return Observation{ResourceExists: true, ResourceUpToDate: true, RequeueAfter: requeueAfter}, nil
	}

	details, err := c.buildSecretDetails(ctx, obj, avnService)
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

// Create implements AivenController[T]: ServiceCreate from the spec.
func (c *serviceController[T]) Create(ctx context.Context, obj T) (CreateResult, error) {
	spec := obj.ServiceCommonSpec()
	ometa := obj.GetObjectMeta()

	userCfg, err := c.resolveUserConfig(ctx, obj)
	if err != nil {
		return CreateResult{}, err
	}

	projectVPCID, err := c.resolveProjectVPCID(ctx, obj, spec)
	if err != nil {
		return CreateResult{}, err
	}

	userConfig, err := CreateUserConfiguration(userCfg)
	if err != nil {
		return CreateResult{}, err
	}

	req := service.ServiceCreateIn{
		ServiceName:           ometa.Name,
		ServiceType:           string(c.kind.serviceType),
		Plan:                  spec.Plan,
		Cloud:                 NilIfZero(spec.CloudName),
		ProjectVpcId:          NilIfZero(projectVPCID),
		Maintenance:           maintenanceIn(spec),
		TerminationProtection: spec.TerminationProtection,
		DiskSpaceMb:           NilIfZero(v1alpha1.ConvertDiskSpace(spec.DiskSpace)),
		Tags:                  tagsIn(spec),
		TechEmails:            techEmailsIn(spec),
		UserConfig:            &userConfig,
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
		// Un-stamp the generation so a failure before the reconciler stamps it is retried through Update.
		delete(obj.GetAnnotations(), processedGenerationAnnotation)
	}

	markServiceCreatedOrUpdated(obj)
	return CreateResult{}, nil
}

// Update implements AivenController[T]: the kind's beforeUpdate hook, ServiceUpdate from the spec,
// then tags when the spec sets them.
func (c *serviceController[T]) Update(ctx context.Context, obj T) (UpdateResult, error) {
	spec := obj.ServiceCommonSpec()
	objMeta := obj.GetObjectMeta()

	userCfg, err := c.resolveUserConfig(ctx, obj)
	if err != nil {
		return UpdateResult{}, err
	}

	projectVPCID, err := c.resolveProjectVPCID(ctx, obj, spec)
	if err != nil {
		return UpdateResult{}, err
	}

	oldService, err := c.avnGen.ServiceGet(ctx, spec.Project, objMeta.Name)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("failed to fetch service: %w", err)
	}

	// Disk space cannot be managed manually when the autoscaler integration owns it.
	diskSpace := v1alpha1.ConvertDiskSpace(spec.DiskSpace)
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

	// E.g. the PostgreSQL upgrade check, which must pass before ServiceUpdate.
	if c.kind.beforeUpdate != nil {
		if err := c.kind.beforeUpdate(ctx, c.avnGen, obj, oldService); err != nil {
			return UpdateResult{}, err
		}
	}

	// A null project_vpc_id moves the service out of its VPC: keep the live one when the spec has none.
	if projectVPCID == "" {
		projectVPCID = oldService.ProjectVpcId
	}
	req := service.ServiceUpdateIn{
		Plan:                  NilIfZero(spec.Plan),
		Cloud:                 NilIfZero(spec.CloudName),
		ProjectVpcId:          NilIfZero(projectVPCID),
		Maintenance:           maintenanceIn(spec),
		TerminationProtection: spec.TerminationProtection,
		Powered:               new(isPowered(spec)),
		DiskSpaceMb:           NilIfZero(diskSpace),
		TechEmails:            techEmailsIn(spec),
		UserConfig:            &userConfig,
	}
	if _, err := c.avnGen.ServiceUpdate(ctx, spec.Project, objMeta.Name, &req); err != nil {
		return UpdateResult{}, fmt.Errorf("failed to update service: %w", err)
	}

	// Tags have their own endpoint; {} clears them.
	if spec.Tags != nil {
		if err := c.replaceTags(ctx, spec.Project, objMeta.Name, spec.Tags); err != nil {
			return UpdateResult{}, err
		}
	}

	markServiceCreatedOrUpdated(obj)
	return UpdateResult{}, nil
}

// Delete implements AivenController[T].
func (c *serviceController[T]) Delete(ctx context.Context, obj T) error {
	spec := obj.ServiceCommonSpec()
	if fromAnyPointer(spec.TerminationProtection) {
		// A hard error keeps the finalizer.
		return errTerminationProtectionOn
	}

	// The workqueue may collapse "terminationProtection: false" and the delete into one reconcile,
	// so protection still on at Aiven is switched off first. The VPC is echoed: null would move the service out.
	live, err := c.avnGen.ServiceGet(ctx, spec.Project, obj.GetName())
	switch {
	case isNotFound(err):
		return nil
	case err != nil:
		return fmt.Errorf("failed to fetch service: %w", err)
	case live.TerminationProtection:
		req := service.ServiceUpdateIn{TerminationProtection: new(false), ProjectVpcId: NilIfZero(live.ProjectVpcId)}
		if _, err := c.avnGen.ServiceUpdate(ctx, spec.Project, obj.GetName(), &req); err != nil && !isNotFound(err) {
			return fmt.Errorf("failed to disable termination protection before deletion: %w", err)
		}
	}

	if err := c.avnGen.ServiceDelete(ctx, spec.Project, obj.GetName()); err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete service in Aiven: %w", err)
	}
	return nil
}

// buildSecretDetails returns the kind's connection secret as publishable details, with the
// project CA certificate added for the kinds that expose one. Nil when the secret is disabled or
// the kind produces none. A failed CA fetch is wrapped in errConnInfoSecret.
func (c *serviceController[T]) buildSecretDetails(
	ctx context.Context,
	obj T,
	avnService *service.ServiceGetOut,
) (map[string]string, error) {
	// The service was fetched without secrets, which secretData dereferences.
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

	if c.kind.secretData == nil {
		return nil, nil
	}

	// Keys are already prefixed.
	details := c.kind.secretData(obj, avnService)
	if !c.kind.projectCACert {
		return details, nil
	}

	cert, err := c.avnGen.ProjectKmsGetCA(ctx, obj.ServiceCommonSpec().Project)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot retrieve project CA certificate: %w", errConnInfoSecret, err)
	}
	details[getSecretPrefix(obj)+"CA_CERT"] = cert
	for _, k := range c.kind.caCertExtraKeys {
		details[k] = cert
	}
	return details, nil
}

// resolveUserConfig returns the user config with the migration credentials from the referenced
// Secret merged in until the migration completes; Aiven persists the block after that.
func (c *serviceController[T]) resolveUserConfig(ctx context.Context, obj T) (any, error) {
	userCfg := obj.ServiceUserConfig()
	src := c.kind.migrationSource(obj)
	if src == nil {
		return userCfg, nil
	}

	if meta.IsStatusConditionTrue(obj.ServiceStatus().Conditions, v1alpha1.ConditionTypeMigrationComplete) {
		return userCfg, nil
	}

	data, err := readMigrationSecret(ctx, c.k8s, obj.GetNamespace(), src)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration secret: %w", err)
	}
	resolved, err := c.kind.migration.merge(obj, data)
	if err != nil {
		return nil, fmt.Errorf("failed to build migration user config: %w", err)
	}
	return resolved, nil
}

// maybeDeleteMigrationSecret deletes the migration Secret once the migration is complete, when
// DeleteAfterMigration is opted in.
func (c *serviceController[T]) maybeDeleteMigrationSecret(ctx context.Context, obj T, src *v1alpha1.MigrationSecretSource) error {
	if !src.DeleteAfterMigration {
		return nil
	}
	if !meta.IsStatusConditionTrue(obj.ServiceStatus().Conditions, v1alpha1.ConditionTypeMigrationComplete) {
		return nil
	}
	if err := deleteMigrationSecret(ctx, c.k8s, obj.GetNamespace(), src); err != nil {
		return err
	}
	c.log.Info("migration secret deleted",
		"name", src.Name,
		"namespace", obj.GetNamespace(),
		"resource", obj.GetName(),
	)
	return nil
}

// resolveProjectVPCID returns spec.projectVPCID or the status ID of the referenced ProjectVPC.
// The reconciler resolves refs only to gate readiness, so the lookup happens here. A ProjectVPC
// without an ID yet is a precondition, not an unmanaged VPC: Update would otherwise echo the live one.
func (c *serviceController[T]) resolveProjectVPCID(ctx context.Context, obj T, spec *v1alpha1.ServiceCommonSpec) (string, error) {
	if spec.ProjectVPCID != "" {
		return spec.ProjectVPCID, nil
	}
	if spec.ProjectVPCRef == nil {
		return "", nil
	}

	ref := spec.ProjectVPCRef.ProjectVPC(obj.GetNamespace())
	projectVPC := &v1alpha1.ProjectVPC{}
	if err := c.k8s.Get(ctx, ref.NamespacedName, projectVPC); err != nil {
		return "", fmt.Errorf("failed to get referenced ProjectVPC %q: %w", ref.NamespacedName, err)
	}
	if projectVPC.Status.ID == "" {
		return "", fmt.Errorf("%w: referenced ProjectVPC %q has no ID yet", errPreconditionNotMet, ref.NamespacedName)
	}
	return projectVPC.Status.ID, nil
}

// tagsIn is nil when the spec omits tags; {} is kept to clear them.
func tagsIn(spec *v1alpha1.ServiceCommonSpec) *map[string]string {
	if spec.Tags == nil {
		return nil
	}
	tags := maps.Clone(spec.Tags)
	return &tags
}

// techEmailsIn is nil when the spec omits technicalEmails; [] is kept to clear them.
func techEmailsIn(spec *v1alpha1.ServiceCommonSpec) *[]service.TechEmailIn {
	if spec.TechnicalEmails == nil {
		return nil
	}
	emails := make([]service.TechEmailIn, 0, len(spec.TechnicalEmails))
	for _, e := range spec.TechnicalEmails {
		emails = append(emails, service.TechEmailIn(e))
	}
	return &emails
}

// replaceTags replaces the service tags at Aiven.
func (c *serviceController[T]) replaceTags(ctx context.Context, project, serviceName string, tags map[string]string) error {
	if err := c.avnGen.ProjectServiceTagsReplace(ctx, project, serviceName, &service.ProjectServiceTagsReplaceIn{Tags: tags}); err != nil {
		return fmt.Errorf("failed to update tags: %w", err)
	}
	return nil
}

// markServiceCreatedOrUpdated records the CreatedOrUpdated conditions and clears the running marker.
func markServiceCreatedOrUpdated(obj v1alpha1.AivenManagedObject) {
	delete(obj.GetAnnotations(), instanceIsRunningAnnotation)

	// We don't really know if an instance is created or updated.
	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(obj.Conditions(), getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(obj.Conditions(), getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))
}

// isPowered: the CRD defaults powered to true; nil only occurs on objects built in Go.
func isPowered(spec *v1alpha1.ServiceCommonSpec) bool {
	return spec.Powered == nil || *spec.Powered
}

// maintenanceIn is the spec's maintenance window; a missing time is omitted.
func maintenanceIn(spec *v1alpha1.ServiceCommonSpec) *service.MaintenanceIn {
	if spec.MaintenanceWindowDow == "" && spec.MaintenanceWindowTime == "" {
		return nil
	}
	return &service.MaintenanceIn{Dow: spec.MaintenanceWindowDow, Time: NilIfZero(spec.MaintenanceWindowTime)}
}
