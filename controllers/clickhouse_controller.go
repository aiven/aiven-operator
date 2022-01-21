// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClickhouseReconciler reconciles a Clickhouse object
type ClickhouseReconciler struct {
	Controller
}

type ClickhouseHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=clickhouses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouses/status,verbs=get;update;patch

func (r *ClickhouseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ClickhouseHandler{}, &v1alpha1.Clickhouse{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClickhouseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Clickhouse{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h ClickhouseHandler) createOrUpdate(avn *aiven.Client, i client.Object) error {
	os, err := h.convert(i)
	if err != nil {
		return err
	}

	var prVPCID *string
	if os.Spec.ProjectVPCID != "" {
		prVPCID = &os.Spec.ProjectVPCID
	}

	exists, err := h.exists(avn, os)
	if err != nil {
		return err
	}

	var reason string
	if !exists {
		_, err := avn.Services.Create(os.Spec.Project, aiven.CreateServiceRequest{
			Cloud: os.Spec.CloudName,
			MaintenanceWindow: getMaintenanceWindow(
				os.Spec.MaintenanceWindowDow,
				os.Spec.MaintenanceWindowTime),
			Plan:                os.Spec.Plan,
			ProjectVPCID:        prVPCID,
			ServiceName:         os.Name,
			ServiceType:         "clickhouse",
			UserConfig:          UserConfigurationToAPI(os.Spec.UserConfig).(map[string]interface{}),
			ServiceIntegrations: nil,
		})
		if err != nil {
			return err
		}

		reason = "Created"
	} else {
		_, err := avn.Services.Update(os.Spec.Project, os.Name, aiven.UpdateServiceRequest{
			Cloud: os.Spec.CloudName,
			MaintenanceWindow: getMaintenanceWindow(
				os.Spec.MaintenanceWindowDow,
				os.Spec.MaintenanceWindowTime),
			Plan:         os.Spec.Plan,
			ProjectVPCID: prVPCID,
			UserConfig:   UserConfigurationToAPI(os.Spec.UserConfig).(map[string]interface{}),
			Powered:      true,
		})
		if err != nil {
			return err
		}

		reason = "Updated"
	}

	meta.SetStatusCondition(&os.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&os.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&os.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(os.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h ClickhouseHandler) convert(o client.Object) (*v1alpha1.Clickhouse, error) {
	r, ok := o.(*v1alpha1.Clickhouse)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to clickhouse")
	}
	return r, nil
}

func (h ClickhouseHandler) exists(avn *aiven.Client, os *v1alpha1.Clickhouse) (bool, error) {
	s, err := avn.Services.Get(os.Spec.Project, os.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h ClickhouseHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	os, err := h.convert(i)
	if err != nil {
		return false, err
	}

	// Delete Clickhouse on Aiven side
	if err := avn.Services.Delete(os.Spec.Project, os.Name); err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("aiven client delete clickhouse error: %w", err)
	}

	return true, nil
}

func (h ClickhouseHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	os, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	s, err := avn.Services.Get(os.Spec.Project, os.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot get os: %w", err)
	}

	os.Status.State = s.State

	if s.State == "RUNNING" {
		meta.SetStatusCondition(&os.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&os.ObjectMeta, instanceIsRunningAnnotation, "true")
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getSecretName(os),
			Namespace: os.Namespace,
		},
		StringData: map[string]string{
			"HOST":     s.URIParams["host"],
			"PASSWORD": s.URIParams["password"],
			"PORT":     s.URIParams["port"],
			"USER":     s.URIParams["user"],
		},
	}, nil
}

func (h ClickhouseHandler) getSecretName(os *v1alpha1.Clickhouse) string {
	if os.Spec.ConnInfoSecretTarget.Name != "" {
		return os.Spec.ConnInfoSecretTarget.Name
	}
	return os.Name
}

func (h ClickhouseHandler) checkPreconditions(_ *aiven.Client, _ client.Object) (bool, error) {
	return true, nil
}
