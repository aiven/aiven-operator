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
	mysqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/mysql"
)

var _ = Describe("MySQL Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		mysql       *v1alpha1.MySQL
		serviceName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-mysql-acc-" + generateRandomID()
		mysql = mysqlSpec(serviceName, namespace)

		By("Creating a new MySQL CR instance")
		Expect(k8sClient.Create(ctx, mysql)).Should(Succeed())

		mysqlLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdMySQL := &v1alpha1.MySQL{}
		// We'll need to retry getting this newly created MySQL,
		// given that creation may not immediately happen.
		By("by retrieving MySQL instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, mysqlLookupKey, createdMySQL)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting MySQL service status to become RUNNING")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, mysqlLookupKey, createdMySQL)
			if err == nil {
				return meta.IsStatusConditionTrue(createdMySQL.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdMySQL.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating MySQL reconciler behaviour", func() {
		It("should createOrUpdate a new MySQL service", func() {
			createdMySQL := &v1alpha1.MySQL{}
			mysqlLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, mysqlLookupKey, createdMySQL)).Should(Succeed())

			By("by checking that after creation of a MySQL service secret is created")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSecret)).Should(Succeed())

			// It is running
			Expect(createdMySQL.Status.State).Should(Equal("RUNNING"))

			// Secretes test
			Expect(createdSecret.Data["MYSQL_HOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["MYSQL_PORT"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["MYSQL_DATABASE"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["MYSQL_USER"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["MYSQL_PASSWORD"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["MYSQL_SSL_MODE"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["MYSQL_URI"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["MYSQL_REPLICA_URI"]).NotTo(BeEmpty())

			// User config test
			Expect(*createdMySQL.Spec.UserConfig.BackupHour).Should(Equal(12))
			Expect(*createdMySQL.Spec.UserConfig.BackupMinute).Should(Equal(42))

			// Ip filters test
			expectedIPFilter := []*mysqluserconfig.IpFilter{
				{
					Network: "10.20.0.0/16",
				},
				{
					Network:     "0.0.0.0",
					Description: anyPointer("whatever"),
				},
			}
			Expect(createdMySQL.Spec.UserConfig.IpFilter).Should(Equal(expectedIPFilter))
		})
	})

	AfterEach(func() {
		By("Ensures that MySQL instance was deleted")
		ensureDelete(ctx, mysql)
	})
})

func mysqlSpec(serviceName, namespace string) *v1alpha1.MySQL {
	return &v1alpha1.MySQL{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "MySQL",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.MySQLSpec{
			DiskSpace: "100Gib",
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "business-4",
				CloudName: "google-europe-west1",
				Tags:      map[string]string{"key1": "value1"},
			},
			UserConfig: &mysqluserconfig.MysqlUserConfig{
				BackupHour:   anyPointer(12),
				BackupMinute: anyPointer(42),
				IpFilter: []*mysqluserconfig.IpFilter{
					{
						Network: "10.20.0.0/16",
					},
					{
						Network:     "0.0.0.0",
						Description: anyPointer("whatever"),
					},
				},
			},
			AuthSecretRef: &v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
