// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aiven/aiven-go-client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	aiveniov1alpha1 "github.com/aiven/aiven-operator/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

var cfg *rest.Config
var k8sClient client.Client
var aivenClient *aiven.Client
var testEnv *envtest.Environment

const secretRefName = "aiven-token"
const secretRefKey = "token"

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	aivenToken := os.Getenv("AIVEN_TOKEN")
	if aivenToken == "" {
		Fail("cannot createOrUpdate Aiven API client, `AIVEN_TOKEN` is required")
	}

	if os.Getenv("AIVEN_PROJECT_NAME") == "" {
		Fail("`AIVEN_PROJECT_NAME` is required")
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = aiveniov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	aivenClient, err = aiven.NewTokenClient(aivenToken, operatorUserAgent)
	Expect(err).NotTo(HaveOccurred())
	Expect(aivenClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	// add Aiven secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretRefName,
			Namespace: "default",
		},
		StringData: map[string]string{
			secretRefKey: aivenToken,
		},
	}

	err = k8sClient.Create(context.TODO(), secret)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			Expect(err).ToNot(HaveOccurred())
		}
	}

	// set-up Project
	err = (&ProjectReconciler{
		Controller: Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("Project"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("project-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up ProjectVPC reconciler
	err = (&ProjectVPCReconciler{
		Controller: Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("ProjectVPC"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("projectvpc-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up Kafka reconciler
	err = (&KafkaReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("Kafka"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("kafka-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up PostgreSQL reconciler
	err = (&PostgreSQLReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("PostgreSQL"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("postgresql-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up KafkaConnect reconciler
	err = (&KafkaConnectReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("KafkaConnect"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("kafka-connect-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up Database reconciler
	err = (&DatabaseReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("Database"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("database-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up ConnectionPool reconciler
	err = (&ConnectionPoolReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("ConnectionPool"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("connection-pool-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up ServiceUser reconciler
	err = (&ServiceUserReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("ServiceUser"),
			Recorder: k8sManager.GetEventRecorderFor("service-user-reconciler"),
			Scheme:   k8sManager.GetScheme(),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up KafkaTopic reconciler
	err = (&KafkaTopicReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("KafkaTopic"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("kafka-topic-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up KafkaACL reconciler
	err = (&KafkaACLReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("KafkaACL"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("kafka-acl-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up KafkaSchema reconciler
	err = (&KafkaSchemaReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("KafkaSchema"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("kafka-schema-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up ServiceIntegration reconciler
	err = (&ServiceIntegrationReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("ServiceIntegration"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("service-integration-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up Redis reconciler
	err = (&RedisReconciler{
		Controller{
			Client:       k8sManager.GetClient(),
			Log:          ctrl.Log.WithName("controllers").WithName("Redis"),
			Scheme:       k8sManager.GetScheme(),
			Recorder:     k8sManager.GetEventRecorderFor("redis-reconciler"),
			DefaultToken: aivenToken,
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up OpenSearch reconciler
	err = (&OpenSearchReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("OpenSearch"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("opensearch-reconciler"),
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// set-up KafkaConnector reconciler
	Expect((&KafkaConnectorReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("KafkaConnector"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("kafka-connector-reconciler"),
		},
	}).SetupWithManager(k8sManager)).To(Succeed())

	// set-up Clickhouse reconciler
	Expect((&ClickhouseReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("Clickhouse"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("clickhouse-reconciler"),
		},
	}).SetupWithManager(k8sManager)).To(Succeed())

	// set-up ClickhouseUser reconciler
	Expect((&ClickhouseUserReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("ClickhouseUser"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("clickhouseuser-reconciler"),
		},
	}).SetupWithManager(k8sManager)).To(Succeed())

	// set-up MySQL reconciler
	Expect((&MySQLReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("MySQL"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("mysql-reconciler"),
		},
	}).SetupWithManager(k8sManager)).To(Succeed())

	// set-up Cassandra reconciler
	Expect((&CassandraReconciler{
		Controller{
			Client:   k8sManager.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("Cassandra"),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("cassandra-reconciler"),
		},
	}).SetupWithManager(k8sManager)).To(Succeed())

	go func() {
		Expect(k8sManager.Start(ctrl.SetupSignalHandler())).To(Succeed())
	}()

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

// EnsureDelete deletes the instance and waits for it to be gone or timeout
func ensureDelete(ctx context.Context, instance client.Object) {
	Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())

	res, err := meta.Accessor(instance)
	Expect(err).NotTo(HaveOccurred())

	names := types.NamespacedName{Name: res.GetName(), Namespace: res.GetNamespace()}

	Eventually(func() bool {
		err := k8sClient.Get(ctx, names, instance)

		return apierrors.IsNotFound(err)
	}, time.Minute*20, time.Second*5, "wait for instance to be gone from k8s").Should(BeTrue())
}

func anyPointer[T any](v T) *T {
	return &v
}

// boolPointer converts boolean to *bool
func boolPointer(b bool) *bool {
	return &b
}

// int64Pointer converts int64 to a pointer int64
func int64Pointer(i int64) *int64 {
	return &i
}

// generateRandomString generate a random id
func generateRandomID() string {
	var src = rand.NewSource(time.Now().UnixNano())
	return strconv.FormatInt(src.Int63(), 10)
}
