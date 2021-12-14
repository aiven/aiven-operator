// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

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

// OpenSearchReconciler reconciles a OpenSearch object
type OpenSearchReconciler struct {
	Controller
}

type OpenSearchHandler struct{}

func (h OpenSearchHandler) createOrUpdate(avn *aiven.Client, i client.Object) error {
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
			ServiceType:         "opensearch",
			UserConfig:          UserConfigurationToAPI(os.Spec.UserConfig).(map[string]interface{}),
			ServiceIntegrations: nil,
			DiskSpaceMB:         v1alpha1.ConvertDiscSpace(os.Spec.DiskSpace),
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
			DiskSpaceMB:  v1alpha1.ConvertDiscSpace(os.Spec.DiskSpace),
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

func (h OpenSearchHandler) convert(o client.Object) (*v1alpha1.OpenSearch, error) {
	r, ok := o.(*v1alpha1.OpenSearch)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to OpenSearch")
	}
	return r, nil
}

func (h OpenSearchHandler) exists(avn *aiven.Client, os *v1alpha1.OpenSearch) (bool, error) {
	s, err := avn.Services.Get(os.Spec.Project, os.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h OpenSearchHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	os, err := h.convert(i)
	if err != nil {
		return false, err
	}

	// Delete OpenSearch on Aiven side
	if err := avn.Services.Delete(os.Spec.Project, os.Name); err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("aiven client delete opensearch error: %w", err)
	}

	return true, nil
}

func (h OpenSearchHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
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

func (h OpenSearchHandler) getSecretName(os *v1alpha1.OpenSearch) string {
	if os.Spec.ConnInfoSecretTarget.Name != "" {
		return os.Spec.ConnInfoSecretTarget.Name
	}
	return os.Name
}

func (h OpenSearchHandler) checkPreconditions(_ *aiven.Client, _ client.Object) (bool, error) {
	return true, nil
}

//+kubebuilder:rbac:groups=aiven.io,resources=opensearches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=opensearches/status,verbs=get;update;patch

func (r *OpenSearchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, OpenSearchHandler{}, &v1alpha1.OpenSearch{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenSearchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OpenSearch{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
