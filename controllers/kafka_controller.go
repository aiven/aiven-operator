// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
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

//+kubebuilder:rbac:groups=aiven.io,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkas/finalizers,verbs=get;list;watch;create;update;patch;delete

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
	return &a.Spec.UserConfig
}

func (a *kafkaAdapter) newSecret(ctx context.Context, s *aiven.Service) (*corev1.Secret, error) {
	var userName, password string
	if len(s.Users) > 0 {
		userName = s.Users[0].Username
		password = s.Users[0].Password
	}

	caCert, err := a.avn.CA.Get(ctx, a.getServiceCommonSpec().Project)
	if err != nil {
		return nil, fmt.Errorf("aiven client error %w", err)
	}

	prefix := getSecretPrefix(a)
	stringData := map[string]string{
		prefix + "HOST":        s.URIParams["host"],
		prefix + "PORT":        s.URIParams["port"],
		prefix + "PASSWORD":    password,
		prefix + "USERNAME":    userName,
		prefix + "ACCESS_CERT": s.ConnectionInfo.KafkaAccessCert,
		prefix + "ACCESS_KEY":  s.ConnectionInfo.KafkaAccessKey,
		prefix + "CA_CERT":     caCert,
		// todo: remove in future releases
		"HOST":        s.URIParams["host"],
		"PORT":        s.URIParams["port"],
		"PASSWORD":    password,
		"USERNAME":    userName,
		"ACCESS_CERT": s.ConnectionInfo.KafkaAccessCert,
		"ACCESS_KEY":  s.ConnectionInfo.KafkaAccessKey,
		"CA_CERT":     caCert,
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
