package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"testing"
	"time"

	"github.com/aiven/aiven-go-client/v2"
	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

var (
	cfg       *testConfig
	k8sClient client.Client
	avnClient *aiven.Client
)

const (
	secretRefName = "aiven-token"
	secretRefKey  = "token"
)

type testConfig struct {
	Token              string        `envconfig:"AIVEN_TOKEN" required:"true"`
	Project            string        `envconfig:"AIVEN_PROJECT_NAME" required:"true"`
	PrimaryCloudName   string        `envconfig:"AIVEN_CLOUD_NAME" default:"google-europe-west1"`
	SecondaryCloudName string        `envconfig:"AIVEN_SECONDARY_CLOUD_NAME" default:"google-europe-west2"`
	TertiaryCloudName  string        `envconfig:"AIVEN_TERTIARY_CLOUD_NAME" default:"google-europe-west3"`
	DebugLogging       bool          `envconfig:"ENABLE_DEBUG_LOGGING"`
	TestCaseTimeout    time.Duration `envconfig:"TEST_CASE_TIMEOUT" default:"20m"`
}

func TestMain(m *testing.M) {
	if os.Getenv("LIST_ONLY") != "" {
		// For go test ./... -list=.
		// Lists test names without running them.
		m.Run()
		return
	}

	env, err := setupSuite()
	if err != nil {
		log.Fatal(err)
	}

	defer teardownSuite(env)
	os.Exit(m.Run())
}

func teardownSuite(env *envtest.Environment) {
	if env == nil {
		return
	}

	err := env.Stop()
	if err != nil {
		log.Printf("failed to teardown: %s", err)
	}
}

func setupSuite() (*envtest.Environment, error) {
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

	avnClient, err = controllers.NewAivenClient(cfg.Token)
	if err != nil {
		return nil, err
	}

	mgr, err := ctrl.NewManager(c, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
		CertDir:            env.WebhookInstallOptions.LocalServingCertDir,
		Port:               env.WebhookInstallOptions.LocalServingPort,
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

	ctx, cancel := testCtx()
	defer cancel()

	err = k8sClient.Create(ctx, secret)
	if err != nil {
		return nil, err
	}

	err = controllers.SetupControllers(mgr, cfg.Token)
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
