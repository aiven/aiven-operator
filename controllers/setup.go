package controllers

import (
	"fmt"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type reconcilerBuilder func(controller Controller) reconcilerType

type reconcilerType interface {
	reconcile.Reconciler
	SetupWithManager(mgr ctrl.Manager) error
}

const defaultPollInterval = 10 * time.Minute

type SetupConfig struct {
	DefaultToken    string
	KubeVersion     string
	OperatorVersion string
	PollInterval    time.Duration
}

func SetupControllers(mgr ctrl.Manager, defaultToken, kubeVersion, operatorVersion string) error {
	return SetupControllersWithConfig(mgr, SetupConfig{
		DefaultToken:    defaultToken,
		KubeVersion:     kubeVersion,
		OperatorVersion: operatorVersion,
	})
}

func SetupControllersWithConfig(mgr ctrl.Manager, cfg SetupConfig) error {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}

	if err := (&SecretFinalizerGCController{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SecretFinalizerGCController"),
	}).SetupWithManager(mgr, cfg.DefaultToken != ""); err != nil {
		return fmt.Errorf("controller SecretFinalizerGCController: %w", err)
	}

	if err := (&SecretWatchController{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SecretWatchController"),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("controller SecretWatchController: %w", err)
	}

	builders := map[string]reconcilerBuilder{
		"AlloyDBOmni":                      newAlloyDBOmniReconciler,
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
		"KafkaNativeACL":                   newKafkaNativeACLReconciler,
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
		"ServiceIntegration":               newServiceIntegrationReconciler,
		"ServiceIntegrationEndpoint":       newServiceIntegrationEndpointReconciler,
		"ServiceUser":                      newServiceUserReconciler,
		"Valkey":                           newValkeyReconciler,
	}

	for k, v := range builders {
		err := v(newController(mgr, k, cfg)).SetupWithManager(mgr)
		if err != nil {
			return fmt.Errorf("controller %s setup error: %w", k, err)
		}
	}

	//+kubebuilder:scaffold:builder
	return nil
}

func newController(mgr ctrl.Manager, name string, cfg SetupConfig) Controller {
	return Controller{
		Client:          mgr.GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName(name),
		Scheme:          mgr.GetScheme(),
		Recorder:        mgr.GetEventRecorderFor(strings.ToLower(name) + "-reconciler"),
		DefaultToken:    cfg.DefaultToken,
		KubeVersion:     cfg.KubeVersion,
		OperatorVersion: cfg.OperatorVersion,
		PollInterval:    cfg.PollInterval,
	}
}
