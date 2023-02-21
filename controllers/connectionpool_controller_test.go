// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

var _ = Describe("ConnectionPool Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		pg          *v1alpha1.PostgreSQL
		user        *v1alpha1.ServiceUser
		db          *v1alpha1.Database
		pool        *v1alpha1.ConnectionPool
		serviceName string
		dbName      string
		userName    string
		poolName    string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-pg-pool-acc-" + generateRandomID()
		dbName = "k8s-db-pool-acc-" + generateRandomID()
		userName = "k8s-user-pool-acc-" + generateRandomID()
		poolName = "k8s-pool-acc-" + generateRandomID()
		pg = pgSpec(serviceName, namespace)
		db = databaseSpec(serviceName, dbName, namespace)
		user = serviceUserSpec(serviceName, userName, namespace)
		pool = connectionPoolSpec(serviceName, dbName, poolName, userName, namespace)

		By("Creating a new PostgreSQL CR instance")
		Expect(k8sClient.Create(ctx, pg)).Should(Succeed())

		By("Creating a new Database CR instance")
		Expect(k8sClient.Create(ctx, db)).Should(Succeed())

		By("Creating a new ServiceUser CR instance")
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())

		By("Creating a new ConnectionPool CR instance")
		Expect(k8sClient.Create(ctx, pool)).Should(Succeed())

		By("by retrieving ConnectionPool instance from k8s")
		Eventually(func() bool {
			lookupKey := types.NamespacedName{Name: poolName, Namespace: namespace}
			createdPool := &v1alpha1.ConnectionPool{}
			err := k8sClient.Get(ctx, lookupKey, createdPool)
			if err == nil {
				return meta.IsStatusConditionTrue(createdPool.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())
	})

	Context("Validating ConnectionPool reconciler behaviour", func() {
		It("should createOrUpdate a new ConnectionPoll instance", func() {
			createdPool := &v1alpha1.ConnectionPool{}
			lookupKey := types.NamespacedName{Name: poolName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdPool)).Should(Succeed())

			By("by checking ConnectionPool secret and status fields")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: poolName, Namespace: namespace}, createdSecret)).Should(Succeed())

			Expect(createdSecret.Data["PGHOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGDATABASE"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGUSER"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGPASSWORD"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGSSLMODE"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["DATABASE_URI"]).NotTo(BeEmpty())
		})
	})

	AfterEach(func() {
		By("Ensures that ConnectionPool instance was deleted")
		ensureDelete(ctx, pool)

		By("Ensures that ServiceUser instance was deleted")
		ensureDelete(ctx, user)

		By("Ensures that Database instance was deleted")
		ensureDelete(ctx, db)

		By("Ensures that PostgreSQL instance was deleted")
		ensureDelete(ctx, pg)
	})
})

var _ = Describe("ConnectionPool Controller re-use incoming user", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		pg          *v1alpha1.PostgreSQL
		db          *v1alpha1.Database
		pool        *v1alpha1.ConnectionPool
		serviceName string
		dbName      string
		poolName    string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-pg-pool-acc-" + generateRandomID()
		dbName = "k8s-db-pool-acc-" + generateRandomID()
		poolName = "k8s-pool-acc-" + generateRandomID()
		pg = pgSpec(serviceName, namespace)
		db = databaseSpec(serviceName, dbName, namespace)
		pool = connectionPoolIncomingUserSpec(serviceName, dbName, poolName, namespace)

		By("Creating a new PostgreSQL CR instance")
		Expect(k8sClient.Create(ctx, pg)).Should(Succeed())

		By("Creating a new Database CR instance")
		Expect(k8sClient.Create(ctx, db)).Should(Succeed())

		By("Creating a new ConnectionPool CR instance")
		Expect(k8sClient.Create(ctx, pool)).Should(Succeed())

		By("by retrieving ConnectionPool instance from k8s")
		Eventually(func() bool {
			lookupKey := types.NamespacedName{Name: poolName, Namespace: namespace}
			createdPool := &v1alpha1.ConnectionPool{}
			err := k8sClient.Get(ctx, lookupKey, createdPool)
			if err == nil {
				return meta.IsStatusConditionTrue(createdPool.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())
	})

	Context("Validating ConnectionPool reconciler behaviour", func() {
		It("should createOrUpdate a new ConnectionPoll instance", func() {
			createdPool := &v1alpha1.ConnectionPool{}
			lookupKey := types.NamespacedName{Name: poolName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdPool)).Should(Succeed())

			By("by checking ConnectionPool secret and status fields")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: poolName, Namespace: namespace}, createdSecret)).Should(Succeed())

			Expect(createdSecret.Data["PGHOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGDATABASE"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGUSER"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGPASSWORD"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PGSSLMODE"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["DATABASE_URI"]).NotTo(BeEmpty())
		})
	})

	AfterEach(func() {
		By("Ensures that ConnectionPool instance was deleted")
		ensureDelete(ctx, pool)

		By("Ensures that Database instance was deleted")
		ensureDelete(ctx, db)

		By("Ensures that PostgreSQL instance was deleted")
		ensureDelete(ctx, pg)
	})
})

func connectionPoolSpec(service, database, pool, user, namespace string) *v1alpha1.ConnectionPool {
	return &v1alpha1.ConnectionPool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "ConnectionPool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pool,
			Namespace: namespace,
		},
		Spec: v1alpha1.ConnectionPoolSpec{
			Project:      os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName:  service,
			DatabaseName: database,
			Username:     user,
			PoolSize:     25,
			PoolMode:     "transaction",
			AuthSecretRef: &v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}

func connectionPoolIncomingUserSpec(service, database, pool, namespace string) *v1alpha1.ConnectionPool {
	return &v1alpha1.ConnectionPool{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "ConnectionPool",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pool,
			Namespace: namespace,
		},
		Spec: v1alpha1.ConnectionPoolSpec{
			Project:      os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName:  service,
			DatabaseName: database,
			PoolSize:     25,
			PoolMode:     "transaction",
			AuthSecretRef: &v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
