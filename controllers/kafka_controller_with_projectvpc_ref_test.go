// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

var _ = Describe("Kafka Controller with ProjectVPC ref", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"
		timeout   = time.Minute * 10
		interval  = time.Second * 10
	)

	var (
		kafkaObj       *v1alpha1.Kafka
		projectVPCObj  *v1alpha1.ProjectVPC
		projectName    string
		serviceName    string
		projectVPCName string
		ctx            context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		projectName = os.Getenv("AIVEN_PROJECT_NAME")
		serviceName = "k8s-test-kafka-acc-" + generateRandomID()
		projectVPCName = "k8s-test-project-vpc-acc-" + generateRandomID()
		projectVPCObj = projectVPCForKafka(projectVPCName, namespace, projectName)
		kafkaObj = kafkaServiceWithProjectVPC(serviceName, namespace, projectName, projectVPCName)

		By("Creating a new ProjectVPC CR instance")
		Expect(k8sClient.Create(ctx, projectVPCObj)).Should(Succeed())

		By("Creating a new Kafka CR instance")
		Expect(k8sClient.Create(ctx, kafkaObj)).Should(Succeed())

		kafkaLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdKafka := &v1alpha1.Kafka{}
		// We'll need to retry getting this newly created Kafka,
		// given that creation may not immediately happen.
		By("by retrieving Kafka instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, kafkaLookupKey, createdKafka)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting Kafka service status to become RUNNING")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, kafkaLookupKey, createdKafka)
			if err == nil {
				return meta.IsStatusConditionTrue(createdKafka.Status.Conditions, conditionTypeRunning)
			}

			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdKafka.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating Kafka and ProjectVPC", func() {
		It("should createOrUpdate a new Kafka service and ProjectVPC", func() {
			createdProjectVPC := &v1alpha1.ProjectVPC{}
			projectVPCLookupKey := types.NamespacedName{Name: projectVPCName, Namespace: namespace}
			Expect(k8sClient.Get(ctx, projectVPCLookupKey, createdProjectVPC)).Should(Succeed())

			createdKafka := &v1alpha1.Kafka{}
			kafkaLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, kafkaLookupKey, createdKafka)).Should(Succeed())

			By("by checking that after creation of a Kafka service secret is created")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSecret)).Should(Succeed())
			Expect(createdKafka.Status.State).Should(Equal("RUNNING"))
		})
	})

	AfterEach(func() {
		By("Ensures that Kafka instance was deleted")
		ensureDelete(ctx, kafkaObj)

		By("Ensures that ProjectVPC instance was deleted")
		ensureDelete(ctx, projectVPCObj)
	})
})

func projectVPCForKafka(projectVPCName, namespace, projectName string) *v1alpha1.ProjectVPC {
	return &v1alpha1.ProjectVPC{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "ProjectVPC",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectVPCName,
			Namespace: namespace,
		},
		Spec: v1alpha1.ProjectVPCSpec{
			Project:     projectName,
			CloudName:   "google-europe-west1",
			NetworkCidr: "10.0.0.0/24",
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}

func kafkaServiceWithProjectVPC(serviceName, namespace, projectName, projectVPCName string) *v1alpha1.Kafka {
	return &v1alpha1.Kafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Kafka",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.KafkaSpec{
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   projectName,
				Plan:      "startup-2",
				CloudName: "google-europe-west1",
				Tags:      map[string]string{"key1": "value1"},
				ProjectVPCRef: &v1alpha1.ResourceReference{
					Name: projectVPCName,
					// it's expected it goes well with the same namespace
				},
			},
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}

var _ = Describe("Tests Kafka's Validator interface", func() {
	const (
		namespace = "default"
		expectErr = "please set ProjectVPCID or ProjectVPCRef, not both"
	)

	It("create fails because of two ProjectVPCID", func() {
		projectName := os.Getenv("AIVEN_PROJECT_NAME")
		serviceName := "k8s-test-kafka-acc-" + generateRandomID()
		projectVPCName := "some-cool-project-vpc-name"
		kafkaObj := kafkaServiceWithProjectVPC(serviceName, namespace, projectName, projectVPCName)

		// Provides extra id to fail
		kafkaObj.Spec.ProjectVPCID = "lol"

		By("Calling ValidateCreate")
		Expect(kafkaObj.ValidateCreate()).Should(MatchError(expectErr))

		By("Calling ValidateUpdate")
		Expect(kafkaObj.ValidateUpdate(kafkaObj.DeepCopyObject())).Should(MatchError(expectErr))

		By("Calling ValidateDelete")
		Expect(kafkaObj.ValidateDelete()).Should(Succeed())
	})
})
