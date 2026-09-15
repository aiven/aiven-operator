package controllers

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestValidatePollInterval(t *testing.T) {
	cases := []struct {
		name    string
		give    time.Duration
		wantErr bool
	}{
		{name: "zero means use the default", give: 0},
		{name: "at the floor", give: 10 * time.Minute},
		{name: "at the ceiling", give: 60 * time.Minute},
		{name: "within the range", give: 30 * time.Minute},
		{name: "below the floor", give: time.Minute, wantErr: true},
		{name: "just below the floor", give: 10*time.Minute - time.Second, wantErr: true},
		// The floor equals the default, so anything faster than the default is rejected.
		{name: "faster than the default", give: 5 * time.Minute, wantErr: true},
		{name: "just above the ceiling", give: 60*time.Minute + time.Second, wantErr: true},
		{name: "above the ceiling", give: 2 * time.Hour, wantErr: true},
		{name: "negative", give: -time.Minute, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePollInterval(tc.give)
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.give.String())
			require.Contains(t, err.Error(), "10m0s-1h0m0s")
		})
	}
}

func TestSetupConfigNormalizePollInterval(t *testing.T) {
	cases := []struct {
		name string
		give time.Duration
		want time.Duration
	}{
		{name: "unset falls back to the default", give: 0, want: DefaultPollInterval},
		{name: "negative falls back to the default", give: -time.Minute, want: DefaultPollInterval},
		{name: "a set value is left alone", give: 30 * time.Minute, want: 30 * time.Minute},
		{name: "an out-of-range value is not clamped", give: time.Minute, want: time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := SetupConfig{PollInterval: tc.give}
			cfg.normalize()
			require.Equal(t, tc.want, cfg.PollInterval)
		})
	}
}

// fakeManager records what SetupControllers registers.
type fakeManager struct {
	ctrl.Manager
	scheme *runtime.Scheme
	client client.Client

	indexed map[string][]string // index key -> kinds, in registration order
	added   int                 // controllers handed to Add
}

func newFakeManager(t *testing.T) *fakeManager {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	return &fakeManager{
		scheme:  scheme,
		client:  fake.NewClientBuilder().WithScheme(scheme).Build(),
		indexed: map[string][]string{},
	}
}

func (m *fakeManager) GetClient() client.Client             { return m.client }
func (m *fakeManager) GetScheme() *runtime.Scheme           { return m.scheme }
func (m *fakeManager) GetFieldIndexer() client.FieldIndexer { return m }
func (m *fakeManager) GetCache() cache.Cache                { return nil }
func (m *fakeManager) GetRESTMapper() apimeta.RESTMapper    { return nil }
func (m *fakeManager) GetLogger() logr.Logger               { return logr.Discard() }
func (m *fakeManager) GetEventRecorderFor(string) record.EventRecorder {
	return record.NewFakeRecorder(1)
}
func (m *fakeManager) Add(manager.Runnable) error { m.added++; return nil }

func (m *fakeManager) GetControllerOptions() config.Controller {
	// Subtests register the same controller names in one process.
	return config.Controller{SkipNameValidation: new(true)}
}

func (m *fakeManager) IndexField(_ context.Context, obj client.Object, field string, _ client.IndexerFunc) error {
	m.indexed[field] = append(m.indexed[field], reflect.TypeOf(obj).Elem().Name())
	return nil
}

func TestSetupControllers(t *testing.T) {
	t.Run("rejects an unknown kind before touching the manager", func(t *testing.T) {
		err := SetupControllers(nil, SetupConfig{Controllers: "Redis"})
		require.ErrorContains(t, err, `unknown kind "Redis"`)
	})

	t.Run("a subset indexes and registers only its kinds", func(t *testing.T) {
		mgr := newFakeManager(t)
		require.NoError(t, SetupControllers(mgr, SetupConfig{Controllers: "Kafka,KafkaTopic,ServiceUser"}))

		require.Equal(t, map[string][]string{
			secretRefIndexKey:         {"Kafka", "KafkaTopic", "ServiceUser"},
			connInfoSecretRefIndexKey: {"ServiceUser"},
		}, mgr.indexed)
		// The two secret controllers plus one per enabled kind.
		require.Equal(t, 2+3, mgr.added)
	})

	t.Run("the default registers every kind", func(t *testing.T) {
		mgr := newFakeManager(t)
		require.NoError(t, SetupControllers(mgr, SetupConfig{}))

		require.Equal(t, knownKinds(), mgr.indexed[secretRefIndexKey])
		require.Equal(t, []string{"ServiceUser", "ClickhouseUser"}, mgr.indexed[connInfoSecretRefIndexKey])
		require.Equal(t, []string{"KafkaSchema"}, mgr.indexed[kafkaSchemaRefIndex])
		require.Equal(t, 2+len(builders), mgr.added)
	})
}
