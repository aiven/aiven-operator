// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
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

func newOpenSearchReconciler(c Controller) reconcilerType {
	return &OpenSearchReconciler{Controller: c}
}

type OpenSearchHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=opensearches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=opensearches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=opensearches/finalizers,verbs=get;create;update

func (r *OpenSearchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newOpenSearchAdapter, r.Log), &v1alpha1.OpenSearch{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenSearchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OpenSearch{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newOpenSearchAdapter(object client.Object) (serviceAdapter, error) {
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

func (a *opensearchAdapter) newSecret(_ context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
	prefix := getSecretPrefix(a)
	stringData := map[string]string{
		prefix + "URI":      s.ServiceUri,
		prefix + "HOST":     s.ServiceUriParams["host"],
		prefix + "PASSWORD": s.ServiceUriParams["password"],
		prefix + "PORT":     s.ServiceUriParams["port"],
		prefix + "USER":     s.ServiceUriParams["user"],
		// todo: remove in future releases
		"HOST":     s.ServiceUriParams["host"],
		"PASSWORD": s.ServiceUriParams["password"],
		"PORT":     s.ServiceUriParams["port"],
		"USER":     s.ServiceUriParams["user"],
	}

	return newSecret(a, stringData, false), nil
}

func (a *opensearchAdapter) getServiceType() serviceType {
	return serviceTypeOpenSearch
}

func (a *opensearchAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *opensearchAdapter) performUpgradeTaskIfNeeded(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}

func (a *opensearchAdapter) createOrUpdateServiceSpecific(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}
