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
	cassandrauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/cassandra"
)

var _ = Describe("Cassandra Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		cassandra   *v1alpha1.Cassandra
		serviceName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-cassandra-acc-" + generateRandomID()
		cassandra = cassandraSpec(serviceName, namespace)

		By("Creating a new Cassandra CR instance")
		Expect(k8sClient.Create(ctx, cassandra)).Should(Succeed())

		cassandraLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdCassandra := &v1alpha1.Cassandra{}
		// We'll need to retry getting this newly created Cassandra,
		// given that creation may not immediately happen.
		By("by retrieving Cassandra instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, cassandraLookupKey, createdCassandra)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting Cassandra service status to become RUNNING")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, cassandraLookupKey, createdCassandra)
			if err == nil {
				return meta.IsStatusConditionTrue(createdCassandra.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdCassandra.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating Cassandra reconciler behaviour", func() {
		It("should createOrUpdate a new Cassandra service", func() {
			createdCassandra := &v1alpha1.Cassandra{}
			cassandraLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, cassandraLookupKey, createdCassandra)).Should(Succeed())

			By("by checking that after creation of a Cassandra service secret is created")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSecret)).Should(Succeed())

			// It is running
			Expect(createdCassandra.Status.State).Should(Equal("RUNNING"))

			// Secretes test
			Expect(createdSecret.Data["CASSANDRA_HOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["CASSANDRA_PORT"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["CASSANDRA_USER"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["CASSANDRA_PASSWORD"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["CASSANDRA_URI"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["CASSANDRA_HOSTS"]).NotTo(BeEmpty())

			// User config test
			Expect(*createdCassandra.Spec.UserConfig.MigrateSstableloader).Should(Equal(true))
			Expect(*createdCassandra.Spec.UserConfig.PublicAccess.Prometheus).Should(Equal(true))

			// Ip filters test
			expectedIPFilter := []*cassandrauserconfig.IpFilter{
				{
					Network: "10.20.0.0/16",
				},
				{
					Network:     "0.0.0.0",
					Description: anyPointer("whatever"),
				},
			}
			Expect(createdCassandra.Spec.UserConfig.IpFilter).Should(Equal(expectedIPFilter))
		})
	})

	AfterEach(func() {
		By("Ensures that Cassandra instance was deleted")
		ensureDelete(ctx, cassandra)
	})
})

func cassandraSpec(serviceName, namespace string) *v1alpha1.Cassandra {
	return &v1alpha1.Cassandra{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Cassandra",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.CassandraSpec{
			DiskSpace: "450Gib",
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "startup-4",
				CloudName: "google-europe-west1",
				Tags:      map[string]string{"key1": "value1"},
			},
			UserConfig: &cassandrauserconfig.CassandraUserConfig{
				MigrateSstableloader: anyPointer(true),
				PublicAccess: &cassandrauserconfig.PublicAccess{
					Prometheus: anyPointer(true),
				},
				IpFilter: []*cassandrauserconfig.IpFilter{
					{
						Network: "10.20.0.0/16",
					},
					{
						Network:     "0.0.0.0",
						Description: anyPointer("whatever"),
					},
				},
			},
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
