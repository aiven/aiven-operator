package controllers

import (
	"context"
	"net/http"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	mysqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/mysql"
	pguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/pg"
)

var migrationKinds = []migrationKindCase{
	{"pg", func(ctx context.Context, ns, secret string, k8s *fake.ClientBuilder) (any, error) {
		return resolvePGForTest(ctx, newTestPG(ns, secret), k8s.Build())
	}},
	{"mysql", func(ctx context.Context, ns, secret string, k8s *fake.ClientBuilder) (any, error) {
		return resolveMySQLForTest(ctx, newTestMySQL(ns, secret), k8s.Build())
	}},
}

func TestPostgreSQLMigrationUserConfig_FullFields(t *testing.T) {
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
	pg := newTestPG("default", "creds")
	result, err := resolvePGForTest(context.Background(), pg, newFakeClient(s).Build())
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
	assert.Nil(t, pg.Spec.UserConfig)
}

func TestMySQLMigrationUserConfig_FullFields(t *testing.T) {
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
	mysql := newTestMySQL("default", "creds")
	result, err := resolveMySQLForTest(context.Background(), mysql, newFakeClient(s).Build())
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
	assert.Nil(t, mysql.Spec.UserConfig)
}

func TestMigrationUserConfig_StringFieldsPreservedAsStrings(t *testing.T) {
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
			result, err := resolvePGForTest(context.Background(), newTestPG("default", "creds"), newFakeClient(s).Build())
			require.NoError(t, err)

			cfg := result.(*pguserconfig.PgUserConfig)
			m := cfg.Migration
			assert.Equal(t, tt.password, *m.Password)
			assert.Equal(t, tt.username, *m.Username)
		})
	}
}

func TestMigrationUserConfig_Errors(t *testing.T) {
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
		{"trailing newline in password", "default", "creds", map[string][]byte{"host": []byte("db.example.com"), "port": []byte("5432"), "password": []byte("pw\n")}, "whitespace"},
		{"leading space in host", "default", "creds", map[string][]byte{"host": []byte(" db.example.com"), "port": []byte("5432")}, "whitespace"},
	}
	for _, mk := range migrationKinds {
		for _, tt := range cases {
			t.Run(mk.name+"/"+tt.name, func(t *testing.T) {
				var k *fake.ClientBuilder
				if tt.data != nil {
					k = newFakeClient(secretInNs(tt.secret, tt.ns, tt.data))
				} else {
					k = newFakeClient()
				}
				_, err := mk.resolve(context.Background(), tt.ns, tt.secret, k)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			})
		}
	}
}

func TestMySQLMigrationUserConfig_InvalidReestablishReplication(t *testing.T) {
	t.Parallel()

	s := secretInNs("creds", "default", map[string][]byte{
		"host":                    []byte("mysql-source.example.com"),
		"port":                    []byte("3306"),
		"reestablish_replication": []byte("nope"),
	})
	_, err := resolveMySQLForTest(context.Background(), newTestMySQL("default", "creds"), newFakeClient(s).Build())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reestablish_replication")
}

func TestPostgreSQLMigrationUserConfig_SecretOverridesInlineConfig(t *testing.T) {
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
	result, err := resolvePGForTest(context.Background(), pg, newFakeClient(s).Build())
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

func TestPostgreSQLMigrationUserConfig_NilRef(t *testing.T) {
	t.Parallel()

	pg := &v1alpha1.PostgreSQL{}
	result, err := resolvePGForTest(context.Background(), pg, nil)
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

func newTestPG(ns string, secretName string) *v1alpha1.PostgreSQL {
	pg := &v1alpha1.PostgreSQL{}
	pg.Namespace = ns
	pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: secretName}
	return pg
}

// resolvePGForTest resolves the PostgreSQL user config through the service controller.
func resolvePGForTest(ctx context.Context, pg *v1alpha1.PostgreSQL, k8s client.Client) (any, error) {
	return newTestController(nil, postgreSQLKind, k8s).resolveUserConfig(ctx, pg)
}

func newTestMySQL(ns string, secretName string) *v1alpha1.MySQL {
	mysql := &v1alpha1.MySQL{}
	mysql.Namespace = ns
	mysql.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: secretName}
	return mysql
}

// resolveMySQLForTest resolves the MySQL user config through the service controller.
func resolveMySQLForTest(ctx context.Context, mysql *v1alpha1.MySQL, k8s client.Client) (any, error) {
	return newTestController(nil, mySQLKind, k8s).resolveUserConfig(ctx, mysql)
}

type migrationKindCase struct {
	name    string
	resolve func(ctx context.Context, ns, secretName string, k8s *fake.ClientBuilder) (any, error)
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

func markMigrationFailed(pg *v1alpha1.PostgreSQL) {
	pg.Status.Conditions = append(pg.Status.Conditions, metav1.Condition{
		Type:    v1alpha1.ConditionTypeMigrationComplete,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.MigrationReasonFailed,
		Message: "connection refused",
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

	newPGWithClient := func(k8s client.Client, deleteAfter bool) (*v1alpha1.PostgreSQL, *serviceController[*v1alpha1.PostgreSQL]) {
		pg := &v1alpha1.PostgreSQL{}
		pg.Namespace = "default"
		pg.Name = "pg-1"
		pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{
			Name:                 "creds",
			DeleteAfterMigration: deleteAfter,
		}
		return pg, newTestController(nil, postgreSQLKind, k8s)
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

	t.Run("Deletes secret when flag true and migration complete", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(true)
		pg, c := newPGWithClient(k8s, true)
		markMigrationDone(pg)

		require.NoError(t, c.maybeDeleteMigrationSecret(context.Background(), pg, pg.Spec.MigrationSecretSource))
		assert.False(t, secretExists(t, k8s, "creds", "default"))
	})

	t.Run("Does not delete when flag is false", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(true)
		pg, c := newPGWithClient(k8s, false)
		markMigrationDone(pg)

		require.NoError(t, c.maybeDeleteMigrationSecret(context.Background(), pg, pg.Spec.MigrationSecretSource))
		assert.True(t, secretExists(t, k8s, "creds", "default"))
	})

	t.Run("Does not delete when condition absent", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(true)
		pg, c := newPGWithClient(k8s, true)

		require.NoError(t, c.maybeDeleteMigrationSecret(context.Background(), pg, pg.Spec.MigrationSecretSource))
		assert.True(t, secretExists(t, k8s, "creds", "default"))
	})

	t.Run("Does not delete when migration is still in progress", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(true)
		pg, c := newPGWithClient(k8s, true)
		markMigrationInProgress(pg)

		require.NoError(t, c.maybeDeleteMigrationSecret(context.Background(), pg, pg.Spec.MigrationSecretSource))
		assert.True(t, secretExists(t, k8s, "creds", "default"))
	})

	t.Run("Keeps the secret of a failed migration on 404", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(true)
		pg, _ := newPGWithClient(k8s, true)
		markMigrationFailed(pg)
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil, avngen.Error{Status: http.StatusNotFound, Message: "not found"}).Once()
		c := newTestController(avn, postgreSQLKind, k8s)

		_, err := c.observeMigration(context.Background(), pg)
		require.NoError(t, err)
		assert.True(t, secretExists(t, k8s, "creds", "default"))
	})

	t.Run("Idempotent when secret already gone", func(t *testing.T) {
		t.Parallel()

		k8s := buildClient(false)
		pg, c := newPGWithClient(k8s, true)
		markMigrationDone(pg)

		require.NoError(t, c.maybeDeleteMigrationSecret(context.Background(), pg, pg.Spec.MigrationSecretSource))
	})
}

func TestUpdateMigrationStatus(t *testing.T) {
	t.Parallel()

	newPGWithMigration := func() *v1alpha1.PostgreSQL {
		pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
		pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: "creds"}
		return pg
	}
	update := func(t *testing.T, avn avngen.Client, pg *v1alpha1.PostgreSQL) error {
		return updateMigrationCondition(t.Context(), avn, pg.Spec.Project, pg.Name, pg.Generation, &pg.Status.Conditions)
	}

	t.Run("Sets MigrationDone on 404 (no active migration)", func(t *testing.T) {
		t.Parallel()

		pg := newPGWithMigration()
		markMigrationApplied(&pg.Status.Conditions, pg.Generation)
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil, avngen.Error{Status: http.StatusNotFound, Message: "not found"}).Once()

		require.NoError(t, update(t, avn, pg))

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, v1alpha1.MigrationReasonDone, cond.Reason)
	})

	t.Run("Leaves the condition unset on 404 when the migration config was never applied", func(t *testing.T) {
		t.Parallel()

		pg := newPGWithMigration()
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil, avngen.Error{Status: http.StatusNotFound, Message: "not found"}).Once()

		require.NoError(t, update(t, avn, pg))
		assert.Nil(t, meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete))
	})

	t.Run("Keeps MigrationFailed on 404", func(t *testing.T) {
		t.Parallel()

		pg := newPGWithMigration()
		markMigrationFailed(pg)
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil, avngen.Error{Status: http.StatusNotFound, Message: "not found"}).Once()

		require.NoError(t, update(t, avn, pg))

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, v1alpha1.MigrationReasonFailed, cond.Reason)
	})

	t.Run("Sets MigrationDone when status is done", func(t *testing.T) {
		t.Parallel()

		pg := newPGWithMigration()
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetMigrationStatusOut{
				Migration: service.MigrationOut{Status: service.MigrationStatusTypeDone},
			}, nil).Once()

		require.NoError(t, update(t, avn, pg))

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, v1alpha1.MigrationReasonDone, cond.Reason)
	})

	t.Run("Sets MigrationFailed with error message", func(t *testing.T) {
		t.Parallel()

		pg := newPGWithMigration()
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetMigrationStatusOut{
				Migration: service.MigrationOut{
					Status: service.MigrationStatusTypeFailed,
					Error:  new("connection refused"),
				},
			}, nil).Once()

		require.NoError(t, update(t, avn, pg))

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, v1alpha1.MigrationReasonFailed, cond.Reason)
		assert.Equal(t, "connection refused", cond.Message)
	})

	t.Run("Sets MigrationInProgress for other statuses", func(t *testing.T) {
		t.Parallel()

		pg := newPGWithMigration()
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetMigrationStatusOut{
				Migration: service.MigrationOut{Status: "syncing"},
			}, nil).Once()

		require.NoError(t, update(t, avn, pg))

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, v1alpha1.MigrationReasonInProgress, cond.Reason)
	})

	t.Run("Returns error on non-404 error", func(t *testing.T) {
		t.Parallel()

		pg := newPGWithMigration()
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil, avngen.Error{Status: http.StatusInternalServerError, Message: "server error"}).Once()

		err := update(t, avn, pg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server error")

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		assert.Nil(t, cond)
	})
}

func TestHasPendingMigration(t *testing.T) {
	t.Parallel()

	t.Run("No condition — returns false", func(t *testing.T) {
		t.Parallel()
		pg := &v1alpha1.PostgreSQL{}
		assert.False(t, hasPendingMigration(pg))
	})

	t.Run("Condition True — returns false", func(t *testing.T) {
		t.Parallel()
		pg := &v1alpha1.PostgreSQL{}
		meta.SetStatusCondition(&pg.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.ConditionTypeMigrationComplete,
			Status: metav1.ConditionTrue,
			Reason: v1alpha1.MigrationReasonDone,
		})
		assert.False(t, hasPendingMigration(pg))
	})

	t.Run("Condition False (in progress) — returns true", func(t *testing.T) {
		t.Parallel()
		pg := &v1alpha1.PostgreSQL{}
		meta.SetStatusCondition(&pg.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.ConditionTypeMigrationComplete,
			Status: metav1.ConditionFalse,
			Reason: v1alpha1.MigrationReasonInProgress,
		})
		assert.True(t, hasPendingMigration(pg))
	})

	t.Run("Condition False (failed) — returns false (terminal state)", func(t *testing.T) {
		t.Parallel()
		pg := &v1alpha1.PostgreSQL{}
		meta.SetStatusCondition(&pg.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.ConditionTypeMigrationComplete,
			Status: metav1.ConditionFalse,
			Reason: v1alpha1.MigrationReasonFailed,
		})
		assert.False(t, hasPendingMigration(pg))
	})

	t.Run("Condition False with unknown Reason — returns false", func(t *testing.T) {
		t.Parallel()
		pg := &v1alpha1.PostgreSQL{}
		meta.SetStatusCondition(&pg.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.ConditionTypeMigrationComplete,
			Status: metav1.ConditionFalse,
			Reason: "SomethingElse",
		})
		assert.False(t, hasPendingMigration(pg))
	})
}
