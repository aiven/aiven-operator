package tests

import (
	"context"
	"fmt"
	"log"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// SharedResources creates and manages shared resources that can be used across multiple tests.
// Destroys all the resources on Destroy() on session teardown.
type SharedResources interface {
	Acquire(key sharedResourceKeyType) (client.Object, func(), error)
	Destroy() error
}

type sharedResourceKeyType string

const (
	pgStartup4SharedResource sharedResourceKeyType = "pg-startup-4"
)

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

// Acquire returns a shared resource: first call creates the resource.
func (s *sharedResourcesImpl) Acquire(key sharedResourceKeyType) (obj client.Object, release func(), err error) {
	log.Printf("Acquiring shared %q", key)
	mtx, _ := s.mutexes.LoadOrStore(key, &sync.Mutex{})
	m := mtx.(*sync.Mutex)
	m.Lock()
	log.Printf("Locked shared %q", key)

	defer func() {
		if err != nil {
			m.Unlock()
			log.Printf("Unlocked shared %q, error: %s", key, err)
		}

		release = func() {
			m.Unlock()
			log.Printf("Unlocked shared %q, all good.", key)
		}
	}()

	v, ok := s.resources.Load(key)
	if ok {
		log.Printf("Using shared from cache %q", key)
		return v.(client.Object), release, nil
	}

	var yml string
	name := randName(key)
	obj, yml, err = configSharedResource(key, name)
	if err != nil {
		return nil, nil, err
	}

	s.resources.Store(key, obj)
	err = s.session.Apply(yml)
	if err != nil {
		return nil, nil, err
	}

	err = s.session.GetRunning(obj, name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get shared resource %q: %w", key, err)
	}

	log.Printf("Shared %q created", key)
	return obj, release, nil
}

func (s *sharedResourcesImpl) Destroy() error {
	s.resources.Range(func(key, _ any) bool {
		log.Printf("Destroying shared resource %q", key)
		return true
	})

	return s.session.DestroyError()
}

func configSharedResource(key sharedResourceKeyType, name string) (client.Object, string, error) {
	fileMap := map[sharedResourceKeyType]string{
		pgStartup4SharedResource: "postgresql.yaml",
	}

	objMap := map[sharedResourceKeyType]client.Object{
		pgStartup4SharedResource: new(v1alpha1.PostgreSQL),
	}

	filePath, ok := fileMap[key]
	if !ok {
		return nil, "", fmt.Errorf("shared resource %q not found in config", key)
	}

	yml, err := loadExampleYaml(filePath, map[string]string{
		"spec.cloudName": cfg.PrimaryCloudName,
		"spec.project":   cfg.Project,
		"metadata.name":  name,
	})
	if err != nil {
		return nil, "", err
	}

	return objMap[key], yml, nil
}
