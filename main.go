// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	k8soperatoraiveniov1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/aiven/aiven-k8s-operator/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(k8soperatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(k8soperatoraiveniov1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "00272a53.aiven.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ProjectReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Project"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Project")
		os.Exit(1)
	}

	if err = (&controllers.PGReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("PG"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PG")
		os.Exit(1)
	}

	if err = (&controllers.ConnectionPoolReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("ConnectionPool"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ConnectionPool")
		os.Exit(1)
	}

	if err = (&controllers.DatabaseReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Database"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Database")
		os.Exit(1)
	}

	if err = (&controllers.KafkaReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Kafka"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Kafka")
		os.Exit(1)
	}

	if err = (&controllers.ProjectVPCReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("ProjectVPC"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ProjectVPC")
		os.Exit(1)
	}

	if err = (&controllers.KafkaTopicReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("KafkaTopic"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KafkaTopic")
		os.Exit(1)
	}

	if err = (&controllers.KafkaACLReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("KafkaACL"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KafkaACL")
		os.Exit(1)
	}

	if err = (&controllers.KafkaConnectReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("KafkaConnect"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KafkaConnect")
		os.Exit(1)
	}

	if err = (&controllers.ServiceUserReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("ServiceUser"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceUser")
		os.Exit(1)
	}

	if err = (&controllers.KafkaSchemaReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("KafkaSchema"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KafkaSchema")
		os.Exit(1)
	}

	if err = (&controllers.ServiceIntegrationReconciler{
		Controller: controllers.Controller{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("ServiceIntegration"),
			Scheme: mgr.GetScheme(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ServiceIntegration")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&k8soperatorv1alpha1.Project{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Project")
			os.Exit(1)
		}
		if err = (&k8soperatorv1alpha1.PG{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "PG")
			os.Exit(1)
		}
		if err = (&k8soperatorv1alpha1.Database{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Database")
			os.Exit(1)
		}

		if err = (&k8soperatorv1alpha1.ConnectionPool{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ConnectionPool")
			os.Exit(1)
		}

		if err = (&k8soperatorv1alpha1.ServiceUser{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ServiceUser")
			os.Exit(1)
		}

		if err = (&k8soperatorv1alpha1.ProjectVPC{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ProjectVPC")
			os.Exit(1)
		}

		if err = (&k8soperatorv1alpha1.Kafka{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Kafka")
			os.Exit(1)
		}

		if err = (&k8soperatorv1alpha1.KafkaConnect{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KafkaConnect")
			os.Exit(1)
		}

		if err = (&k8soperatorv1alpha1.KafkaTopic{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KafkaTopic")
			os.Exit(1)
		}

		if err = (&k8soperatorv1alpha1.KafkaACL{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KafkaACL")
			os.Exit(1)
		}

		if err = (&k8soperatorv1alpha1.KafkaSchema{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "KafkaSchema")
			os.Exit(1)
		}
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
