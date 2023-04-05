package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/aiven/aiven-go-client"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/controllers"
)

const (
	retryInterval      = time.Second * 5
	createTimeout      = time.Second * 15
	waitRunningTimeout = time.Minute * 10
	yamlBufferSize     = 100
)

type session struct {
	k8s     client.Client
	avn     *aiven.Client
	objs    []client.Object
	project string
}

func NewSession(k8s client.Client, avn *aiven.Client, project, src string) (*session, error) {
	s := &session{
		k8s:     k8s,
		avn:     avn,
		project: project,
	}

	// Creds: https://gist.github.com/pytimer/0ad436972a073bb37b8b6b8b474520fc
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(src)), yamlBufferSize)
	for {
		var rawExt runtime.RawExtension
		if err := decoder.Decode(&rawExt); err != nil {
			break
		}

		if len(rawExt.Raw) == 0 {
			// if the yaml object is empty just continue to the next one
			continue
		}

		uObj, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawExt.Raw, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize raw object %s", err)
		}

		uMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(uObj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to unstructured map %s", err)
		}

		obj := unstructured.Unstructured{Object: uMap}
		if obj.GetNamespace() == "" {
			obj.SetNamespace(defaultNamespace)
		}

		s.objs = append(s.objs, &obj)
	}
	return s, nil
}

func (s *session) Apply() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()
	var g errgroup.Group
	for i := range s.objs {
		o := s.objs[i]
		g.Go(func() error {
			return s.k8s.Create(ctx, o)
		})
	}
	return g.Wait()
}

func (s *session) GetRunning(obj client.Object, keys ...string) error {
	var name, namespace string
	switch len(keys) {
	case 1:
		name = keys[0]
		namespace = defaultNamespace
	case 2: //nolint:gomnd
		name = keys[0]
		namespace = keys[1]
	default:
		return fmt.Errorf("provide name or/and namespace")
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, waitRunningTimeout)
	defer cancel()

	key := types.NamespacedName{Name: name, Namespace: namespace}
	return retryForever(ctx, func() (bool, error) {
		err := s.k8s.Get(ctx, key, obj)
		if err != nil {
			// The error is quite verbose
			log.Printf("waiting resource running: %s", err)

			// Do not retry kube errors
			return isNotFound(err), err
		}
		return !controllers.IsAlreadyRunning(obj), nil
	})
}

func (s *session) Destroy() {
	if len(s.objs) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(s.objs))

	ctx := context.Background()
	for i := range s.objs {
		o := s.objs[i]
		go func() {
			defer wg.Done()
			err := k8sDelete(ctx, s.k8s, o)
			if err != nil {
				log.Printf("failed to delete %s: %s", o.GetName(), err)
			}
		}()
	}
	wg.Wait()
}

func retryForever(ctx context.Context, f func() (bool, error)) (err error) {
	retry := false
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context timeout while retrying operation, error=%q", err)
		case <-time.After(retryInterval):
			retry, err = f()
			if retry {
				continue
			}
			return err
		}
	}
}

const (
	randIDSize = 7
	// randIDChars Aiven allowed chars for "names"
	randIDChars = "0123456789abcdefghijklmnopqrstuvwxyz"
)

// randID generates Aiven compatible random id
func randID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, randIDSize)
	for i := range b {
		b[i] = randIDChars[r.Intn(len(randIDChars))]
	}
	return string(b)
}

func randName(name string) string {
	return fmt.Sprintf("test-%s-%s", randID(), name)
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

func anyPointer[T any](v T) *T {
	return &v
}

// castInterface Using json is the easiest way to convert interface{} into object
func castInterface(in, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// k8sDelete deletes object from kube, and makes sure it is not available anymore
func k8sDelete(ctx context.Context, k8s client.Client, o client.Object) error {
	err := k8s.Delete(ctx, o)
	if err != nil {
		return fmt.Errorf("kubernetes error: %w", err)
	}

	// Waits being deleted from kube
	key := types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}
	return retryForever(ctx, func() (bool, error) {
		err := k8s.Get(ctx, key, o)
		return !isNotFound(err), nil
	})
}
