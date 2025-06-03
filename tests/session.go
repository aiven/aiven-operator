package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

const (
	retryInterval      = time.Second * 10
	createTimeout      = time.Second * 15
	waitRunningTimeout = time.Minute * 20
	deleteTimeout      = time.Minute * 5
	yamlBufferSize     = 100
	defaultNamespace   = "default"
)

type Session interface {
	Apply(src string) error
	ApplyObjects(objects ...client.Object) error
	GetRunning(obj client.Object, keys ...string) error
	GetSecret(keys ...string) (*corev1.Secret, error)
	Destroy(t testingT)
	Delete(o client.Object, exists func() error) error
}

var _ Session = &session{}

type session struct {
	k8s     client.Client
	ctx     context.Context
	objs    map[string]client.Object
	project string
}

func NewSession(ctx context.Context, k8s client.Client, project string) Session {
	s := &session{
		k8s:     k8s,
		ctx:     ctx,
		objs:    make(map[string]client.Object),
		project: project,
	}
	return s
}

// Apply parses and applies Kubernetes resources defined in YAML format
func (s *session) Apply(src string) error {
	objs, err := parseObjs(src)
	if err != nil {
		return err
	}

	// Convert map to slice for ApplyObjects
	objSlice := make([]client.Object, 0, len(objs))
	for _, o := range objs {
		objSlice = append(objSlice, o)
	}

	return s.ApplyObjects(objSlice...)
}

// ApplyObjects applies multiple Kubernetes objects
func (s *session) ApplyObjects(objects ...client.Object) error {
	// Store all objects being applied
	for _, o := range objects {
		s.objs[o.GetName()] = o
	}

	ctx, cancel := context.WithTimeout(s.ctx, createTimeout)
	defer cancel()

	var g errgroup.Group
	for _, o := range objects {
		// Create a local variable to avoid closure issues
		obj := o
		g.Go(func() error {
			defer s.recover()

			// Clear resource version before attempting to create
			// This is important to avoid the "resourceVersion should not be set on objects to be created" error
			obj.SetResourceVersion("")
			err := s.k8s.Create(ctx, obj)
			if alreadyExists(err) {
				c := obj.DeepCopyObject().(client.Object)
				key := types.NamespacedName{
					Name:      c.GetName(),
					Namespace: c.GetNamespace(),
				}
				err = s.k8s.Get(ctx, key, c)
				if err != nil {
					return err
				}
				obj.SetResourceVersion(c.GetResourceVersion())
				return s.k8s.Update(ctx, obj)
			}
			return err
		})
	}
	return g.Wait()
}

// GetRunning waits until the specified object is in "running" state
// and returns an error if the object has error conditions
func (s *session) GetRunning(obj client.Object, keys ...string) error {
	key, err := getNamespacedName(keys...)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(s.ctx, waitRunningTimeout)
	defer cancel()

	return retryForever(ctx, fmt.Sprintf("verify %s is running", key), func() (bool, error) {
		err := s.k8s.Get(ctx, key, obj)
		if err != nil {
			// The error is quite verbose
			log.Printf("waiting resource %q running: %s", key, err)

			// Do not retry kube errors
			return isNotFound(err), err
		}

		// Exits on condition errors
		if o, ok := obj.(v1alpha1.AivenManagedObject); ok {
			for _, c := range *o.Conditions() {
				if c.Type == controllers.ConditionTypeError {
					// Sometimes it is OK that API returns "not ready" condition.
					// Retries "try again later" errors.
					return strings.Contains(c.Message, "try again later"), errors.New(c.Message)
				}
			}
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

type testingT interface {
	Errorf(format string, args ...any)
}

// Destroy deletes all applied resources.
// Tolerant to "not found" error,
// because resource may have been deleted manually
func (s *session) Destroy(t testingT) {
	if err := recover(); err != nil {
		t.Errorf("panicked, deleting resources: %s\n%s", err, debug.Stack())
	}

	var wg sync.WaitGroup
	wg.Add(len(s.objs))
	for n := range s.objs {
		go func(n string) {
			defer wg.Done()
			defer s.recover()
			err := s.delete(s.objs[n])
			if err != nil && !isNotFound(err) {
				t.Errorf("failed to delete %q: %s", n, err)
			}
		}(n)
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
	if aiven.IsNotFound(err) || avngen.IsNotFound(err) {
		return nil
	}
	return err
}

// delete deletes object from kube, and makes sure it is not there anymore
// Removes from applied list to not delete object on Destroy()
func (s *session) delete(o client.Object) error {
	_, ok := s.objs[o.GetName()]
	if !ok {
		return fmt.Errorf("resource %q not applied", o.GetName())
	}

	// Delete operation doesn't share the context,
	// because it shouldn't leave artifacts
	ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
	defer cancel()

	err := s.k8s.Delete(ctx, o)
	if err != nil {
		return fmt.Errorf("kubernetes error: %w", err)
	}

	// Waits being deleted from kube
	key := types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}
	return retryForever(ctx, fmt.Sprintf("delete %s", o.GetName()), func() (bool, error) {
		err := s.k8s.Get(ctx, key, o)
		return !isNotFound(err), nil
	})
}

func (s *session) recover() {
	err := recover()
	if err != nil {
		log.Printf("panicked: %s", err)
		log.Printf("stacktrace: \n%s", string(debug.Stack()))
	}
}

func retryForever(ctx context.Context, operation string, f func() (bool, error)) (err error) {
	retry := false
	log.Printf("Operation %q started\n", operation)

outer:
	for {
		select {
		case <-ctx.Done():
			err = multierror.Append(err, ctx.Err())
			break outer
		case <-time.After(retryInterval):
			retry, err = f()
			if err != nil {
				err = multierror.Append(err, err)
			}

			if !retry {
				break outer
			}
		}
	}

	if err != nil {
		log.Printf("Operation %q failed: %s\n", operation, err)
		return err
	}

	log.Printf("Operation %q succeeded\n", operation)
	return nil
}

const (
	randIDSize = 7
	// randIDChars Aiven allowed chars for "names"
	randIDChars = "0123456789abcdefghijklmnopqrstuvwxyz"
	nameMaxSize = 255
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
	s := fmt.Sprintf("test-%s-%s", randID(), name)
	if len(s) > nameMaxSize {
		panic(fmt.Sprintf("invalid name, max length %d: %q", nameMaxSize, s))
	}
	return s
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

func alreadyExists(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
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
	case 2: //nolint:mnd
		name = keys[0]
		namespace = keys[1]
	default:
		return types.NamespacedName{}, fmt.Errorf("provide name or/and namespace")
	}
	return types.NamespacedName{Name: name, Namespace: namespace}, nil
}

func parseObjs(src string) (map[string]client.Object, error) {
	objs := make(map[string]client.Object)

	// Creds: https://gist.github.com/pytimer/0ad436972a073bb37b8b6b8b474520fc
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(src)), yamlBufferSize)
	for {
		var rawExt runtime.RawExtension
		err := decoder.Decode(&rawExt)
		if errors.Is(err, io.EOF) {
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
			return nil, fmt.Errorf("failed to serialize raw object %w", err)
		}

		uMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(uObj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to unstructured map %w", err)
		}

		o := unstructured.Unstructured{Object: uMap}
		if o.GetNamespace() == "" {
			o.SetNamespace(defaultNamespace)
		}

		n := o.GetName()
		if _, ok := objs[n]; ok {
			return nil, fmt.Errorf("resource name %q is not unique", n)
		}
		objs[n] = &o
	}
	return objs, nil
}
