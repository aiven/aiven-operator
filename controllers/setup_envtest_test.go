package controllers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// TestSetupControllersWithCRDSubset installs only the Kafka and KafkaTopic CRDs and checks that the
// operator starts when --controllers matches them, and fails to sync when it does not.
func TestSetupControllersWithCRDSubset(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("KUBEBUILDER_ASSETS is unset; run via 'task test:unit'")
	}

	env := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths: []string{
			"../config/crd/bases/aiven.io_kafkas.yaml",
			"../config/crd/bases/aiven.io_kafkatopics.yaml",
		},
	}
	restConfig, err := env.Start()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, env.Stop()) })

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	const cacheSyncTimeout = 5 * time.Second

	// run builds a manager with the given --controllers value, sets up the controllers and starts it,
	// returning the first error, or nil once the cache has synced and the manager has stopped cleanly.
	run := func(t *testing.T, controllers string) error {
		mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
			Scheme:  scheme,
			Metrics: metricsserver.Options{BindAddress: "0"},
			Controller: config.Controller{
				CacheSyncTimeout: cacheSyncTimeout,
				// Both subtests register the same controller names in one process.
				SkipNameValidation: new(true),
			},
		})
		require.NoError(t, err)

		if err := SetupControllers(mgr, SetupConfig{DefaultToken: "token", Controllers: controllers}); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*cacheSyncTimeout)
		defer cancel()
		errCh := make(chan error, 1)
		go func() { errCh <- mgr.Start(ctx) }()
		// The field indexers requested every enabled kind's informer at setup, so a sync covers them all.
		require.True(t, mgr.GetCache().WaitForCacheSync(ctx), "cache did not sync")
		cancel()
		return <-errCh
	}

	t.Run("matching subset starts", func(t *testing.T) {
		require.NoError(t, run(t, "Kafka,KafkaTopic"))
	})

	// The secret finalizer GC controller indexes every enabled kind at setup, so a missing CRD is
	// fatal before the manager even starts.
	t.Run("all controllers fail on the missing CRDs", func(t *testing.T) {
		err := run(t, "*")
		require.ErrorContains(t, err, "unable to add index for secret ref fields: no matches for kind")
	})
}
