package controllers

import (
	"context"
	"k8s.io/apimachinery/pkg/api/meta"
	"os"
	"time"

	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

		By("Creating a new KafkaSchema instance")
		Expect(k8sClient.Create(ctx, schema)).Should(Succeed())

		By("by retrieving KafkaSchema instance from k8s")
		Eventually(func() bool {
			lookupKey := types.NamespacedName{Name: schemaSubject, Namespace: namespace}
			createdSchema := &v1alpha1.KafkaSchema{}
			err := k8sClient.Get(ctx, lookupKey, createdSchema)
			if err == nil {
				return meta.IsStatusConditionTrue(createdSchema.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())
	})

	Context("Validating Kafka ACL reconciler behaviour", func() {
		It("should createOrUpdate a new Kafka Schema", func() {
			createdSchema := &v1alpha1.KafkaSchema{}
			lookupKey := types.NamespacedName{Name: schemaSubject, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdSchema)).Should(Succeed())

			By("by checking that after creation KafkaSchema status fields were properly populated")
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
			APIVersion: "aiven.io/v1alpha1",
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
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
