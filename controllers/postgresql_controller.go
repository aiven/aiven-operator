// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/avast/retry-go"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	pguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/pg"
)

const waitForTaskToCompleteInterval = time.Second * 3

//+kubebuilder:rbac:groups=aiven.io,resources=postgresqls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=postgresqls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=postgresqls/finalizers,verbs=get;create;update

func newPostgreSQLReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.PostgreSQL] {
			return newServiceController(c, avnGen, postgreSQLKind)
		},
		nil,
	)
}

// postgreSQLKind is the serviceController definition for PostgreSQL.
var postgreSQLKind = newServiceKind(serviceTypePostgreSQL, postgreSQLSecretData,
	withProjectCACert[*v1alpha1.PostgreSQL](),
	withBeforeUpdate(postgreSQLUpgradeCheck),
	withMigrationSecret(func(o *v1alpha1.PostgreSQL) *v1alpha1.MigrationSecretSource { return o.Spec.MigrationSecretSource }, postgreSQLMigrationUserConfig),
)

// postgreSQLSecretData returns the prefixed connection secret keys.
func postgreSQLSecretData(obj *v1alpha1.PostgreSQL, s *service.ServiceGetOut) map[string]string {
	prefix := getSecretPrefix(obj)
	stringData := map[string]string{
		prefix + "HOST":         s.ServiceUriParams["host"],
		prefix + "PORT":         s.ServiceUriParams["port"],
		prefix + "DATABASE":     s.ServiceUriParams["dbname"],
		prefix + "USER":         s.ServiceUriParams["user"],
		prefix + "PASSWORD":     s.ServiceUriParams["password"],
		prefix + "SSLMODE":      s.ServiceUriParams["sslmode"],
		prefix + "DATABASE_URI": s.ServiceUri,
		// todo: remove in future releases
		"PGHOST":       s.ServiceUriParams["host"],
		"PGPORT":       s.ServiceUriParams["port"],
		"PGDATABASE":   s.ServiceUriParams["dbname"],
		"PGUSER":       s.ServiceUriParams["user"],
		"PGPASSWORD":   s.ServiceUriParams["password"],
		"PGSSLMODE":    s.ServiceUriParams["sslmode"],
		"DATABASE_URI": s.ServiceUri,
	}

	return stringData
}

// postgreSQLMigrationUserConfig merges pre-resolved migration Secret data into a clone of the user config.
func postgreSQLMigrationUserConfig(obj *v1alpha1.PostgreSQL, data map[string]string) (any, error) {
	ref := obj.Spec.MigrationSecretSource
	if ref == nil {
		return obj.Spec.UserConfig, nil
	}

	port, err := parseMigrationPort(data["port"])
	if err != nil {
		return nil, err
	}

	host := data["host"]
	if host == "" {
		return nil, fmt.Errorf("migration secret %q must contain \"host\"", ref.Name)
	}

	m := &pguserconfig.Migration{
		Host: host,
		Port: port,
	}
	if v, ok := data["password"]; ok {
		m.Password = new(v)
	}
	if v, ok := data["dbname"]; ok {
		m.Dbname = new(v)
	}
	if v, ok := data["username"]; ok {
		m.Username = new(v)
	}
	if v, ok := data["ssl"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("migration secret %q: invalid ssl value %q: %w", ref.Name, v, err)
		}
		m.Ssl = new(b)
	}
	if v, ok := data["method"]; ok {
		m.Method = new(v)
	}
	if v, ok := data["ignore_dbs"]; ok {
		m.IgnoreDbs = new(v)
	}
	if v, ok := data["ignore_roles"]; ok {
		m.IgnoreRoles = new(v)
	}

	// Build a shallow copy of the user config with migration from the secret.
	// The original CR spec is never mutated.
	cfg := &pguserconfig.PgUserConfig{}
	if obj.Spec.UserConfig != nil {
		clone := *obj.Spec.UserConfig
		cfg = &clone
	}
	cfg.Migration = m
	return cfg, nil
}

// postgreSQLUpgradeCheck runs the upgrade check task when pg_version changes.
func postgreSQLUpgradeCheck(ctx context.Context, avnGen avngen.Client, obj *v1alpha1.PostgreSQL, old *service.ServiceGetOut) error {
	currentVersion, _ := old.UserConfig["pg_version"].(string)
	targetUserConfig := obj.Spec.UserConfig
	if targetUserConfig == nil || targetUserConfig.PgVersion == nil {
		return nil
	}
	targetVersion := *targetUserConfig.PgVersion

	// No need to upgrade if pg_version hasn't changed
	if targetVersion == currentVersion {
		return nil
	}

	task, err := avnGen.ServiceTaskCreate(ctx, obj.Spec.Project, obj.Name, &service.ServiceTaskCreateIn{
		UpgradeCheck: &service.UpgradeCheckIn{TargetVersion: targetVersion},
		TaskType:     service.TaskTypeUpgradeCheck,
	})
	if err != nil {
		return fmt.Errorf("cannot create PG upgrade check task: %w", err)
	}

	errTaskInProgress := fmt.Errorf("task in progress")

	err = retry.Do(
		func() error {
			t, getErr := avnGen.ServiceTaskGet(
				ctx,
				obj.Spec.Project,
				obj.Name,
				task.TaskId,
			)
			if getErr != nil {
				return getErr
			}

			if t.Success {
				return nil
			}

			if t.Result != "" {
				return fmt.Errorf(
					"PG service upgrade check error, version upgrade from %s to %s, result: %s",
					currentVersion,
					targetVersion,
					t.Result,
				)
			}

			return errTaskInProgress
		},
		retry.RetryIf(func(err error) bool {
			return isServerError(err) || isNotFound(err) || errors.Is(err, errTaskInProgress)
		}),
		retry.Context(ctx),
		retry.Attempts(3), //nolint:mnd
		retry.Delay(waitForTaskToCompleteInterval),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
	)

	return err
}
