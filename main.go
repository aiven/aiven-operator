// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package main

import (
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
	//+kubebuilder:scaffold:imports
)

//go:generate go run ./userconfigs_generator/... --services mysql,cassandra,grafana,pg,kafka,redis,clickhouse

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const port = 9443

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
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

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
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
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	defaultToken := os.Getenv("DEFAULT_AIVEN_TOKEN")

	if err = (&controllers.SecretFinalizerGCController{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SecretFinalizerGCController"),
	}).SetupWithManager(mgr, len(defaultToken) > 0); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SecretFinalizerGCController")
		os.Exit(1)
	}

	if err = (&controllers.ProjectReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("Project"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("project-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Project")
		os.Exit(1)
	}

	if err = (&controllers.PostgreSQLReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("PostgreSQL"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("postgresql-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PostgreSQL")
		os.Exit(1)
	}

	if err = (&controllers.ConnectionPoolReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("ConnectionPool"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("connection-pool-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ConnectionPool")
		os.Exit(1)
	}

	if err = (&controllers.DatabaseReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("Database"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("database-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Database")
		os.Exit(1)
	}

	if err = (&controllers.KafkaReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("Kafka"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("kafka-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Kafka")
		os.Exit(1)
	}

	if err = (&controllers.ProjectVPCReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("ProjectVPC"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("project-vpc-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ProjectVPC")
		os.Exit(1)
	}

	if err = (&controllers.KafkaTopicReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("KafkaTopic"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("kafka-topic-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KafkaTopic")
		os.Exit(1)
	}

	if err = (&controllers.KafkaACLReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("KafkaACL"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("kafka-acl-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KafkaACL")
		os.Exit(1)
	}

	if err = (&controllers.KafkaConnectReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("KafkaConnect"),
			Recorder:     mgr.GetEventRecorderFor("kafka-connect-reconciler"),
			Scheme:       mgr.GetScheme(),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KafkaConnect")
		os.Exit(1)
	}

	if err = (&controllers.ServiceUserReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("ServiceUser"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("service-user-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceUser")
		os.Exit(1)
	}

	if err = (&controllers.KafkaSchemaReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("KafkaSchema"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("kafka-schema-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KafkaSchema")
		os.Exit(1)
	}

	if err = (&controllers.ServiceIntegrationReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("ServiceIntegration"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("service-integration-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceIntegration")
		os.Exit(1)
	}
	if err = (&controllers.KafkaConnectorReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("KafkaConnector"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("kafka-connector-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KafkaConnector")
		os.Exit(1)
	}

	if err = (&controllers.RedisReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("Redis"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("redis-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Redis")
		os.Exit(1)
	}

	if err = (&controllers.OpenSearchReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("OpenSearch"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("opensearch-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OpenSearch")
		os.Exit(1)
	}

	if err = (&controllers.ClickhouseReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("Clickhouse"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("clickhouse-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Clickhouse")
		os.Exit(1)
	}

	if err = (&controllers.ClickhouseUserReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("ClickhouseUser"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("clickhouse-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClickhouseUser")
		os.Exit(1)
	}

	if err = (&controllers.MySQLReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("MySQL"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("mysql-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MySQL")
		os.Exit(1)
	}

	if err = (&controllers.CassandraReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("Cassandra"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("cassandra-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cassandra")
		os.Exit(1)
	}

	if err = (&controllers.GrafanaReconciler{
		Controller: controllers.Controller{
			Client:       mgr.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("Grafana"),
			Scheme:       mgr.GetScheme(),
			Recorder:     mgr.GetEventRecorderFor("grafana-reconciler"),
			DefaultToken: defaultToken,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Grafana")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&v1alpha1.Project{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Project")
			os.Exit(1)
		}
		if err = (&v1alpha1.PostgreSQL{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "PostgreSQL")
			os.Exit(1)
		}
		if err = (&v1alpha1.Database{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Database")
			os.Exit(1)
		}

		if err = (&v1alpha1.ConnectionPool{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ConnectionPool")
			os.Exit(1)
		}

		if err = (&v1alpha1.ServiceUser{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ServiceUser")
			os.Exit(1)
		}

		if err = (&v1alpha1.Kafka{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Kafka")
			os.Exit(1)
		}

		if err = (&v1alpha1.KafkaConnect{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KafkaConnect")
			os.Exit(1)
		}

		if err = (&v1alpha1.KafkaTopic{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KafkaTopic")
			os.Exit(1)
		}

		if err = (&v1alpha1.KafkaACL{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KafkaACL")
			os.Exit(1)
		}

		if err = (&v1alpha1.KafkaSchema{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KafkaSchema")
			os.Exit(1)
		}

		if err = (&v1alpha1.ServiceIntegration{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ServiceIntegration")
			os.Exit(1)
		}

		if err = (&v1alpha1.KafkaConnector{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KafkaConnector")
			os.Exit(1)
		}

		if err = (&v1alpha1.Redis{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Redis")
			os.Exit(1)
		}

		if err = (&v1alpha1.OpenSearch{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpenSearch")
			os.Exit(1)
		}

		if err = (&v1alpha1.Clickhouse{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Clickhouse")
			os.Exit(1)
		}

		if err = (&v1alpha1.ClickhouseUser{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ClickhouseUser")
			os.Exit(1)
		}
		if err = (&v1alpha1.MySQL{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "MySQL")
			os.Exit(1)
		}
		if err = (&v1alpha1.Cassandra{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Cassandra")
			os.Exit(1)
		}
		if err = (&v1alpha1.Grafana{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Grafana")
			os.Exit(1)
		}
	}

	//+kubebuilder:scaffold:builder

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
