package controllers

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"os"
	"time"

	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ConnectionPool Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		db          *v1alpha1.Database
		pg          *v1alpha1.PG
		user        *v1alpha1.ServiceUser
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

		By("Creating a new PG CR instance")
		Expect(k8sClient.Create(ctx, pg)).Should(Succeed())

		By("Creating a new Database CR instance")
		Expect(k8sClient.Create(ctx, db)).Should(Succeed())

		By("Creating a new ServiceUser CR instance")
		Expect(k8sClient.Create(ctx, user)).Should(Succeed())

		By("Creating a new ConnectionPool CR instance")
		Expect(k8sClient.Create(ctx, pool)).Should(Succeed())

		time.Sleep(10 * time.Second)

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
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: poolName, Namespace: namespace}, createdSecret))

			Expect(createdSecret.StringData["PGHOST"]).NotTo(BeEmpty())
			Expect(createdSecret.StringData["PGDATABASE"]).NotTo(BeEmpty())
			Expect(createdSecret.StringData["PGUSER"]).NotTo(BeEmpty())
			Expect(createdSecret.StringData["PGPASSWORD"]).NotTo(BeEmpty())
			Expect(createdSecret.StringData["PGSSLMODE"]).NotTo(BeEmpty())
			Expect(createdSecret.StringData["DATABASE_URI"]).NotTo(BeEmpty())
		})
	})

	AfterEach(func() {
		By("Ensures that PG instance was deleted")
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
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
