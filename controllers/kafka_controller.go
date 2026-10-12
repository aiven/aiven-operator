// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

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
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(r.Client, r.Recorder, newKafkaAdapter, r.Log), &v1alpha1.Kafka{})
}

func (r *KafkaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Kafka{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newKafkaAdapter(object client.Object) (serviceAdapter, error) {
	kafka, ok := object.(*v1alpha1.Kafka)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.Kafka")
	}
	return &kafkaAdapter{Kafka: kafka}, nil
}

// kafkaAdapter handles an Aiven Kafka service
type kafkaAdapter struct {
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

func (a *kafkaAdapter) newSecret(s *service.ServiceGetOut) *corev1.Secret {
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

	addKafkaEndpointDetails(stringData, s.Components, routeLegacy, prefix)

	for _, c := range s.Components {
		switch c.Component {
		case "kafka_connect":
			stringData[prefix+"CONNECT_HOST"] = c.Host
			stringData[prefix+"CONNECT_PORT"] = strconv.Itoa(c.Port)
		case "kafka_rest":
			stringData[prefix+"REST_HOST"] = c.Host
			stringData[prefix+"REST_PORT"] = strconv.Itoa(c.Port)
		}
	}

	return newSecret(a, stringData, false)
}

func (a *kafkaAdapter) getServiceType() serviceType {
	return serviceTypeKafka
}

func (a *kafkaAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *kafkaAdapter) performUpgradeTaskIfNeeded(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}

func (a *kafkaAdapter) createOrUpdateServiceSpecific(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}

// addKafkaEndpointDetails sets the SASL and schema registry keys from the primary components on the
// given route. routeLegacy keeps the last matching entry in API order, whatever its route.
func addKafkaEndpointDetails(details SecretDetails, components []service.ComponentOut, route service.RouteType, prefix string) {
	if route == routeLegacy {
		for _, c := range components {
			switch c.Component {
			case string(serviceTypeKafka):
				if isSaslComponent(c) {
					details[prefix+"SASL_HOST"] = c.Host
					details[prefix+"SASL_PORT"] = strconv.Itoa(c.Port)
				}
			case "schema_registry":
				details[prefix+"SCHEMA_REGISTRY_HOST"] = c.Host
				details[prefix+"SCHEMA_REGISTRY_PORT"] = strconv.Itoa(c.Port)
			}
		}
		return
	}

	// Blank the keys of a component missing on the route: the secret publisher merges, so a value
	// from a previous route would otherwise survive the switch.
	set := func(hostKey, portKey string, c *service.ComponentOut, ok bool) {
		if !ok {
			details[prefix+hostKey] = ""
			details[prefix+portKey] = ""
			return
		}
		details[prefix+hostKey] = c.Host
		details[prefix+portKey] = strconv.Itoa(c.Port)
	}

	c, ok := findSaslComponent(components, route)
	set("SASL_HOST", "SASL_PORT", c, ok)
	c, ok = findComponent(components, "schema_registry", route, nil)
	set("SCHEMA_REGISTRY_HOST", "SCHEMA_REGISTRY_PORT", c, ok)
}

func findSaslComponent(components []service.ComponentOut, route service.RouteType) (*service.ComponentOut, bool) {
	if c, ok := findComponent(components, string(serviceTypeKafka), route, func(c service.ComponentOut) bool {
		return isSaslComponent(c) && isProjectCaComponent(c)
	}); ok {
		return c, true
	}
	return findComponent(components, string(serviceTypeKafka), route, isSaslComponent)
}

func isSaslComponent(c service.ComponentOut) bool {
	return c.KafkaAuthenticationMethod == service.KafkaAuthenticationMethodTypeSasl
}

// isProjectCaComponent treats an unset kafka_ssl_ca as project CA, which the API omits by default.
func isProjectCaComponent(c service.ComponentOut) bool {
	return c.KafkaSslCa == "" || c.KafkaSslCa == service.KafkaSslCaTypeProjectCa
}

func isCertificateComponent(c service.ComponentOut) bool {
	return c.KafkaAuthenticationMethod == service.KafkaAuthenticationMethodTypeCertificate
}

// refreshKafkaEndpointDetails replaces optional Kafka endpoints in existing Secret data.
func refreshKafkaEndpointDetails(data map[string][]byte, components []service.ComponentOut, prefix string) {
	for _, key := range []string{
		prefix + "SASL_HOST",
		prefix + "SASL_PORT",
		prefix + "SCHEMA_REGISTRY_HOST",
		prefix + "SCHEMA_REGISTRY_PORT",
	} {
		delete(data, key)
	}
	details := make(SecretDetails)
	addKafkaEndpointDetails(details, components, routeLegacy, prefix)
	for key, value := range details {
		data[key] = []byte(value)
	}
}
