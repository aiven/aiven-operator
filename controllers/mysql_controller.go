// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"fmt"
	"strconv"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	mysqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/mysql"
)

//+kubebuilder:rbac:groups=aiven.io,resources=mysqls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=mysqls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=mysqls/finalizers,verbs=get;create;update

func newMySQLReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.MySQL] {
			return newServiceController(c, avnGen, mySQLKind)
		},
		nil,
	)
}

// mySQLKind is the serviceController definition for MySQL.
var mySQLKind = newServiceKind(serviceTypeMySQL, mySQLSecretData,
	withProjectCACert[*v1alpha1.MySQL](),
	withMigrationSecret(func(o *v1alpha1.MySQL) *v1alpha1.MigrationSecretSource { return o.Spec.MigrationSecretSource }, mySQLMigrationUserConfig),
)

// mySQLSecretData returns the prefixed connection secret keys.
func mySQLSecretData(obj *v1alpha1.MySQL, s *service.ServiceGetOut) map[string]string {
	stringData := map[string]string{
		"HOST":        s.ServiceUriParams["host"],
		"PORT":        s.ServiceUriParams["port"],
		"DATABASE":    s.ServiceUriParams["dbname"],
		"USER":        s.ServiceUriParams["user"],
		"PASSWORD":    s.ServiceUriParams["password"],
		"SSL_MODE":    s.ServiceUriParams["ssl-mode"],
		"URI":         s.ServiceUri,
		"REPLICA_URI": *s.ConnectionInfo.MysqlReplicaUri,
	}

	return prefixKeys(getSecretPrefix(obj), stringData)
}

// mySQLMigrationUserConfig merges pre-resolved migration Secret data into a clone of the user config.
func mySQLMigrationUserConfig(obj *v1alpha1.MySQL, data map[string]string) (any, error) {
	ref := obj.Spec.MigrationSecretSource
	if ref == nil {
		return obj.Spec.UserConfig, nil
	}

	port, err := parseMigrationPort(data["port"])
	if err != nil {
		return nil, err
	}

	host := data["host"]
	if host == "" {
		return nil, fmt.Errorf("migration secret %q must contain \"host\"", ref.Name)
	}

	m := &mysqluserconfig.Migration{
		Host: host,
		Port: port,
	}
	if v, ok := data["password"]; ok {
		m.Password = new(v)
	}
	if v, ok := data["dbname"]; ok {
		m.Dbname = new(v)
	}
	if v, ok := data["username"]; ok {
		m.Username = new(v)
	}
	if v, ok := data["ssl"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("migration secret %q: invalid ssl value %q: %w", ref.Name, v, err)
		}
		m.Ssl = new(b)
	}
	if v, ok := data["method"]; ok {
		m.Method = new(v)
	}
	if v, ok := data["ignore_dbs"]; ok {
		m.IgnoreDbs = new(v)
	}
	if v, ok := data["ignore_roles"]; ok {
		m.IgnoreRoles = new(v)
	}
	if v, ok := data["dump_tool"]; ok {
		m.DumpTool = new(v)
	}
	if v, ok := data["reestablish_replication"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("migration secret %q: invalid reestablish_replication value %q: %w", ref.Name, v, err)
		}
		m.ReestablishReplication = new(b)
	}

	// Build a shallow copy of the user config with migration from the secret.
	// The original CR spec is never mutated.
	cfg := &mysqluserconfig.MysqlUserConfig{}
	if obj.Spec.UserConfig != nil {
		clone := *obj.Spec.UserConfig
		cfg = &clone
	}
	cfg.Migration = m
	return cfg, nil
}
