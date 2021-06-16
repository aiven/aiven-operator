package controllers

import (
	"context"
	"os"
	"time"

	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Service Integration Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		kafka            *v1alpha1.Kafka
		kafkaConnect     *v1alpha1.KafkaConnect
		si               *v1alpha1.ServiceIntegration
		kafkaName        string
		kafkaConnectName string
		siName           string
		ctx              context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		kafkaName = "k8s-test-kafka-si-acc-" + generateRandomID()
		kafkaConnectName = "k8s-test-kafka-connect-si-acc-" + generateRandomID()
		siName = "si1"
		kafka = kafkaSpec(kafkaName, namespace)
		kafkaConnect = kafkaConnectSpec(kafkaConnectName, namespace)
		si = serviceIntegrationSpec(siName, kafkaName, kafkaConnectName, namespace)

		By("Creating a new Kafka CR instance")
		Expect(k8sClient.Create(ctx, kafka)).Should(Succeed())

		By("Creating a new KafkaConnect CR instance")
		Expect(k8sClient.Create(ctx, kafkaConnect)).Should(Succeed())

		By("by waiting Kafka service status to become RUNNING")
		Eventually(func() string {
			lookupKey := types.NamespacedName{Name: kafkaName, Namespace: namespace}
			createdKafka := &v1alpha1.Kafka{}
			err := k8sClient.Get(ctx, lookupKey, createdKafka)
			if err == nil {
				return createdKafka.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("RUNNING"))

		By("by waiting KafkaConnect service status to become RUNNING")
		Eventually(func() string {
			lookupKey := types.NamespacedName{Name: kafkaConnectName, Namespace: namespace}
			createdKafkaConnect := &v1alpha1.KafkaConnect{}
			err := k8sClient.Get(ctx, lookupKey, createdKafkaConnect)
			if err == nil {
				return createdKafkaConnect.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("RUNNING"))

		By("Creating a new ServiceIntegration CR instance")
		Expect(k8sClient.Create(ctx, si)).Should(Succeed())

		time.Sleep(10 * time.Second)

		// We'll need to retry getting this newly created instance,
		// given that creation may not immediately happen.
		By("by retrieving ServiceIntegration instance from k8s")
		Eventually(func() bool {
			lookupKey := types.NamespacedName{Name: siName, Namespace: namespace}
			createdSI := &v1alpha1.ServiceIntegration{}
			err := k8sClient.Get(ctx, lookupKey, createdSI)

			return err == nil
		}, timeout, interval).Should(BeTrue())
	})

	Context("Validating ServiceIntegration reconciler behaviour", func() {
		It("should create a new ServiceIntegration service", func() {
			si := &v1alpha1.ServiceIntegration{}
			lookupKey := types.NamespacedName{Name: siName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, si)).Should(Succeed())

			// Let's make sure our Kafka status was properly populated.
			By("by checking that after creation Kafka service status fields were properly populated")
			Expect(si.Status.IntegrationType).Should(Equal("kafka_connect"))
			Expect(si.Status.SourceServiceName).Should(Equal(kafkaName))
			Expect(si.Status.DestinationServiceName).Should(Equal(kafkaConnectName))
			Expect(si.Status.ID).ShouldNot(BeEmpty())

			By("by checking finalizers")
			Expect(si.GetFinalizers()).ToNot(BeEmpty())
		})
	})

	AfterEach(func() {
		By("Ensures that ServiceIntegration instance was deleted")
		ensureDelete(ctx, si)
		By("Ensures that Kafka instance was deleted")
		ensureDelete(ctx, kafka)
		By("Ensures that KafkaConnect instance was deleted")
		ensureDelete(ctx, kafkaConnect)
	})
})

func serviceIntegrationSpec(siName, source, destination, namespace string) *v1alpha1.ServiceIntegration {
	return &v1alpha1.ServiceIntegration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "ServiceIntegration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      siName,
			Namespace: namespace,
		},
		Spec: v1alpha1.ServiceIntegrationSpec{
			Project:                os.Getenv("AIVEN_PROJECT_NAME"),
			IntegrationType:        "kafka_connect",
			SourceServiceName:      source,
			DestinationServiceName: destination,
			KafkaConnectUserConfig: v1alpha1.ServiceIntegrationKafkaConnectUserConfig{
				KafkaConnect: v1alpha1.ServiceIntegrationKafkaConnect{
					GroupID:            "connect",
					OffsetStorageTopic: "__connect_status",
					StatusStorageTopic: "__connect_offsets",
				},
			},
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
