//go:build suite

package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"testing"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

func TestMain(m *testing.M) {
	if os.Getenv("LIST_ONLY") != "" {
		// For go test ./... -list=.
		// Lists test names without running them.
		m.Run()
		return
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	env, err := setupSuite(ctx)
	if err != nil {
		log.Fatal(err)
	}

	exitCode := 0
	defer func() {
		cancel()
		teardownSuite(env)
		os.Exit(exitCode)
	}()

	exitCode = m.Run()
}

func teardownSuite(env *envtest.Environment) {
	if sharedResources != nil {
		err := sharedResources.Destroy()
		if err != nil {
			log.Printf("shared resources teardown error: %s", err)
		}
	}

	if env == nil {
		return
	}

	err := env.Stop()
	if err != nil {
		log.Printf("failed to teardown: %s", err)
	}
}

func setupSuite(ctx context.Context) (*envtest.Environment, error) {
	cfg = new(testConfig)
	err := envconfig.Process("", cfg)
	if err != nil {
		return nil, err
	}

	if cfg.DebugLogging {
		ctrl.SetLogger(zap.New(func(o *zap.Options) {
			o.Development = true
		}))
	}

	env := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{"../config/crd/bases"},
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{"../config/webhook"},
		},
	}

	c, err := env.Start()
	if err != nil {
		return nil, err
	}

	err = v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	mgr, err := ctrl.NewManager(c, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			CertDir: env.WebhookInstallOptions.LocalServingCertDir,
			Port:    env.WebhookInstallOptions.LocalServingPort,
		}),
	})
	if err != nil {
		return nil, err
	}
	k8sClient = mgr.GetClient()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretRefName,
			Namespace: defaultNamespace,
		},
		StringData: map[string]string{
			secretRefKey: cfg.Token,
		},
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("unable to create discovery client: %w", err)
	}
	kubeVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("unable to get k8s version: %w", err)
	}

	avnGen, err = controllers.NewAivenGeneratedClient(cfg.Token, kubeVersion.String(), operatorVersion)
	if err != nil {
		return nil, err
	}

	err = k8sClient.Create(ctx, secret)
	if err != nil {
		return nil, err
	}

	err = controllers.SetupControllers(mgr, cfg.Token, kubeVersion.String(), operatorVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to setup controllers: %w", err)
	}

	err = v1alpha1.SetupWebhooks(mgr)
	if err != nil {
		return nil, fmt.Errorf("unable to setup webhooks: %w", err)
	}

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		if err != nil {
			log.Fatal(err)
		}
	}()

	sharedResources = NewSharedResources(ctx, k8sClient)
	return env, nil
}

func recoverPanic(t *testing.T) {
	if err := recover(); err != nil {
		t.Logf("panicked: %s", err)
		t.Logf("stacktrace: \n%s", string(debug.Stack()))
		t.Fail()
	}
}

func testCtx() (context.Context, func()) {
	return context.WithTimeout(context.Background(), cfg.TestCaseTimeout)
}
