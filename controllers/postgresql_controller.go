// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/avast/retry-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	pguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/pg"
)

// PostgreSQLReconciler reconciles a PostgreSQL object
type PostgreSQLReconciler struct {
	Controller
}

func newPostgreSQLReconciler(c Controller) reconcilerType {
	return &PostgreSQLReconciler{Controller: c}
}

const waitForTaskToCompleteInterval = time.Second * 3

//+kubebuilder:rbac:groups=aiven.io,resources=postgresqls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=postgresqls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=postgresqls/finalizers,verbs=get;create;update

func (r *PostgreSQLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newPostgreSQLAdapter, r.Log), &v1alpha1.PostgreSQL{})
}

func (r *PostgreSQLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PostgreSQL{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newPostgreSQLAdapter(object client.Object) (serviceAdapter, error) {
	pg, ok := object.(*v1alpha1.PostgreSQL)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.PostgreSQL")
	}
	return &postgreSQLAdapter{pg}, nil
}

// postgreSQLAdapter handles an Aiven PostgreSQL service
type postgreSQLAdapter struct {
	*v1alpha1.PostgreSQL
}

func (a *postgreSQLAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *postgreSQLAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *postgreSQLAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *postgreSQLAdapter) getUserConfig() any {
	return a.Spec.UserConfig
}

func (a *postgreSQLAdapter) newSecret(_ context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
	prefix := getSecretPrefix(a)
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

	return newSecret(a, stringData, false), nil
}

func (a *postgreSQLAdapter) getServiceType() serviceType {
	return serviceTypePostgreSQL
}

func (a *postgreSQLAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

// performUpgradeTaskIfNeeded validates that a PostgreSQL version upgrade is possible before
// attempting the actual upgrade.
//
//  1. ServiceTaskCreate runs a compatibility check that validates if the upgrade is
//     possible. Multiple checks can be created without conflicts.
//  2. ServiceTaskGet polls for the check result. This step is required to unblock ServiceUpdate with
//     upgrade request.
//  3. If the check fails, the function returns the error from the
//     task result, preventing the upgrade attempt.
//  4. Once the check passes, ServiceUpdate can proceed. The actual upgrade happens async after ServiceUpdate.
func (a *postgreSQLAdapter) performUpgradeTaskIfNeeded(ctx context.Context, avnGen avngen.Client, old *service.ServiceGetOut) error {
	currentVersion := old.UserConfig["pg_version"].(string)
	targetUserConfig := a.getUserConfig().(*pguserconfig.PgUserConfig)
	if targetUserConfig == nil || targetUserConfig.PgVersion == nil {
		return nil
	}
	targetVersion := *targetUserConfig.PgVersion

	// No need to upgrade if pg_version hasn't changed
	if targetVersion == currentVersion {
		return nil
	}

	task, err := avnGen.ServiceTaskCreate(ctx, a.getServiceCommonSpec().Project, a.getObjectMeta().Name, &service.ServiceTaskCreateIn{
		TargetVersion: service.TargetVersionType(targetVersion),
		TaskType:      service.TaskTypeUpgradeCheck,
	})
	if err != nil {
		return fmt.Errorf("cannot create PG upgrade check task: %w", err)
	}

	errTaskInProgress := fmt.Errorf("task in progress")

	err = retry.Do(
		func() error {
			t, getErr := avnGen.ServiceTaskGet(
				ctx,
				a.getServiceCommonSpec().Project,
				a.getObjectMeta().Name,
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
					task.Result,
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

func (a *postgreSQLAdapter) createOrUpdateServiceSpecific(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}
