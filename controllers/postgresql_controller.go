// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// PostgreSQLReconciler reconciles a PostgreSQL object
type PostgreSQLReconciler struct {
	Controller
}

// PostgreSQLHandler handles an Aiven PostgreSQL service
type PostgreSQLHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=postgresqls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=postgresqls/status,verbs=get;update;patch

func (r *PostgreSQLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, PostgreSQLHandler{}, &v1alpha1.PostgreSQL{})
}

func (r *PostgreSQLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PostgreSQL{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h PostgreSQLHandler) exists(avn *aiven.Client, pg *v1alpha1.PostgreSQL) (bool, error) {
	s, err := avn.Services.Get(pg.Spec.Project, pg.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h PostgreSQLHandler) createOrUpdate(avn *aiven.Client, i client.Object) error {
	pg, err := h.convert(i)
	if err != nil {
		return err
	}

	var prVPCID *string
	if pg.Spec.ProjectVPCID != "" {
		prVPCID = &pg.Spec.ProjectVPCID
	}

	exists, err := h.exists(avn, pg)
	if err != nil {
		return err
	}

	var reason string
	if !exists {
		_, err := avn.Services.Create(pg.Spec.Project, aiven.CreateServiceRequest{
			Cloud: pg.Spec.CloudName,
			MaintenanceWindow: getMaintenanceWindow(
				pg.Spec.MaintenanceWindowDow,
				pg.Spec.MaintenanceWindowTime),
			Plan:                pg.Spec.Plan,
			ProjectVPCID:        prVPCID,
			ServiceName:         pg.Name,
			ServiceType:         "pg",
			UserConfig:          UserConfigurationToAPI(pg.Spec.UserConfig).(map[string]interface{}),
			ServiceIntegrations: nil,
		})
		if err != nil {
			return err
		}

		reason = "Created"
	} else {
		_, err := avn.Services.Update(pg.Spec.Project, pg.Name, aiven.UpdateServiceRequest{
			Cloud: pg.Spec.CloudName,
			MaintenanceWindow: getMaintenanceWindow(
				pg.Spec.MaintenanceWindowDow,
				pg.Spec.MaintenanceWindowTime),
			Plan:         pg.Spec.Plan,
			ProjectVPCID: prVPCID,
			UserConfig:   UserConfigurationToAPI(pg.Spec.UserConfig).(map[string]interface{}),
			Powered:      true,
		})
		if err != nil {
			return err
		}

		reason = "Updated"
	}

	meta.SetStatusCondition(&pg.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&pg.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&pg.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(pg.GetGeneration(), formatIntBaseDecimal))

	return nil
}

// delete deletes Aiven PostgreSQL service
func (h PostgreSQLHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	pg, err := h.convert(i)
	if err != nil {
		return false, err
	}

	// Delete PostgreSQL on Aiven side
	if err := avn.Services.Delete(pg.Spec.Project, pg.Name); err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("aiven client delete pg error: %w", err)
	}

	return true, nil
}

func (h PostgreSQLHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	pg, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	s, err := avn.Services.Get(pg.Spec.Project, pg.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot get pg: %w", err)
	}

	pg.Status.State = s.State

	if s.State == "RUNNING" {
		meta.SetStatusCondition(&pg.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&pg.ObjectMeta, instanceIsRunningAnnotation, "true")
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getSecretName(pg),
			Namespace: pg.Namespace,
		},
		StringData: map[string]string{
			"PGHOST":       s.URIParams["host"],
			"PGPORT":       s.URIParams["port"],
			"PGDATABASE":   s.URIParams["dbname"],
			"PGUSER":       s.URIParams["user"],
			"PGPASSWORD":   s.URIParams["password"],
			"PGSSLMODE":    s.URIParams["sslmode"],
			"DATABASE_URI": s.URI,
		},
	}, nil
}

func (h PostgreSQLHandler) getSecretName(pg *v1alpha1.PostgreSQL) string {
	if pg.Spec.ConnInfoSecretTarget.Name != "" {
		return pg.Spec.ConnInfoSecretTarget.Name
	}
	return pg.Name
}

func (h PostgreSQLHandler) convert(i client.Object) (*v1alpha1.PostgreSQL, error) {
	pg, ok := i.(*v1alpha1.PostgreSQL)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to PostgreSQL")
	}

	return pg, nil
}

func (h PostgreSQLHandler) checkPreconditions(_ *aiven.Client, _ client.Object) (bool, error) {
	return true, nil
}
