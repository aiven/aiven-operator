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

var _ = Describe("Database Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		db          *v1alpha1.Database = nil
		pg          *v1alpha1.PG
		serviceName string
		dbName      string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-pg-db-acc-" + generateRandomID()
		dbName = "k8s-db-acc-" + generateRandomID()
		pg = pgSpec(serviceName, namespace)
		db = databaseSpec(serviceName, dbName, namespace)

		By("Creating a new PG CR instance")
		Expect(k8sClient.Create(ctx, pg)).Should(Succeed())

		By("Creating a new Database CR instance")
		Expect(k8sClient.Create(ctx, db)).Should(Succeed())

		// We'll need to retry getting this newly created instance,
		// given that creation may not immediately happen.
		By("by retrieving Database instance from k8s")
		Eventually(func() bool {
			dbLookupKey := types.NamespacedName{Name: dbName, Namespace: namespace}
			createdDB := &v1alpha1.Database{}
			err := k8sClient.Get(ctx, dbLookupKey, createdDB)
			if err == nil {
				return meta.IsStatusConditionTrue(createdDB.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())
	})

	Context("Validating Database reconciler behaviour", func() {
		It("should createOrUpdate a new Database instance", func() {
			createdDB := &v1alpha1.Database{}
			lookupKey := types.NamespacedName{Name: dbName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdDB)).Should(Succeed())

			By("by checking that after Database was created")
			Expect(meta.IsStatusConditionTrue(createdDB.Status.Conditions, conditionTypeRunning)).Should(BeTrue())
		})
	})

	AfterEach(func() {
		By("Ensures that PG instance was deleted")
		ensureDelete(ctx, pg)
	})
})

func databaseSpec(service, database, namespace string) *v1alpha1.Database {
	return &v1alpha1.Database{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Database",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      database,
			Namespace: namespace,
		},
		Spec: v1alpha1.DatabaseSpec{
			Project:     os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName: service,
			LcType:      "en_US.UTF-8",
			LcCollate:   "en_US.UTF-8",
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
