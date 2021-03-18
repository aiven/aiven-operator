// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	token := os.Getenv("AIVEN_TOKEN")
	if token == "" {
		Fail("cannot create Aiven API client, `AIVEN_TOKEN` is required")
	}

	if os.Getenv("AIVEN_PROJECT_NAME") == "" {
		Fail("`AIVEN_PROJECT_NAME` is required")
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = k8soperatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	// add Aiven secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aiven-token",
			Namespace: "default",
		},
		StringData: map[string]string{
			"token": os.Getenv("AIVEN_TOKEN"),
		},
	}

	err = k8sClient.Create(context.TODO(), secret)
	Expect(err).ToNot(HaveOccurred())

	// set-up roject
	err = (&ProjectReconciler{
		Controller{
			Client: k8sManager.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Project"),
			Scheme: k8sManager.GetScheme(),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up Kafka reconciler
	err = (&KafkaReconciler{
		Controller{
			Client: k8sManager.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Kafka"),
			Scheme: k8sManager.GetScheme(),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up PG reconciler
	err = (&PGReconciler{
		Controller{
			Client: k8sManager.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("PG"),
			Scheme: k8sManager.GetScheme(),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up KafkaConnect reconciler
	err = (&KafkaConnectReconciler{
		Controller{
			Client: k8sManager.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("KafkaConnect"),
			Scheme: k8sManager.GetScheme(),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()


	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

// EnsureDelete deletes the instance and waits for it to be gone or timeout
func ensureDelete(ctx context.Context, instance runtime.Object) {
	Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

	res, err := meta.Accessor(instance)
	Expect(err).NotTo(HaveOccurred())

	names := types.NamespacedName{Name: res.GetName(), Namespace: res.GetNamespace()}

	Eventually(func() bool {
		err := k8sClient.Get(ctx, names, instance)

		return apierrors.IsNotFound(err)
	}, time.Minute*1, time.Second*5, "wait for instance to be gone from k8s").Should(BeTrue())

}
