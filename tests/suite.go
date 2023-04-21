package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/aiven/aiven-go-client"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
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

type Session interface {
	Apply() error
	GetRunning(obj client.Object, keys ...string) error
	GetSecret(keys ...string) (*corev1.Secret, error)
	Destroy()
	Delete(o client.Object, exists func() error) error
}

var _ Session = &session{}

type session struct {
	k8s     client.Client
	avn     *aiven.Client
	ctx     context.Context
	objs    map[int]client.Object
	project string
}

func NewSession(k8s client.Client, avn *aiven.Client, project, src string) (Session, error) {
	s := &session{
		k8s:     k8s,
		avn:     avn,
		ctx:     context.Background(),
		objs:    make(map[int]client.Object),
		project: project,
	}

	var objIndex int

	// Creds: https://gist.github.com/pytimer/0ad436972a073bb37b8b6b8b474520fc
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(src)), yamlBufferSize)
	for {
		var rawExt runtime.RawExtension
		err := decoder.Decode(&rawExt)
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
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

		s.objs[objIndex] = &obj
		objIndex++
	}
	return s, nil
}

func (s *session) Apply() error {
	ctx, cancel := context.WithTimeout(s.ctx, createTimeout)
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
	key, err := getNamespacedName(keys...)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(s.ctx, waitRunningTimeout)
	defer cancel()

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

func (s *session) GetSecret(keys ...string) (*corev1.Secret, error) {
	key, err := getNamespacedName(keys...)
	if err != nil {
		return nil, err
	}

	secret := new(corev1.Secret)
	err = s.k8s.Get(s.ctx, key, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (s *session) Destroy() {
	if len(s.objs) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(s.objs))

	for i := range s.objs {
		go func(i int) {
			o := s.objs[i]
			defer wg.Done()
			if err := s.delete(o); err != nil {
				log.Printf("failed to delete %s: %s", o.GetName(), err)
			}
		}(i)
	}
	wg.Wait()
}

// Delete deletes object from kube, hence from Aiven
// Validates it is exists before deleted and after, to avoid false positive deletion
func (s *session) Delete(o client.Object, exists func() error) error {
	err := exists()
	if err != nil {
		return err
	}
	err = s.delete(o)
	if err != nil {
		return err
	}
	err = exists()
	if aiven.IsNotFound(err) {
		return nil
	}
	return err
}

// delete deletes object from kube, and makes sure it is not there anymore
// Removes from applied list to not delete object on Destroy()
func (s *session) delete(o client.Object) error {
	if o.GetUID() == "" {
		return fmt.Errorf("object %s has no UID to delete by", o.GetName())
	}

	oldLen := len(s.objs)
	for i, a := range s.objs {
		if o.GetUID() == a.GetUID() {
			delete(s.objs, i)
			break
		}
	}

	if oldLen == len(s.objs) {
		return fmt.Errorf("object not applied: %s", o.GetName())
	}

	err := s.k8s.Delete(s.ctx, o)
	if err != nil {
		return fmt.Errorf("kubernetes error: %w", err)
	}

	// Waits being deleted from kube
	key := types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}
	return retryForever(s.ctx, func() (bool, error) {
		err := s.k8s.Get(s.ctx, key, o)
		return !isNotFound(err), nil
	})
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

func getNamespacedName(keys ...string) (types.NamespacedName, error) {
	var name, namespace string
	switch len(keys) {
	case 1:
		name = keys[0]
		namespace = defaultNamespace
	case 2: //nolint:gomnd
		name = keys[0]
		namespace = keys[1]
	default:
		return types.NamespacedName{}, fmt.Errorf("provide name or/and namespace")
	}
	return types.NamespacedName{Name: name, Namespace: namespace}, nil
}
