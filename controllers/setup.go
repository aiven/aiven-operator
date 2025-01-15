package controllers

import (
	"fmt"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type reconcilerBuilder func(controller Controller) reconcilerType

type reconcilerType interface {
	reconcile.Reconciler
	SetupWithManager(mgr ctrl.Manager) error
}

func SetupControllers(mgr ctrl.Manager, defaultToken, kubeVersion, operatorVersion string) error {
	if err := (&SecretFinalizerGCController{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SecretFinalizerGCController"),
	}).SetupWithManager(mgr, defaultToken != ""); err != nil {
		return fmt.Errorf("controller SecretFinalizerGCController: %w", err)
	}

	builders := map[string]reconcilerBuilder{
		"Cassandra":                        newCassandraReconciler,
		"Clickhouse":                       newClickhouseReconciler,
		"ClickhouseDatabase":               newClickhouseDatabaseReconciler,
		"ClickhouseRole":                   newClickhouseRoleReconciler,
		"ClickhouseUser":                   newClickhouseUserReconciler,
		"ClickhouseGrant":                  newClickhouseGrantReconciler,
		"ConnectionPool":                   newConnectionPoolReconciler,
		"Database":                         newDatabaseReconciler,
		"Flink":                            newFlinkReconciler,
		"Grafana":                          newGrafanaReconciler,
		"Kafka":                            newKafkaReconciler,
		"KafkaACL":                         newKafkaACLReconciler,
		"KafkaConnect":                     newKafkaConnectReconciler,
		"KafkaConnector":                   newKafkaConnectorReconciler,
		"KafkaSchema":                      newKafkaSchemaReconciler,
		"KafkaSchemaRegistryACLReconciler": newKafkaSchemaRegistryACLReconciler,
		"KafkaTopic":                       newKafkaTopicReconciler,
		"MySQL":                            newMySQLReconciler,
		"OpenSearch":                       newOpenSearchReconciler,
		"PostgreSQL":                       newPostgreSQLReconciler,
		"Project":                          newProjectReconciler,
		"ProjectVPC":                       newProjectVPCReconciler,
		"Redis":                            newRedisReconciler,
		"ServiceIntegration":               newServiceIntegrationReconciler,
		"ServiceIntegrationEndpoint":       newServiceIntegrationEndpointReconciler,
		"ServiceUser":                      newServiceUserReconciler,
		"Valkey":                           newValkeyReconciler,
	}

	for k, v := range builders {
		err := v(newController(mgr, k, defaultToken, kubeVersion, operatorVersion)).SetupWithManager(mgr)
		if err != nil {
			return fmt.Errorf("controller %s setup error: %w", k, err)
		}
	}

	//+kubebuilder:scaffold:builder
	return nil
}

func newController(mgr ctrl.Manager, name, defaultToken, kubeVersion, operatorVersion string) Controller {
	return Controller{
		Client:          mgr.GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName(name),
		Scheme:          mgr.GetScheme(),
		Recorder:        mgr.GetEventRecorderFor(strings.ToLower(name) + "-reconciler"),
		DefaultToken:    defaultToken,
		KubeVersion:     kubeVersion,
		OperatorVersion: operatorVersion,
	}
}
