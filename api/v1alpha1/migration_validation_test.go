package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mysqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/mysql"
	pguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/pg"
)

func TestPostgreSQLSpec_migrationWarnings(t *testing.T) {
	tests := []struct {
		name        string
		spec        PostgreSQLSpec
		wantWarning bool
	}{
		{
			name: "neither set",
			spec: PostgreSQLSpec{},
		},
		{
			name: "only migrationSecretSource",
			spec: PostgreSQLSpec{
				MigrationSecretSource: &MigrationSecretSource{Name: "creds"},
			},
		},
		{
			name: "only userConfig.migration",
			spec: PostgreSQLSpec{
				UserConfig: &pguserconfig.PgUserConfig{
					Migration: &pguserconfig.Migration{Host: "db.example.com", Port: 5432},
				},
			},
		},
		{
			name: "both set — warning, not error",
			spec: PostgreSQLSpec{
				MigrationSecretSource: &MigrationSecretSource{Name: "creds"},
				UserConfig: &pguserconfig.PgUserConfig{
					Migration: &pguserconfig.Migration{Host: "db.example.com", Port: 5432},
				},
			},
			wantWarning: true,
		},
		{
			name: "migrationSecretSource with userConfig but no migration",
			spec: PostgreSQLSpec{
				MigrationSecretSource: &MigrationSecretSource{Name: "creds"},
				UserConfig:            &pguserconfig.PgUserConfig{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := tt.spec.migrationWarnings()
			if tt.wantWarning {
				assert.NotEmpty(t, warnings)
				assert.Contains(t, warnings[0], "migrationSecretSource is set")
			} else {
				assert.Empty(t, warnings)
			}
		})
	}
}

func TestMySQLSpec_migrationWarnings(t *testing.T) {
	tests := []struct {
		name        string
		spec        MySQLSpec
		wantWarning bool
	}{
		{
			name: "neither set",
			spec: MySQLSpec{},
		},
		{
			name: "only migrationSecretSource",
			spec: MySQLSpec{
				MigrationSecretSource: &MigrationSecretSource{Name: "creds"},
			},
		},
		{
			name: "only userConfig.migration",
			spec: MySQLSpec{
				UserConfig: &mysqluserconfig.MysqlUserConfig{
					Migration: &mysqluserconfig.Migration{Host: "db.example.com", Port: 3306},
				},
			},
		},
		{
			name: "both set — warning, not error",
			spec: MySQLSpec{
				MigrationSecretSource: &MigrationSecretSource{Name: "creds"},
				UserConfig: &mysqluserconfig.MysqlUserConfig{
					Migration: &mysqluserconfig.Migration{Host: "db.example.com", Port: 3306},
				},
			},
			wantWarning: true,
		},
		{
			name: "migrationSecretSource with userConfig but no migration",
			spec: MySQLSpec{
				MigrationSecretSource: &MigrationSecretSource{Name: "creds"},
				UserConfig:            &mysqluserconfig.MysqlUserConfig{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := tt.spec.migrationWarnings()
			if tt.wantWarning {
				assert.NotEmpty(t, warnings)
				assert.Contains(t, warnings[0], "migrationSecretSource is set")
			} else {
				assert.Empty(t, warnings)
			}
		})
	}
}
