//go:build suite

package tests

import (
	"context"
	"fmt"
	"log"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	clickhouseuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/clickhouse"
	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
)

// SharedResources creates and manages shared resources that can be used across multiple tests.
// Destroys all the resources on Destroy() on session teardown.
// AcquireX holds the service shared with other tests; AcquireXExclusive holds it alone,
// for tests that change service-level state (user config, avnadmin, integration counts).
type SharedResources interface {
	AcquirePostgreSQL(ctx context.Context) (*v1alpha1.PostgreSQL, func(), error)
	AcquirePostgreSQLExclusive(ctx context.Context) (*v1alpha1.PostgreSQL, func(), error)
	AcquireClickhouse(ctx context.Context) (*v1alpha1.Clickhouse, func(), error)
	AcquireClickhouseExclusive(ctx context.Context) (*v1alpha1.Clickhouse, func(), error)
	AcquireKafka(ctx context.Context) (*v1alpha1.Kafka, func(), error)
	Destroy() error
}

type sharedResourcesImpl struct {
	resources sync.Map // map[string]*sharedResource
	session   Session
}

type sharedResource struct {
	lock   sync.RWMutex // held for the whole test: read for shared users, write for exclusive ones
	create sync.Mutex   // guards obj
	obj    client.Object
}

func NewSharedResources(ctx context.Context, k8sClient client.Client) SharedResources {
	s := &sharedResourcesImpl{
		session: NewSession(ctx, k8sClient),
	}
	return s
}

func (s *sharedResourcesImpl) AcquirePostgreSQL(ctx context.Context) (*v1alpha1.PostgreSQL, func(), error) {
	return s.acquirePostgreSQL(ctx, false)
}

func (s *sharedResourcesImpl) AcquirePostgreSQLExclusive(ctx context.Context) (*v1alpha1.PostgreSQL, func(), error) {
	return s.acquirePostgreSQL(ctx, true)
}

func (s *sharedResourcesImpl) acquirePostgreSQL(ctx context.Context, exclusive bool) (*v1alpha1.PostgreSQL, func(), error) {
	obj := &v1alpha1.PostgreSQL{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "PostgreSQL",
		},
	}
	obj.Spec.Plan = "startup-4"
	obj.Spec.Project = cfg.Project
	obj.Spec.CloudName = cfg.PrimaryCloudName
	return acquire(ctx, s, "PostgreSQL", obj, exclusive)
}

func (s *sharedResourcesImpl) AcquireClickhouse(ctx context.Context) (*v1alpha1.Clickhouse, func(), error) {
	return s.acquireClickhouse(ctx, false)
}

func (s *sharedResourcesImpl) AcquireClickhouseExclusive(ctx context.Context) (*v1alpha1.Clickhouse, func(), error) {
	return s.acquireClickhouse(ctx, true)
}

func (s *sharedResourcesImpl) acquireClickhouse(ctx context.Context, exclusive bool) (*v1alpha1.Clickhouse, func(), error) {
	obj := &v1alpha1.Clickhouse{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Clickhouse",
		},
	}
	obj.Spec.Plan = "startup-16"
	obj.Spec.Project = cfg.Project
	obj.Spec.CloudName = cfg.PrimaryCloudName
	obj.Spec.UserConfig = &clickhouseuserconfig.ClickhouseUserConfig{
		ClickhouseVersion: anyPointer("25.3"),
	}
	return acquire(ctx, s, "Clickhouse", obj, exclusive)
}

func (s *sharedResourcesImpl) AcquireKafka(ctx context.Context) (*v1alpha1.Kafka, func(), error) {
	obj := &v1alpha1.Kafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Kafka",
		},
	}
	obj.Spec.Plan = "business-4"
	obj.Spec.Project = cfg.Project
	obj.Spec.CloudName = cfg.PrimaryCloudName
	// SASL for TestServiceUserRotationKafka; the other users don't depend on the auth methods.
	obj.Spec.UserConfig = &kafkauserconfig.KafkaUserConfig{
		SchemaRegistry: anyPointer(true),
		KafkaAuthenticationMethods: &kafkauserconfig.KafkaAuthenticationMethods{
			Certificate: anyPointer(true), Sasl: anyPointer(true),
		},
		KafkaSaslMechanisms: &kafkauserconfig.KafkaSaslMechanisms{Plain: anyPointer(true)},
	}
	return acquire(ctx, s, "Kafka", obj, false)
}

// acquire returns a copy of a shared resource: first call creates the resource.
// todo: listen for context cancellation and release the lock if it happens
func acquire[T client.Object](_ context.Context, s *sharedResourcesImpl, key string, obj T, exclusive bool) (T, func(), error) {
	v, _ := s.resources.LoadOrStore(key, new(sharedResource))
	r := v.(*sharedResource)
	if err := r.ensure(s.session, key, obj); err != nil {
		return obj, nil, err
	}

	lock, unlock, mode := r.lock.RLock, r.lock.RUnlock, "shared"
	if exclusive {
		lock, unlock, mode = r.lock.Lock, r.lock.Unlock, "exclusive"
	}
	lock()
	log.Printf("Locked shared resource %q (%s)", key, mode)

	// Each test gets its own copy, so concurrent holders can Get into it without racing.
	got := r.obj.DeepCopyObject().(T)
	releaseFunc := func() {
		log.Printf("SHARED RESOURCE RELEASE: Releasing shared resource %q (name: %s, %s)", key, got.GetName(), mode)
		unlock()
		log.Printf("SHARED RESOURCE RELEASE: Released shared resource %q", key)
	}
	return got, releaseFunc, nil
}

// ensure creates the resource once; a failed attempt is retried by the next caller.
func (r *sharedResource) ensure(session Session, key string, obj client.Object) error {
	r.create.Lock()
	defer r.create.Unlock()
	if r.obj != nil {
		log.Printf("Using shared resource from cache %q", key)
		return nil
	}

	// Generate a random name for the resource if not set.
	// The resource then is cached, so it is random only on the first call.
	if obj.GetName() == "" {
		obj.SetName(randName(key))
	}

	if err := session.ApplyObjects(obj); err != nil {
		log.Printf("Failed to create shared resource %q: %s", key, err)
		return err
	}

	if err := session.GetRunning(obj, obj.GetName()); err != nil {
		return fmt.Errorf("failed to get running shared resource %q: %w", key, err)
	}

	r.obj = obj
	log.Printf("Shared resource %q created", key)
	return nil
}

func (s *sharedResourcesImpl) Destroy() error {
	s.resources.Range(func(key, _ any) bool {
		log.Printf("Destroying shared resource %q", key)
		return true
	})

	return s.session.DestroyError()
}
