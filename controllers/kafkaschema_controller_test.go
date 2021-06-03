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

var _ = Describe("Kafka Schema Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		kafka         *v1alpha1.Kafka
		schema        *v1alpha1.KafkaSchema
		serviceName   string
		schemaSubject string
		ctx           context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-kafka-schema-acc-" + generateRandomID()
		schemaSubject = "k8s-subj1"
		kafka = kafkaSpec(serviceName, namespace)
		schema = kafkaSchemaSpec(serviceName, schemaSubject, namespace)

		By("Creating a new Kafka instance")
		Expect(k8sClient.Create(ctx, kafka)).Should(Succeed())

		By("by waiting Kafka service status to become RUNNING")
		Eventually(func() string {
			kafkaLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
			createdKafka := &v1alpha1.Kafka{}
			err := k8sClient.Get(ctx, kafkaLookupKey, createdKafka)
			if err == nil {
				return createdKafka.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("RUNNING"))

		By("Creating a new KafkaSchema instance")
		Expect(k8sClient.Create(ctx, schema)).Should(Succeed())

		time.Sleep(5 * time.Second)

		By("by retrieving KafkaSchema instance from k8s")
		Eventually(func() bool {
			lookupKey := types.NamespacedName{Name: schemaSubject, Namespace: namespace}
			createdSchema := &v1alpha1.KafkaSchema{}
			err := k8sClient.Get(ctx, lookupKey, createdSchema)

			return err == nil
		}, timeout, interval).Should(BeTrue())
	})

	Context("Validating Kafka ACL reconciler behaviour", func() {
		It("should create a new Kafka Schema", func() {
			createdSchema := &v1alpha1.KafkaSchema{}
			lookupKey := types.NamespacedName{Name: schemaSubject, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdSchema)).Should(Succeed())

			By("by checking that after creation KafkaSchema status fields were properly populated")
			Expect(createdSchema.Status.ServiceName).Should(Equal(serviceName))
			Expect(createdSchema.Status.Project).Should(Equal(os.Getenv("AIVEN_PROJECT_NAME")))
			Expect(createdSchema.Status.CompatibilityLevel).Should(Equal("BACKWARD"))
			Expect(createdSchema.Status.Schema).NotTo(BeEmpty())
			Expect(createdSchema.Status.Version).Should(Equal(1))
		})
	})

	AfterEach(func() {
		By("Ensures that Kafka Schema instance was deleted")
		ensureDelete(ctx, schema)
		By("Ensures that Kafka instance was deleted")
		ensureDelete(ctx, kafka)
	})
})

func kafkaSchemaSpec(service, subjName, namespace string) *v1alpha1.KafkaSchema {
	return &v1alpha1.KafkaSchema{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k8s-operator.aiven.io/v1alpha1",
			Kind:       "KafkaSchema",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      subjName,
			Namespace: namespace,
		},
		Spec: v1alpha1.KafkaSchemaSpec{
			Project:     os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName: service,
			SubjectName: subjName,
			Schema: `{
					"doc": "example",
					"fields": [{
						"default": 5,
						"doc": "my test number",
						"name": "test",
						"namespace": "test",
						"type": "int"
					}],
					"name": "example",
					"namespace": "example",
					"type": "record"
				}`,
			CompatibilityLevel: "BACKWARD",
		},
	}
}
