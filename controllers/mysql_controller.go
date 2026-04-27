// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	mysqluserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/mysql"
)

// MySQLReconciler reconciles a MySQL object
type MySQLReconciler struct {
	Controller
}

func newMySQLReconciler(c Controller) reconcilerType {
	return &MySQLReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=mysqls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=mysqls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=mysqls/finalizers,verbs=get;create;update

func (r *MySQLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandlerWithClient(r.Client, newMySQLAdapterFactory(r.Client), r.Log), &v1alpha1.MySQL{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *MySQLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.MySQL{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newMySQLAdapterFactory(k8s client.Client) serviceAdapterFabric {
	return func(object client.Object) (serviceAdapter, error) {
		mysql, ok := object.(*v1alpha1.MySQL)
		if !ok {
			return nil, fmt.Errorf("object is not of type v1alpha1.MySQL")
		}
		return &mySQLAdapter{MySQL: mysql, k8s: k8s}, nil
	}
}

// mySQLAdapter handles an Aiven MySQL service
type mySQLAdapter struct {
	*v1alpha1.MySQL
	k8s client.Client
}

func (a *mySQLAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *mySQLAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *mySQLAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *mySQLAdapter) getUserConfig() any {
	return a.Spec.UserConfig
}

func (a *mySQLAdapter) newSecret(_ context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
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

	return newSecret(a, stringData, true), nil
}

func (a *mySQLAdapter) getMigrationSecretSource() *v1alpha1.MigrationSecretSource {
	return a.Spec.MigrationSecretSource
}

// buildUserConfigWithMigration merges pre-resolved migration Secret data into a
// clone of the user config. Pure: no I/O, no context. The Secret is read upstream
// in the handler; this method only maps fields and validates them.
func (a *mySQLAdapter) buildUserConfigWithMigration(data map[string]string) (any, error) {
	ref := a.Spec.MigrationSecretSource
	if ref == nil {
		return a.getUserConfig(), nil
	}

	port, err := parseMigrationPort(ref.Name, data["port"])
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
	if a.Spec.UserConfig != nil {
		clone := *a.Spec.UserConfig
		cfg = &clone
	}
	cfg.Migration = m
	return cfg, nil
}

func (a *mySQLAdapter) getServiceType() serviceType {
	return serviceTypeMySQL
}

func (a *mySQLAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *mySQLAdapter) performUpgradeTaskIfNeeded(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}

func (a *mySQLAdapter) createOrUpdateServiceSpecific(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}

func (a *mySQLAdapter) deleteMigrationSecret(ctx context.Context) error {
	return deleteMigrationSecret(ctx, a.k8s, a.GetNamespace(), a.Spec.MigrationSecretSource)
}
