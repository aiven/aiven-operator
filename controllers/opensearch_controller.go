// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
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
//+kubebuilder:rbac:groups=aiven.io,resources=opensearches/finalizers,verbs=get;list;watch;create;update;patch;delete

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
	return a.Spec.UserConfig
}

func (a *opensearchAdapter) newSecret(ctx context.Context, s *aiven.Service) (*corev1.Secret, error) {
	prefix := getSecretPrefix(a)
	stringData := map[string]string{
		prefix + "HOST":     s.URIParams["host"],
		prefix + "PASSWORD": s.URIParams["password"],
		prefix + "PORT":     s.URIParams["port"],
		prefix + "USER":     s.URIParams["user"],
		// todo: remove in future releases
		"HOST":     s.URIParams["host"],
		"PASSWORD": s.URIParams["password"],
		"PORT":     s.URIParams["port"],
		"USER":     s.URIParams["user"],
	}

	return newSecret(a, stringData, false), nil
}

func (a *opensearchAdapter) getServiceType() string {
	return "opensearch"
}

func (a *opensearchAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *opensearchAdapter) performUpgradeTaskIfNeeded(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, old *aiven.Service) error {
	return nil
}
