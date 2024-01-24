package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strconv"
	"testing"

	"github.com/aiven/aiven-go-client/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

var testEnv *envtest.Environment
var testProject string
var testPrimaryCloudName string
var testSecondaryCloudName string
var testTertiaryCloudName string
var k8sClient client.Client
var avnClient *aiven.Client

const (
	secretRefName    = "aiven-token"
	secretRefKey     = "token"
	defaultNamespace = "default"
)

func TestMain(m *testing.M) {
	err := setupSuite()
	if err != nil {
		log.Fatal(err)
	}

	defer teardownSuite()
	os.Exit(m.Run())
}

func teardownSuite() {
	err := testEnv.Stop()
	if err != nil {
		log.Printf("failed to teardown: %s", err)
	}
}

func setupSuite() error {
	aivenToken := os.Getenv("AIVEN_TOKEN")
	if aivenToken == "" {
		return fmt.Errorf("missing AIVEN_TOKEN set")
	}

	testProject = os.Getenv("AIVEN_PROJECT_NAME")
	if testProject == "" {
		return fmt.Errorf("missing AIVEN_PROJECT_NAME set")
	}

	testPrimaryCloudName = os.Getenv("AIVEN_CLOUD_NAME")
	if testPrimaryCloudName == "" {
		testPrimaryCloudName = "google-europe-west1"
	}

	testSecondaryCloudName = os.Getenv("AIVEN_SECONDARY_CLOUD_NAME")
	if testSecondaryCloudName == "" {
		testSecondaryCloudName = "google-europe-west2"
	}

	testTertiaryCloudName = os.Getenv("AIVEN_TERTIARY_CLOUD_NAME")
	if testTertiaryCloudName == "" {
		testTertiaryCloudName = "google-europe-west3"
	}

	enableLogs, _ := strconv.ParseBool(os.Getenv("ENABLE_DEBUG_LOGGING"))
	if enableLogs {
		ctrl.SetLogger(zap.New(func(o *zap.Options) {
			o.Development = true
		}))
	}

	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{"../config/crd/bases"},
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{"../config/webhook"},
		},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return err
	}

	err = v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return err
	}

	avnClient, err = controllers.NewAivenClient(aivenToken)
	if err != nil {
		return err
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    testEnv.WebhookInstallOptions.LocalServingPort,
			CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
		}),
	})
	if err != nil {
		return err
	}
	k8sClient = mgr.GetClient()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretRefName,
			Namespace: defaultNamespace,
		},
		StringData: map[string]string{
			secretRefKey: aivenToken,
		},
	}

	ctx := context.Background()
	err = k8sClient.Create(ctx, secret)
	if err != nil {
		return err
	}

	err = controllers.SetupControllers(mgr, aivenToken)
	if err != nil {
		return fmt.Errorf("unable to setup controllers: %w", err)
	}

	err = v1alpha1.SetupWebhooks(mgr)
	if err != nil {
		return fmt.Errorf("unable to setup webhooks: %w", err)
	}

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		if err != nil {
			log.Fatal(err)
		}
	}()
	return nil
}

func recoverPanic(t *testing.T) {
	if err := recover(); err != nil {
		t.Logf("panicked: %s", err)
		t.Logf("stacktrace: \n%s", string(debug.Stack()))
		t.Fail()
	}
}
