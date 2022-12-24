// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

var _ = Describe("ProjectVPC can't be deleted while it has dependencies", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"
		timeout   = time.Minute * 20
		interval  = time.Second * 10
	)

	Context("Validating Kafka blocks ProjectVPC deleting", func() {
		ctx := context.Background()
		projectName := os.Getenv("AIVEN_PROJECT_NAME")
		serviceName := "k8s-test-kafka-acc-" + generateRandomID()
		projectVPCName := "k8s-test-project-vpc-acc-" + generateRandomID()
		projectVPCObj := projectVPC(projectVPCName, namespace, projectName)
		kafkaObj := kafkaForProjectVPC(serviceName, namespace, projectName, projectVPCName)
		kafkaLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		projectVPLookupCKey := types.NamespacedName{Name: projectVPCName, Namespace: namespace}

		It("Deletes project VPC after dependent kafka deleted", func() {
			// "ProjectVPC and Kafka are created"
			By("Creating a new ProjectVPC CR instance")
			Expect(k8sClient.Create(ctx, projectVPCObj)).Should(Succeed())

			By("Creating a new Kafka CR instance")
			Expect(k8sClient.Create(ctx, kafkaObj)).Should(Succeed())

			// "Kafka is running"
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

			// "Kafka blocks ProjectVPC deletion"
			By("Tries to delete ProjectVPC but fails")
			Expect(k8sClient.Delete(ctx, projectVPCObj)).Should(Succeed())

			By("checking ProjectVPC is ACTIVE even though it was sent to delete")
			createdProjectVPC := &v1alpha1.ProjectVPC{}
			Expect(k8sClient.Get(ctx, projectVPLookupCKey, createdProjectVPC)).Should(Succeed())
			Expect(createdProjectVPC.Status.State).Should(Equal("ACTIVE"))

			// "Deletes Kafka, ProjectVPC deleted later automatically"
			By("Ensures that Kafka instance was deleted")
			ensureDelete(ctx, kafkaObj)

			By("Waits until ProjectVPC is deleted automatically")
			Eventually(func() bool {
				createdProjectVPC := &v1alpha1.ProjectVPC{}
				err := k8sClient.Get(ctx, projectVPLookupCKey, createdProjectVPC)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})
})

func projectVPC(projectVPCName, namespace, projectName string) *v1alpha1.ProjectVPC {
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
			CloudName:   "google-europe-west2",
			NetworkCidr: "10.0.0.0/24",
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}

func kafkaForProjectVPC(serviceName, namespace, projectName, projectVPCName string) *v1alpha1.Kafka {
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
				CloudName: "google-europe-west2",
				Tags:      map[string]string{"key1": "value1"},
				ProjectVPCRef: &v1alpha1.ResourceReference{
					Name:      projectVPCName,
					Namespace: namespace,
				},
			},
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
