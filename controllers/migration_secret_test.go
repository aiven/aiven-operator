package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	mysqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/mysql"
	pguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/pg"
)

var adapters = []adapterFactory{
	{"pg", func(ns, secret string, k8s *fake.ClientBuilder) migrationAdapter {
		return newPgAdapter(ns, secret, k8s)
	}},
	{"mysql", func(ns, secret string, k8s *fake.ClientBuilder) migrationAdapter {
		return newTestMySQLAdapter(ns, secret, k8s)
	}},
}

func TestPgAdapter_FullFields(t *testing.T) {
	t.Parallel()

	s := secretInNs("creds", "default", map[string][]byte{
		"host":         []byte("source-db.example.com"),
		"port":         []byte("5432"),
		"password":     []byte("super-secret"),
		"dbname":       []byte("mydb"),
		"username":     []byte("postgres"),
		"ssl":          []byte("true"),
		"method":       []byte("dump"),
		"ignore_dbs":   []byte("template0,template1"),
		"ignore_roles": []byte("rdsadmin"),
	})
	adapter := newPgAdapter("default", "creds", newFakeClient(s))
	result, err := resolveForTest(context.Background(), adapter)
	require.NoError(t, err)

	cfg := result.(*pguserconfig.PgUserConfig)
	m := cfg.Migration
	require.NotNil(t, m)
	assert.Equal(t, "source-db.example.com", m.Host)
	assert.Equal(t, 5432, m.Port)
	assert.Equal(t, "super-secret", *m.Password)
	assert.Equal(t, "mydb", *m.Dbname)
	assert.Equal(t, "postgres", *m.Username)
	assert.True(t, *m.Ssl)
	assert.Equal(t, "dump", *m.Method)
	assert.Equal(t, "template0,template1", *m.IgnoreDbs)
	assert.Equal(t, "rdsadmin", *m.IgnoreRoles)

	// Original spec must not be mutated
	assert.Nil(t, adapter.Spec.UserConfig)
}

func TestMySQLAdapter_FullFields(t *testing.T) {
	t.Parallel()

	s := secretInNs("creds", "default", map[string][]byte{
		"host":                    []byte("mysql-source.example.com"),
		"port":                    []byte("3306"),
		"password":                []byte("mysql-secret"),
		"dbname":                  []byte("mydb"),
		"username":                []byte("root"),
		"ssl":                     []byte("false"),
		"method":                  []byte("dump"),
		"ignore_dbs":              []byte("sys,information_schema"),
		"ignore_roles":            []byte("rdsadmin"),
		"dump_tool":               []byte("mydumper"),
		"reestablish_replication": []byte("true"),
	})
	adapter := newTestMySQLAdapter("default", "creds", newFakeClient(s))
	result, err := resolveForTest(context.Background(), adapter)
	require.NoError(t, err)

	cfg := result.(*mysqluserconfig.MysqlUserConfig)
	m := cfg.Migration
	require.NotNil(t, m)
	assert.Equal(t, "mysql-source.example.com", m.Host)
	assert.Equal(t, 3306, m.Port)
	assert.Equal(t, "mysql-secret", *m.Password)
	assert.Equal(t, "mydb", *m.Dbname)
	assert.Equal(t, "root", *m.Username)
	assert.False(t, *m.Ssl)
	assert.Equal(t, "dump", *m.Method)
	assert.Equal(t, "sys,information_schema", *m.IgnoreDbs)
	assert.Equal(t, "rdsadmin", *m.IgnoreRoles)
	assert.Equal(t, "mydumper", *m.DumpTool)
	assert.True(t, *m.ReestablishReplication)

	// Original spec must not be mutated
	assert.Nil(t, adapter.Spec.UserConfig)
}

func TestAdapters_StringFieldsPreservedAsStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		username string
	}{
		{"numeric values", "12345", "0"},
		{"bool-looking values", "true", "false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := secretInNs("creds", "default", map[string][]byte{
				"host":     []byte("db.example.com"),
				"port":     []byte("5432"),
				"password": []byte(tt.password),
				"username": []byte(tt.username),
			})
			adapter := newPgAdapter("default", "creds", newFakeClient(s))
			result, err := resolveForTest(context.Background(), adapter)
			require.NoError(t, err)

			cfg := result.(*pguserconfig.PgUserConfig)
			m := cfg.Migration
			assert.Equal(t, tt.password, *m.Password)
			assert.Equal(t, tt.username, *m.Username)
		})
	}
}

func TestAdapters_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		ns        string
		secret    string
		data      map[string][]byte
		errSubstr string
	}{
		{"missing secret", "default", "nonexistent", nil, "nonexistent"},
		{"missing host", "default", "creds", map[string][]byte{"port": []byte("5432")}, "host"},
		{"missing port", "default", "creds", map[string][]byte{"host": []byte("db.example.com")}, "port"},
		{"missing both", "default", "creds", map[string][]byte{"password": []byte("secret")}, "port"},
		{"invalid port", "default", "creds", map[string][]byte{"host": []byte("db.example.com"), "port": []byte("not-a-number")}, "port"},
		{"invalid ssl", "default", "creds", map[string][]byte{"host": []byte("db.example.com"), "port": []byte("5432"), "ssl": []byte("not-a-bool")}, "ssl"},
	}
	for _, af := range adapters {
		for _, tt := range cases {
			t.Run(af.name+"/"+tt.name, func(t *testing.T) {
				var k *fake.ClientBuilder
				if tt.data != nil {
					k = newFakeClient(secretInNs(tt.secret, tt.ns, tt.data))
				} else {
					k = newFakeClient()
				}
				_, err := resolveForTest(context.Background(), af.newFunc(tt.ns, tt.secret, k))
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			})
		}
	}
}

func TestMySQLAdapter_InvalidReestablishReplication(t *testing.T) {
	t.Parallel()

	s := secretInNs("creds", "default", map[string][]byte{
		"host":                    []byte("mysql-source.example.com"),
		"port":                    []byte("3306"),
		"reestablish_replication": []byte("nope"),
	})
	adapter := newTestMySQLAdapter("default", "creds", newFakeClient(s))

	_, err := resolveForTest(context.Background(), adapter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reestablish_replication")
}

func TestPgAdapter_SecretOverridesInlineConfig(t *testing.T) {
	t.Parallel()

	s := secretInNs("creds", "default", map[string][]byte{
		"host": []byte("secret-host.example.com"),
		"port": []byte("5432"),
	})
	pg := &v1alpha1.PostgreSQL{}
	pg.Namespace = "default"
	pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: "creds"}
	pg.Spec.UserConfig = &pguserconfig.PgUserConfig{
		Migration: &pguserconfig.Migration{
			Host:     "inline-host.example.com",
			Port:     9999,
			Password: new("inline-password"),
		},
	}
	adapter := &postgreSQLAdapter{PostgreSQL: pg, k8s: newFakeClient(s).Build()}
	result, err := resolveForTest(context.Background(), adapter)
	require.NoError(t, err)

	cfg := result.(*pguserconfig.PgUserConfig)
	m := cfg.Migration
	assert.Equal(t, "secret-host.example.com", m.Host)
	assert.Equal(t, 5432, m.Port)
	assert.Nil(t, m.Password)

	// Original spec must not be mutated
	origM := pg.Spec.UserConfig.Migration
	require.NotNil(t, origM)
	assert.Equal(t, "inline-host.example.com", origM.Host)
	assert.Equal(t, 9999, origM.Port)
	assert.Equal(t, "inline-password", *origM.Password)
}

func TestPgAdapter_NilRef(t *testing.T) {
	t.Parallel()

	pg := &v1alpha1.PostgreSQL{}
	adapter := &postgreSQLAdapter{PostgreSQL: pg, k8s: nil}

	result, err := resolveForTest(context.Background(), adapter)
	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Nil(t, pg.Spec.UserConfig)
}

func newFakeClient(objects ...corev1.Secret) *fake.ClientBuilder {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	b := fake.NewClientBuilder().WithScheme(scheme)
	for i := range objects {
		b = b.WithObjects(&objects[i])
	}
	return b
}

func secretInNs(name, ns string, data map[string][]byte) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Data:       data,
	}
}

func newPgAdapter(ns string, secretName string, k8s *fake.ClientBuilder) *postgreSQLAdapter {
	pg := &v1alpha1.PostgreSQL{}
	pg.Namespace = ns
	pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: secretName}
	return &postgreSQLAdapter{PostgreSQL: pg, k8s: k8s.Build()}
}

func newTestMySQLAdapter(ns string, secretName string, k8s *fake.ClientBuilder) *mySQLAdapter {
	mysql := &v1alpha1.MySQL{}
	mysql.Namespace = ns
	mysql.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: secretName}
	return &mySQLAdapter{MySQL: mysql, k8s: k8s.Build()}
}

// migrationAdapter abstracts over PG and MySQL adapters for shared tests.
type migrationAdapter interface {
	migrationSecretProvider
	k8sClient() client.Client
	namespace() string
}

func (a *postgreSQLAdapter) k8sClient() client.Client { return a.k8s }
func (a *postgreSQLAdapter) namespace() string        { return a.GetNamespace() }
func (a *mySQLAdapter) k8sClient() client.Client      { return a.k8s }
func (a *mySQLAdapter) namespace() string             { return a.GetNamespace() }

// resolveForTest reproduces the handler's two-step resolution (read Secret → build config)
// so tests can exercise the adapter mapping without duplicating handler wiring.
func resolveForTest(ctx context.Context, a migrationAdapter) (any, error) {
	ref := a.getMigrationSecretSource()
	if ref == nil {
		return a.buildUserConfigWithMigration(nil)
	}
	data, err := readMigrationSecret(ctx, a.k8sClient(), a.namespace(), ref)
	if err != nil {
		return nil, err
	}
	return a.buildUserConfigWithMigration(data)
}

type adapterFactory struct {
	name    string
	newFunc func(ns, secretName string, k8s *fake.ClientBuilder) migrationAdapter
}

func markMigrationDone(pg *v1alpha1.PostgreSQL) {
	pg.Status.Conditions = append(pg.Status.Conditions, metav1.Condition{
		Type:    v1alpha1.ConditionTypeMigrationComplete,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.MigrationReasonDone,
		Message: "done",
	})
}

func markMigrationInProgress(pg *v1alpha1.PostgreSQL) {
	pg.Status.Conditions = append(pg.Status.Conditions, metav1.Condition{
		Type:    v1alpha1.ConditionTypeMigrationComplete,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.MigrationReasonInProgress,
		Message: "syncing",
	})
}

func secretExists(t *testing.T, k8s client.Client, name, ns string) bool {
	t.Helper()
	err := k8s.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ns}, &corev1.Secret{})
	if err == nil {
		return true
	}
	if apierrors.IsNotFound(err) {
		return false
	}
	t.Fatalf("unexpected error reading secret: %v", err)
	return false
}

func TestMaybeDeleteMigrationSecret(t *testing.T) {
	t.Parallel()

	newPgAdapterWithClient := func(k8s client.Client, deleteAfter bool) (*v1alpha1.PostgreSQL, *postgreSQLAdapter) {
		pg := &v1alpha1.PostgreSQL{}
		pg.Namespace = "default"
		pg.Name = "pg-1"
		pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{
			Name:                 "creds",
			DeleteAfterMigration: deleteAfter,
		}
		return pg, &postgreSQLAdapter{PostgreSQL: pg, k8s: k8s}
	}

	buildClient := func(withSecret bool) client.Client {
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		b := fake.NewClientBuilder().WithScheme(scheme)
		if withSecret {
			s := secretInNs("creds", "default", map[string][]byte{"host": []byte("x")})
			b = b.WithObjects(&s)
		}
		return b.Build()
	}

	h := &genericServiceHandler{log: logr.Discard()}

	t.Run("Deletes secret when flag true and migration complete", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(true)
		pg, adapter := newPgAdapterWithClient(k8s, true)
		markMigrationDone(pg)

		require.NoError(t, h.maybeDeleteMigrationSecret(context.Background(), adapter, adapter))
		assert.False(t, secretExists(t, k8s, "creds", "default"))
	})

	t.Run("Does not delete when flag is false", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(true)
		pg, adapter := newPgAdapterWithClient(k8s, false)
		markMigrationDone(pg)

		require.NoError(t, h.maybeDeleteMigrationSecret(context.Background(), adapter, adapter))
		assert.True(t, secretExists(t, k8s, "creds", "default"))
	})

	t.Run("Does not delete when condition absent", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(true)
		_, adapter := newPgAdapterWithClient(k8s, true)

		require.NoError(t, h.maybeDeleteMigrationSecret(context.Background(), adapter, adapter))
		assert.True(t, secretExists(t, k8s, "creds", "default"))
	})

	t.Run("Does not delete when migration is still in progress", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(true)
		pg, adapter := newPgAdapterWithClient(k8s, true)
		markMigrationInProgress(pg)

		require.NoError(t, h.maybeDeleteMigrationSecret(context.Background(), adapter, adapter))
		assert.True(t, secretExists(t, k8s, "creds", "default"))
	})

	t.Run("Idempotent when secret already gone", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(false)
		pg, adapter := newPgAdapterWithClient(k8s, true)
		markMigrationDone(pg)

		require.NoError(t, h.maybeDeleteMigrationSecret(context.Background(), adapter, adapter))
	})

	t.Run("No-op when MigrationSecretSource is nil", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(false)
		pg := &v1alpha1.PostgreSQL{}
		pg.Namespace = "default"
		adapter := &postgreSQLAdapter{PostgreSQL: pg, k8s: k8s}

		require.NoError(t, h.maybeDeleteMigrationSecret(context.Background(), adapter, adapter))
	})
}
