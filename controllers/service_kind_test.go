// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"maps"
	"slices"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const kindTestProject = "test-project"

// kindWant is the kind-specific behavior a service kind must keep.
type kindWant struct {
	serviceType serviceType
	secretKeys  []string // excluding CA keys; nil: no secret
	caCertKeys  []string // keys set to the project CA certificate
	// powerOffBackupCheck: powered: false waits for a backup.
	powerOffBackupCheck bool
	beforeUpdate        bool
	migration           bool
	// driftFields: compared field names; nil means the default set.
	driftFields []string
}

// defaultDriftFieldNames is the compared field set every kind starts from.
var defaultDriftFieldNames = []string{
	"plan", "cloudName", "projectVpcId", "maintenanceWindowDow", "maintenanceWindowTime",
	"terminationProtection", "powered", "disk_space", "tags", "technicalEmails",
}

func fieldNames(fields []driftField) []string {
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.name())
	}
	return names
}

func TestServiceKinds(t *testing.T) {
	t.Parallel()

	accessCert, accessKey, restURI, schemaURI, replicaURI := "cert", "key", "rest", "schema", "replica"
	svcOut := &service.ServiceGetOut{
		State:      service.ServiceStateTypeRunning,
		ServiceUri: "uri",
		ServiceUriParams: map[string]string{
			"host": "h", "port": "1", "user": "u", "password": "p", "dbname": "d",
			"sslmode": "require", "ssl-mode": "REQUIRED", "ssl": "true",
		},
		Users: []service.UserOut{{Username: "u", Password: "p"}},
		ConnectionInfo: &service.ConnectionInfoOut{
			KafkaAccessCert:   &accessCert,
			KafkaAccessKey:    &accessKey,
			KafkaRestUri:      &restURI,
			SchemaRegistryUri: &schemaURI,
			MysqlReplicaUri:   &replicaURI,
			Flink:             []string{"f1"},
			Grafana:           []string{"g1"},
		},
	}

	common := func() v1alpha1.ServiceCommonSpec {
		return v1alpha1.ServiceCommonSpec{
			BaseServiceFields: v1alpha1.BaseServiceFields{
				ProjectDependant: v1alpha1.ProjectDependant{ProjectField: v1alpha1.ProjectField{Project: kindTestProject}},
				Powered:          new(false),
			},
			SecretFields: v1alpha1.SecretFields{ConnInfoSecretTarget: v1alpha1.ConnInfoSecretTarget{Prefix: "P_"}},
		}
	}
	om := metav1.ObjectMeta{Name: "svc", Namespace: "default"}

	t.Run("clickhouse", func(t *testing.T) {
		t.Parallel()
		assertServiceKind(t, clickhouseKind, &v1alpha1.Clickhouse{ObjectMeta: om, Spec: v1alpha1.ClickhouseSpec{ServiceCommonSpec: common()}}, svcOut, kindWant{
			serviceType:         serviceTypeClickhouse,
			secretKeys:          []string{"P_HOST", "P_PASSWORD", "P_PORT", "P_USER", "HOST", "PASSWORD", "PORT", "USER"},
			powerOffBackupCheck: true,
		})
	})

	t.Run("flink", func(t *testing.T) {
		t.Parallel()
		assertServiceKind(t, flinkKind, &v1alpha1.Flink{ObjectMeta: om, Spec: v1alpha1.FlinkSpec{ServiceCommonSpec: common()}}, svcOut, kindWant{
			serviceType:         serviceTypeFlink,
			secretKeys:          []string{"P_HOST", "P_USER", "P_PASSWORD", "P_URI", "P_HOSTS"},
			powerOffBackupCheck: true,
		})
	})

	t.Run("grafana", func(t *testing.T) {
		t.Parallel()
		assertServiceKind(t, grafanaKind, &v1alpha1.Grafana{ObjectMeta: om, Spec: v1alpha1.GrafanaSpec{ServiceCommonSpec: common()}}, svcOut, kindWant{
			serviceType:         serviceTypeGrafana,
			secretKeys:          []string{"P_HOST", "P_PORT", "P_USER", "P_PASSWORD", "P_URI", "P_HOSTS"},
			powerOffBackupCheck: true,
		})
	})

	t.Run("kafka", func(t *testing.T) {
		t.Parallel()
		assertServiceKind(t, kafkaKind, &v1alpha1.Kafka{ObjectMeta: om, Spec: v1alpha1.KafkaSpec{ServiceCommonSpec: common()}}, svcOut, kindWant{
			serviceType: serviceTypeKafka,
			secretKeys: []string{
				"P_HOST", "P_PORT", "P_PASSWORD", "P_USERNAME", "P_ACCESS_CERT", "P_ACCESS_KEY", "P_REST_URI", "P_SCHEMA_REGISTRY_URI",
				"HOST", "PORT", "PASSWORD", "USERNAME", "ACCESS_CERT", "ACCESS_KEY",
			},
			caCertKeys:          []string{"P_CA_CERT", "CA_CERT"},
			powerOffBackupCheck: true,
		})
	})

	t.Run("kafkaconnect", func(t *testing.T) {
		t.Parallel()
		kc := &v1alpha1.KafkaConnect{ObjectMeta: om, Spec: v1alpha1.KafkaConnectSpec{BaseServiceFields: common().BaseServiceFields}}
		assertServiceKind(t, kafkaConnectKind, kc, svcOut, kindWant{
			serviceType: serviceTypeKafkaConnect,
		})
	})

	t.Run("mysql", func(t *testing.T) {
		t.Parallel()
		assertServiceKind(t, mySQLKind, &v1alpha1.MySQL{ObjectMeta: om, Spec: v1alpha1.MySQLSpec{ServiceCommonSpec: common()}}, svcOut, kindWant{
			serviceType:         serviceTypeMySQL,
			secretKeys:          []string{"P_HOST", "P_PORT", "P_DATABASE", "P_USER", "P_PASSWORD", "P_SSL_MODE", "P_URI", "P_REPLICA_URI"},
			caCertKeys:          []string{"P_CA_CERT"},
			powerOffBackupCheck: true,
			migration:           true,
		})
	})

	t.Run("opensearch", func(t *testing.T) {
		t.Parallel()
		assertServiceKind(t, openSearchKind, &v1alpha1.OpenSearch{ObjectMeta: om, Spec: v1alpha1.OpenSearchSpec{ServiceCommonSpec: common()}}, svcOut, kindWant{
			serviceType:         serviceTypeOpenSearch,
			secretKeys:          []string{"P_URI", "P_HOST", "P_PASSWORD", "P_PORT", "P_USER", "HOST", "PASSWORD", "PORT", "USER"},
			powerOffBackupCheck: true,
		})
	})

	t.Run("postgresql", func(t *testing.T) {
		t.Parallel()
		assertServiceKind(t, postgreSQLKind, &v1alpha1.PostgreSQL{ObjectMeta: om, Spec: v1alpha1.PostgreSQLSpec{ServiceCommonSpec: common()}}, svcOut, kindWant{
			serviceType: serviceTypePostgreSQL,
			secretKeys: []string{
				"P_HOST", "P_PORT", "P_DATABASE", "P_USER", "P_PASSWORD", "P_SSLMODE", "P_DATABASE_URI",
				"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGSSLMODE", "DATABASE_URI",
			},
			caCertKeys:          []string{"P_CA_CERT"},
			powerOffBackupCheck: true,
			beforeUpdate:        true,
			migration:           true,
		})
	})

	t.Run("valkey", func(t *testing.T) {
		t.Parallel()
		assertServiceKind(t, valkeyKind, &v1alpha1.Valkey{ObjectMeta: om, Spec: v1alpha1.ValkeySpec{ServiceCommonSpec: common()}}, svcOut, kindWant{
			serviceType:         serviceTypeValkey,
			secretKeys:          []string{"P_HOST", "P_PASSWORD", "P_PORT", "P_SSL", "P_USER"},
			powerOffBackupCheck: true,
		})
	})

	t.Run("withDriftFields replaces the table of that kind only", func(t *testing.T) {
		t.Parallel()
		k := newServiceKind[*v1alpha1.Grafana](serviceTypeGrafana, nil, withDriftFields[*v1alpha1.Grafana](planField{}, tagsField{}))
		require.Equal(t, []string{"plan", "tags"}, fieldNames(k.driftFields))
		require.Equal(t, defaultDriftFieldNames, fieldNames(grafanaKind.driftFields))
	})
}

// assertServiceKind checks kind against want. obj must be powered off with no read replica.
func assertServiceKind[T v1alpha1.ServiceObject](t *testing.T, kind serviceKind[T], obj T, svcOut *service.ServiceGetOut, want kindWant) {
	t.Helper()

	require.Equal(t, want.serviceType, kind.serviceType)
	require.Equal(t, want.beforeUpdate, kind.beforeUpdate != nil, "beforeUpdate")
	require.Equal(t, want.migration, kind.migration != nil, "migration")
	if want.driftFields == nil {
		want.driftFields = defaultDriftFieldNames
	}
	require.Equal(t, want.driftFields, fieldNames(kind.driftFields), "driftFields")

	avn := avngen.NewMockClient(t)
	if len(want.caCertKeys) > 0 {
		avn.EXPECT().ProjectKmsGetCA(mock.Anything, kindTestProject).Return("ca-cert", nil).Once()
	}
	c := newTestController(avn, kind, nil)
	details, err := c.buildSecretDetails(t.Context(), obj, svcOut)
	require.NoError(t, err)

	if want.secretKeys == nil {
		require.Nil(t, details)
	} else {
		wantKeys := slices.Concat(want.secretKeys, want.caCertKeys)
		require.ElementsMatch(t, wantKeys, slices.Collect(maps.Keys(details)))
		for _, k := range want.caCertKeys {
			require.Equal(t, "ca-cert", details[k], k)
		}
	}

	if want.powerOffBackupCheck {
		avn.EXPECT().
			ServiceBackupsGet(mock.Anything, kindTestProject, obj.GetName()).
			Return(&service.ServiceBackupsGetOut{}, nil).Once()
		require.ErrorIs(t, kind.checkPreconditions(t.Context(), avn, obj), errPreconditionExternal)
	} else {
		// The mock fails the test on an unexpected ServiceBackupsGet.
		require.NoError(t, kind.checkPreconditions(t.Context(), avn, obj))
	}
}
