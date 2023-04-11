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

// KafkaConnectReconciler reconciles a KafkaConnect object
type KafkaConnectReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects/finalizers,verbs=update

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
	return &a.Spec.ServiceCommonSpec
}

func (a *kafkaConnectAdapter) getUserConfig() any {
	return &a.Spec.UserConfig
}

func (a *kafkaConnectAdapter) newSecret(_ *aiven.Service) (*corev1.Secret, error) {
	return nil, nil
}

func (a *kafkaConnectAdapter) getServiceType() string {
	return "kafka_connect"
}

func (a *kafkaConnectAdapter) getDiskSpace() string {
	return ""
}
