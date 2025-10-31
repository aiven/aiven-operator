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
	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
)

// SharedResources creates and manages shared resources that can be used across multiple tests.
// Destroys all the resources on Destroy() on session teardown.
type SharedResources interface {
	AcquirePostgreSQL(ctx context.Context) (*v1alpha1.PostgreSQL, func(), error)
	AcquireClickhouse(ctx context.Context) (*v1alpha1.Clickhouse, func(), error)
	AcquireKafka(ctx context.Context) (*v1alpha1.Kafka, func(), error)
	Destroy() error
}

type sharedResourcesImpl struct {
	mutexes   sync.Map // map[sharedResourceKeyType]*sync.Mutex
	resources sync.Map // map[sharedResourceKeyType]client.Object
	session   Session
}

func NewSharedResources(ctx context.Context, k8sClient client.Client) SharedResources {
	s := &sharedResourcesImpl{
		session: NewSession(ctx, k8sClient),
	}
	return s
}

func (s *sharedResourcesImpl) AcquirePostgreSQL(ctx context.Context) (*v1alpha1.PostgreSQL, func(), error) {
	obj := &v1alpha1.PostgreSQL{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "PostgreSQL",
		},
	}
	obj.Spec.Plan = "startup-4"
	obj.Spec.Project = cfg.Project
	obj.Spec.CloudName = cfg.PrimaryCloudName
	return acquire(ctx, s, "PostgreSQL", obj)
}

func (s *sharedResourcesImpl) AcquireClickhouse(ctx context.Context) (*v1alpha1.Clickhouse, func(), error) {
	obj := &v1alpha1.Clickhouse{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Clickhouse",
		},
	}
	obj.Spec.Plan = "startup-16"
	obj.Spec.Project = cfg.Project
	obj.Spec.CloudName = cfg.PrimaryCloudName
	return acquire(ctx, s, "Clickhouse", obj)
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
	obj.Spec.UserConfig = &kafkauserconfig.KafkaUserConfig{
		SchemaRegistry: anyPointer(true),
	}
	return acquire(ctx, s, "Kafka", obj)
}

// acquire returns a shared resource: first call creates the resource.
// todo: listen for context cancellation and release the lock if it happens
func acquire[T client.Object](_ context.Context, s *sharedResourcesImpl, key string, obj T) (T, func(), error) {
	mtx, _ := s.mutexes.LoadOrStore(key, &sync.Mutex{})
	m := mtx.(*sync.Mutex)
	m.Lock()
	log.Printf("Locked shared resource %q", key)

	var err error
	defer func() {
		if err != nil {
			m.Unlock()
			log.Printf("Unlocked shared resource %q: %s", key, err)
		}
	}()

	v, ok := s.resources.Load(key)
	if ok {
		log.Printf("Using shared resource from cache %q", key)
		releaseFunc := func() {
			log.Printf("SHARED RESOURCE RELEASE: Releasing cached shared resource %q (name: %s)", key, v.(T).GetName())
			m.Unlock()
			log.Printf("SHARED RESOURCE RELEASE: Released cached shared resource %q", key)
		}
		return v.(T), releaseFunc, nil
	}

	// Generate a random name for the resource if not set.
	// The resource then is cached, so it is random only on the first call.
	if obj.GetName() == "" {
		obj.SetName(randName(key))
	}

	err = s.session.ApplyObjects(obj)
	if err != nil {
		return obj, nil, err
	}

	err = s.session.GetRunning(obj, obj.GetName())
	if err != nil {
		err = fmt.Errorf("failed to get running shared resource %q: %w", key, err)
		return obj, nil, err
	}

	s.resources.Store(key, obj)
	log.Printf("Shared resource %q created", key)
	releaseFunc := func() {
		log.Printf("SHARED RESOURCE RELEASE: Releasing shared resource %q (name: %s)", key, obj.GetName())
		m.Unlock()
		log.Printf("SHARED RESOURCE RELEASE: Released shared resource %q", key)
	}
	return obj, releaseFunc, nil
}

func (s *sharedResourcesImpl) Destroy() error {
	s.resources.Range(func(key, _ any) bool {
		log.Printf("Destroying shared resource %q", key)
		return true
	})

	return s.session.DestroyError()
}
