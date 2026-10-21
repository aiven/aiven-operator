// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// serviceKind is the per-service behavior of serviceController[T].
type serviceKind[T v1alpha1.ServiceObject] struct {
	// serviceType is the Aiven API service_type, e.g. "pg".
	serviceType serviceType

	// secretData returns the whole connection Secret with keys already prefixed. Nil: no Secret.
	secretData func(obj T, s *service.ServiceGetOut) map[string]string

	// projectCACert adds the project CA certificate as prefix+"CA_CERT" and under caCertExtraKeys verbatim.
	projectCACert   bool
	caCertExtraKeys []string

	// beforeUpdate runs before ServiceUpdate with the live service; an error aborts the update.
	beforeUpdate func(ctx context.Context, avn avngen.Client, obj T, old *service.ServiceGetOut) error

	// migration is nil for kinds without migration secret support.
	migration *serviceMigration[T]

	// preconditions run in order before Create and before Update of a new generation; first failure wins.
	preconditions []servicePrecondition[T]
}

// serviceMigration is the per-kind half of migration Secret support; the controller reads,
// polls and deletes the Secret itself.
type serviceMigration[T v1alpha1.ServiceObject] struct {
	// source returns spec.migrationSecretSource, nil when the CR has none.
	source func(obj T) *v1alpha1.MigrationSecretSource
	// merge returns a copy of the user config with the Secret data merged in; the CR is never mutated.
	merge func(obj T, data map[string]string) (any, error)
}

// servicePrecondition returns nil, errPreconditionNotMet (clears on its own, short requeue),
// errPreconditionExternal (only Aiven can clear it, condition + poll), or a real error.
type servicePrecondition[T v1alpha1.ServiceObject] func(ctx context.Context, avn avngen.Client, obj T) error

// serviceKindOption adjusts a kind at its definition site; see the with* constructors.
type serviceKindOption[T v1alpha1.ServiceObject] func(*serviceKind[T])

// newServiceKind builds a kind with the default preconditions; nil secretData means no connection Secret.
func newServiceKind[T v1alpha1.ServiceObject](
	t serviceType,
	secretData func(obj T, s *service.ServiceGetOut) map[string]string,
	opts ...serviceKindOption[T],
) serviceKind[T] {
	k := serviceKind[T]{
		serviceType:   t,
		secretData:    secretData,
		preconditions: []servicePrecondition[T]{checkReadReplicaSource[T], checkPowerOffBackup[T]},
	}
	for _, opt := range opts {
		opt(&k)
	}
	return k
}

// withProjectCACert adds the project CA certificate as prefix+"CA_CERT"; extraKeys are added unprefixed.
func withProjectCACert[T v1alpha1.ServiceObject](extraKeys ...string) serviceKindOption[T] {
	return func(k *serviceKind[T]) {
		k.projectCACert = true
		k.caCertExtraKeys = extraKeys
	}
}

// withBeforeUpdate runs fn before every ServiceUpdate.
func withBeforeUpdate[T v1alpha1.ServiceObject](
	fn func(ctx context.Context, avn avngen.Client, obj T, old *service.ServiceGetOut) error,
) serviceKindOption[T] {
	return func(k *serviceKind[T]) {
		k.beforeUpdate = fn
	}
}

// withMigrationSecret enables spec.migrationSecretSource for the kind.
func withMigrationSecret[T v1alpha1.ServiceObject](
	source func(obj T) *v1alpha1.MigrationSecretSource,
	merge func(obj T, data map[string]string) (any, error),
) serviceKindOption[T] {
	return func(k *serviceKind[T]) {
		k.migration = &serviceMigration[T]{source: source, merge: merge}
	}
}

// withPreconditions replaces the default preconditions with checks.
func withPreconditions[T v1alpha1.ServiceObject](checks ...servicePrecondition[T]) serviceKindOption[T] {
	return func(k *serviceKind[T]) {
		k.preconditions = checks
	}
}

// migrationSource returns the migration Secret reference, nil when the kind or the CR has none.
func (k serviceKind[T]) migrationSource(obj T) *v1alpha1.MigrationSecretSource {
	if k.migration == nil {
		return nil
	}
	return k.migration.source(obj)
}

func (k serviceKind[T]) checkPreconditions(ctx context.Context, avn avngen.Client, obj T) error {
	for _, check := range k.preconditions {
		if err := check(ctx, avn, obj); err != nil {
			return err
		}
	}
	return nil
}

// checkReadReplicaSource (default): each read-replica source must be running (NotMet) and have a backup (External).
func checkReadReplicaSource[T v1alpha1.ServiceObject](ctx context.Context, avn avngen.Client, obj T) error {
	spec := obj.ServiceCommonSpec()
	for _, s := range spec.ServiceIntegrations {
		if s.IntegrationType != service.IntegrationTypeReadReplica {
			continue
		}

		isOperational, err := checkServiceIsOperational(ctx, avn, spec.Project, s.SourceServiceName)
		if err != nil {
			return err
		}
		if !isOperational {
			// Resolves without user action.
			return fmt.Errorf("%w: read replica source service %q is not running yet", errPreconditionNotMet, s.SourceServiceName)
		}

		// Covers error "No valid backups for service"
		list, err := avn.ServiceBackupsGet(ctx, spec.Project, s.SourceServiceName)
		if err != nil {
			return err
		}
		if len(list.Backups) == 0 {
			return fmt.Errorf("%w: read replica source service %q has no backups yet", errPreconditionExternal, s.SourceServiceName)
		}
	}
	return nil
}

// checkPowerOffBackup (default): powered: false requires a backup (External); skip for kinds without backups.
func checkPowerOffBackup[T v1alpha1.ServiceObject](ctx context.Context, avn avngen.Client, obj T) error {
	spec := obj.ServiceCommonSpec()
	if fromAnyPointer(spec.Powered) {
		return nil
	}

	list, err := avn.ServiceBackupsGet(ctx, spec.Project, obj.GetName())
	if err != nil {
		return err
	}
	if len(list.Backups) == 0 {
		return fmt.Errorf("%w: service has no backups yet, cannot power it off", errPreconditionExternal)
	}
	return nil
}

// prefixKeys returns a copy of data with every key prefixed.
func prefixKeys(prefix string, data map[string]string) map[string]string {
	out := make(map[string]string, len(data))
	for k, v := range data {
		out[prefix+k] = v
	}
	return out
}
