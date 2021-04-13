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
		serviceName = "k8s-test-pg-db-acc-" + generateRandomID()
		dbName = "k8s-db-acc-" + generateRandomID()
		pg = pgSpec(serviceName, namespace)
		db = databaseSpec(serviceName, dbName, namespace)

		By("Creating a new PG CR instance")
		Expect(k8sClient.Create(ctx, pg)).Should(Succeed())

		By("Waiting PG service status to become RUNNING")
		Eventually(func() string {
			pgLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
			createdPG := &v1alpha1.PG{}
			err := k8sClient.Get(ctx, pgLookupKey, createdPG)
			if err == nil {
				return createdPG.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("RUNNING"))

		ctx = context.Background()

		By("Creating a new Database CR instance")
		Expect(k8sClient.Create(ctx, db)).Should(Succeed())

		// We'll need to retry getting this newly created instance,
		// given that creation may not immediately happen.
		By("by retrieving Database instance from k8s")
		Eventually(func() bool {
			dbLookupKey := types.NamespacedName{Name: dbName, Namespace: namespace}
			createdDB := &v1alpha1.Database{}
			err := k8sClient.Get(ctx, dbLookupKey, createdDB)

			return err == nil
		}, timeout, interval).Should(BeTrue())
	})

	Context("Validating Database reconciler behaviour", func() {
		It("should create a new Database instance", func() {
			createdDB := &v1alpha1.Database{}
			lookupKey := types.NamespacedName{Name: dbName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdDB)).Should(Succeed())

			// Let's make sure our Database status was properly populated.
			By("by checking that after creation Database status fields were properly populated")
			Expect(createdDB.Status.ServiceName).Should(Equal(serviceName))
			Expect(createdDB.Status.Project).Should(Equal(os.Getenv("AIVEN_PROJECT_NAME")))
			Expect(createdDB.Status.DatabaseName).Should(Equal(dbName))
			Expect(createdDB.Status.LcType).Should(Equal("en_US.UTF-8"))
			Expect(createdDB.Status.LcCollate).Should(Equal("en_US.UTF-8"))
		})
	})

	AfterEach(func() {
		By("Ensures that PG instance was deleted")
		ensureDelete(ctx, pg)
		By("Ensures that Database instance was deleted")
		ensureDelete(ctx, db)
	})
})

func databaseSpec(service, database, namespace string) *v1alpha1.Database {
	return &v1alpha1.Database{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k8s-operator.aiven.io/v1alpha1",
			Kind:       "Database",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      database,
			Namespace: namespace,
		},
		Spec: v1alpha1.DatabaseSpec{
			Project:      os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName:  service,
			DatabaseName: database,
			LcType:       "en_US.UTF-8",
			LcCollate:    "en_US.UTF-8",
		},
	}
}
