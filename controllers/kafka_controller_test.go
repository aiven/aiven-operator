package controllers

import (
	"context"
	"github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"time"
)

var _ = Describe("Kafka Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		kafkaNamespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		kafka       *v1alpha1.Kafka
		serviceName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-kafka-acc-" + generateRandomID()
		kafka = kafkaSpec(serviceName, kafkaNamespace)

		By("Creating a new Kafka CR instance")
		Expect(k8sClient.Create(ctx, kafka)).Should(Succeed())

		kafkaLookupKey := types.NamespacedName{Name: serviceName, Namespace: kafkaNamespace}
		createdKafka := &v1alpha1.Kafka{}
		// We'll need to retry getting this newly created Kafka,
		// given that creation may not immediately happen.
		By("by retrieving Kafka instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, kafkaLookupKey, createdKafka)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting Kafka service status to become RUNNING")
		Eventually(func() string {
			err := k8sClient.Get(ctx, kafkaLookupKey, createdKafka)
			if err == nil {
				return createdKafka.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("RUNNING"))

		By("by checking finalizers")
		Expect(createdKafka.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating Kafka reconciler behaviour", func() {
		It("should create a new Kafka service", func() {
			createdKafka := &v1alpha1.Kafka{}
			kafkaLookupKey := types.NamespacedName{Name: serviceName, Namespace: kafkaNamespace}

			Expect(k8sClient.Get(ctx, kafkaLookupKey, createdKafka)).Should(Succeed())

			// Let's make sure our Kafka status was properly populated.
			By("by checking that after creation Kafka service status fields were properly populated")
			Expect(createdKafka.Status.State).Should(Equal("RUNNING"))
		})
	})

	AfterEach(func() {
		By("Ensures that Kafka instance was deleted")
		ensureDelete(ctx, kafka)
	})
})

func kafkaSpec(serviceName, namespace string) *v1alpha1.Kafka {
	return &v1alpha1.Kafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k8s-operator.aiven.io/v1alpha1",
			Kind:       "Kafka",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.KafkaSpec{
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "business-4",
				CloudName: "google-europe-west1",
			},
			KafkaUserConfig: v1alpha1.KafkaUserConfig{
				KafkaRest:      boolPointer(true),
				KafkaConnect:   boolPointer(true),
				SchemaRegistry: boolPointer(true),
				KafkaVersion:   "2.8",
				Kafka: v1alpha1.KafkaSubKafkaUserConfig{
					GroupMaxSessionTimeoutMs: int64Pointer(70000),
					LogRetentionBytes:        int64Pointer(1000000000),
				},
			},
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
