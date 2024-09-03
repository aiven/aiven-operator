// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// KafkaConnectReconciler reconciles a KafkaConnect object
type KafkaConnectReconciler struct {
	Controller
}

func newKafkaConnectReconciler(c Controller) reconcilerType {
	return &KafkaConnectReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects/finalizers,verbs=get;create;update

func (r *KafkaConnectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newKafkaConnectAdapter), &v1alpha1.KafkaConnect{})
}

func (r *KafkaConnectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaConnect{}).
		Complete(r)
}

func newKafkaConnectAdapter(_ *aiven.Client, object client.Object) (serviceAdapter, error) {
	kafkaConnect, ok := object.(*v1alpha1.KafkaConnect)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.KafkaConnect")
	}
	return &kafkaConnectAdapter{kafkaConnect}, nil
}

// kafkaConnectAdapter handles an Aiven KafkaConnect service
type kafkaConnectAdapter struct {
	*v1alpha1.KafkaConnect
}

func (a *kafkaConnectAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *kafkaConnectAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *kafkaConnectAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &v1alpha1.ServiceCommonSpec{BaseServiceFields: a.Spec.BaseServiceFields}
}

func (a *kafkaConnectAdapter) getUserConfig() any {
	return a.Spec.UserConfig
}

func (a *kafkaConnectAdapter) newSecret(ctx context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
	return nil, nil
}

func (a *kafkaConnectAdapter) getServiceType() string {
	return "kafka_connect"
}

func (a *kafkaConnectAdapter) getDiskSpace() string {
	return ""
}

func (a *kafkaConnectAdapter) GetConnInfoSecretTarget() v1alpha1.ConnInfoSecretTarget {
	return v1alpha1.ConnInfoSecretTarget{}
}

func (a *kafkaConnectAdapter) performUpgradeTaskIfNeeded(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, old *aiven.Service) error {
	return nil
}
