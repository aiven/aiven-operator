// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// KafkaReconciler reconciles a Kafka object
type KafkaReconciler struct {
	Controller
}

func newKafkaReconciler(c Controller) reconcilerType {
	return &KafkaReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkas/finalizers,verbs=get;create;update

func (r *KafkaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newKafkaAdapter), &v1alpha1.Kafka{})
}

func (r *KafkaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Kafka{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newKafkaAdapter(avn *aiven.Client, object client.Object) (serviceAdapter, error) {
	kafka, ok := object.(*v1alpha1.Kafka)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.Kafka")
	}
	return &kafkaAdapter{avn: avn, Kafka: kafka}, nil
}

// kafkaAdapter handles an Aiven Kafka service
type kafkaAdapter struct {
	avn *aiven.Client
	*v1alpha1.Kafka
}

func (a *kafkaAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *kafkaAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *kafkaAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *kafkaAdapter) getUserConfig() any {
	return a.Spec.UserConfig
}

func (a *kafkaAdapter) newSecret(ctx context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
	var userName, password string
	if len(s.Users) > 0 {
		userName = s.Users[0].Username
		password = s.Users[0].Password
	}

	prefix := getSecretPrefix(a)
	stringData := map[string]string{
		prefix + "HOST":                s.ServiceUriParams["host"],
		prefix + "PORT":                s.ServiceUriParams["port"],
		prefix + "PASSWORD":            password,
		prefix + "USERNAME":            userName,
		prefix + "ACCESS_CERT":         *s.ConnectionInfo.KafkaAccessCert,
		prefix + "ACCESS_KEY":          *s.ConnectionInfo.KafkaAccessKey,
		prefix + "REST_URI":            *s.ConnectionInfo.KafkaRestUri,
		prefix + "SCHEMA_REGISTRY_URI": *s.ConnectionInfo.SchemaRegistryUri,
		// todo: remove in future releases
		"HOST":        s.ServiceUriParams["host"],
		"PORT":        s.ServiceUriParams["port"],
		"PASSWORD":    password,
		"USERNAME":    userName,
		"ACCESS_CERT": *s.ConnectionInfo.KafkaAccessCert,
		"ACCESS_KEY":  *s.ConnectionInfo.KafkaAccessKey,
	}

	for _, c := range s.Components {
		switch c.Component {
		case "kafka":
			if c.KafkaAuthenticationMethod == "sasl" {
				stringData[prefix+"SASL_HOST"] = c.Host
				stringData[prefix+"SASL_PORT"] = strconv.Itoa(c.Port)
			}
		case "schema_registry":
			stringData[prefix+"SCHEMA_REGISTRY_HOST"] = c.Host
			stringData[prefix+"SCHEMA_REGISTRY_PORT"] = strconv.Itoa(c.Port)
		case "kafka_connect":
			stringData[prefix+"CONNECT_HOST"] = c.Host
			stringData[prefix+"CONNECT_PORT"] = strconv.Itoa(c.Port)
		case "kafka_rest":
			stringData[prefix+"REST_HOST"] = c.Host
			stringData[prefix+"REST_PORT"] = strconv.Itoa(c.Port)
		}
	}

	return newSecret(a, stringData, false), nil
}

func (a *kafkaAdapter) getServiceType() string {
	return "kafka"
}

func (a *kafkaAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *kafkaAdapter) performUpgradeTaskIfNeeded(ctx context.Context, avn avngen.Client, old *service.ServiceGetOut) error {
	return nil
}

func (a *kafkaAdapter) createOrUpdateServiceSpecific(ctx context.Context, avn avngen.Client, old *service.ServiceGetOut) error {
	return nil
}
