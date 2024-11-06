// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
	"github.com/aiven/aiven-operator/utils"
	//+kubebuilder:scaffold:imports
)

//go:generate go run ./generators/userconfigs/... --services mysql,cassandra,grafana,pg,kafka,redis,clickhouse,opensearch,kafka_connect
//go:generate go run ./generators/userconfigs/... --integrations autoscaler,clickhouse_kafka,clickhouse_postgresql,datadog,kafka_connect,kafka_logs,kafka_mirrormaker,logs,metrics,external_aws_cloudwatch_metrics
//go:generate go run ./generators/userconfigs/... --integration-endpoints autoscaler,datadog,external_aws_cloudwatch_logs,external_aws_cloudwatch_metrics,external_elasticsearch_logs,external_google_cloud_bigquery,external_google_cloud_logging,external_kafka,external_opensearch_logs,external_postgresql,external_schema_registry,jolokia,prometheus,rsyslog

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// operatorVersion is the current version of the operator.
var operatorVersion = "dev"

const port = 9443

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var development bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&development, "development", true, "Configures the logger to use a development config (stacktraces on warnings, no sampling)")
	opts := zap.Options{
		Development: development,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ctrlOptions := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   port,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "40db2fac.aiven.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	}

	// restrict the operator access to only specific namespaces, if `WATCHED_NAMESPACES` variable is set
	watchedNamespaces := os.Getenv("WATCHED_NAMESPACES")
	if watchedNamespaces != "" {
		namespaces := strings.Split(watchedNamespaces, ",")
		for _, namespace := range namespaces {
			if err := utils.ValidateNamespaceName(namespace); err != nil {
				setupLog.Error(err, "invalid namespace")
				os.Exit(1)
			}
		}
		setupLog.Info(fmt.Sprintf("Watching namespaces: %s", strings.Join(namespaces, ", ")))
		ctrlOptions.NewCache = cache.MultiNamespacedCacheBuilder(namespaces)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrlOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create discovery client")
		os.Exit(1)
	}
	kubeVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		setupLog.Error(err, "unable to get kube version")
		os.Exit(1)
	}

	defaultToken := os.Getenv("DEFAULT_AIVEN_TOKEN")
	err = controllers.SetupControllers(mgr, defaultToken, kubeVersion.String(), operatorVersion)
	if err != nil {
		setupLog.Error(err, "controllers setup error")
	}

	// Webhooks are enabled by default
	switch strings.ToLower(os.Getenv("ENABLE_WEBHOOKS")) {
	case "false", "0", "f":
	default:
		err = v1alpha1.SetupWebhooks(mgr)
		if err != nil {
			setupLog.Error(err, "unable to create webhook")
			os.Exit(1)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
