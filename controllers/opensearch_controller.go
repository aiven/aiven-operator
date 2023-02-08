// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	"github.com/aiven/aiven-go-client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// OpenSearchReconciler reconciles a OpenSearch object
type OpenSearchReconciler struct {
	Controller
}

type OpenSearchHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=opensearches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=opensearches/status,verbs=get;update;patch

func (r *OpenSearchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newOpenSearchAdapter), &v1alpha1.OpenSearch{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenSearchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OpenSearch{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newOpenSearchAdapter(_ *aiven.Client, object client.Object) (serviceAdapter, error) {
	opensearch, ok := object.(*v1alpha1.OpenSearch)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.OpenSearch")
	}
	return &opensearchAdapter{opensearch}, nil
}

// opensearchAdapter handles an Aiven OpenSearch service
type opensearchAdapter struct {
	*v1alpha1.OpenSearch
}

func (a *opensearchAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *opensearchAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *opensearchAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *opensearchAdapter) getUserConfig() any {
	return &a.Spec.UserConfig
}

func (a *opensearchAdapter) newSecret(s *aiven.Service) (*corev1.Secret, error) {
	name := a.Spec.ConnInfoSecretTarget.Name
	if name == "" {
		name = a.Name
	}

	stringData := map[string]string{
		"HOST":     s.URIParams["host"],
		"PASSWORD": s.URIParams["password"],
		"PORT":     s.URIParams["port"],
		"USER":     s.URIParams["user"],
	}

	// Removes empties
	for k, v := range stringData {
		if v == "" {
			delete(stringData, k)
		}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: a.Namespace},
		StringData: stringData,
	}, nil
}

func (a *opensearchAdapter) getServiceType() string {
	return "opensearch"
}

func (a *opensearchAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}
