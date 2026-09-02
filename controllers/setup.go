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

// DefaultPollInterval is how often a Ready resource is re-reconciled when no interval is configured.
const DefaultPollInterval = 10 * time.Minute

const (
	minPollInterval = DefaultPollInterval
	maxPollInterval = 60 * time.Minute
)

// ValidatePollInterval reports whether d is an acceptable value for the --poll-interval flag.
func ValidatePollInterval(d time.Duration) error {
	if d == 0 {
		return nil
	}

	if d < minPollInterval || d > maxPollInterval {
		return fmt.Errorf(
			"poll interval %s is outside the accepted range %s-%s",
			d,
			minPollInterval,
			maxPollInterval,
		)
	}

	return nil
}

type SetupConfig struct {
	DefaultToken    string
	KubeVersion     string
	OperatorVersion string

	// PollInterval is how often a Ready resource is re-reconciled against the Aiven API.
	PollInterval time.Duration
}

// normalize applies built-in defaults to unset fields.
func (c *SetupConfig) normalize() {
	if c.PollInterval <= 0 {
		c.PollInterval = DefaultPollInterval
	}
}

func SetupControllers(mgr ctrl.Manager, cfg SetupConfig) error {
	cfg.normalize()

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
		"Clickhouse":                 newClickhouseReconciler,
		"ClickhouseDatabase":         newClickhouseDatabaseReconciler,
		"ClickhouseRole":             newClickhouseRoleReconciler,
		"ClickhouseUser":             newClickhouseUserReconciler,
		"ClickhouseGrant":            newClickhouseGrantReconciler,
		"ConnectionPool":             newConnectionPoolReconciler,
		"Database":                   newDatabaseReconciler,
		"Flink":                      newFlinkReconciler,
		"Grafana":                    newGrafanaReconciler,
		"Kafka":                      newKafkaReconciler,
		"KafkaACL":                   newKafkaACLReconciler,
		"KafkaNativeACL":             newKafkaNativeACLReconciler,
		"KafkaConnect":               newKafkaConnectReconciler,
		"KafkaConnector":             newKafkaConnectorReconciler,
		"KafkaQuota":                 newKafkaQuotaReconciler,
		"KafkaSchema":                newKafkaSchemaReconciler,
		"KafkaSchemaRegistryACL":     newKafkaSchemaRegistryACLReconciler,
		"KafkaTopic":                 newKafkaTopicReconciler,
		"MySQL":                      newMySQLReconciler,
		"OpenSearch":                 newOpenSearchReconciler,
		"OpenSearchACLConfig":        newOpenSearchACLConfigReconciler,
		"OrganizationProject":        newOrganizationProjectReconciler,
		"PostgreSQL":                 newPostgreSQLReconciler,
		"Project":                    newProjectReconciler,
		"ProjectVPC":                 newProjectVPCReconciler,
		"ServiceIntegration":         newServiceIntegrationReconciler,
		"ServiceIntegrationEndpoint": newServiceIntegrationEndpointReconciler,
		"ServiceUser":                newServiceUserReconciler,
		"UpgradePipelineStep":        newUpgradePipelineStepReconciler,
		"Valkey":                     newValkeyReconciler,
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
