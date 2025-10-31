//go:build suite

package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
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
	DestroyError() error
	Delete(o client.Object, exists func() error) error
}

var _ Session = &session{}

type session struct {
	k8s  client.Client
	ctx  context.Context
	objs map[string]client.Object
}

func NewSession(ctx context.Context, k8s client.Client) Session {
	s := &session{
		k8s:  k8s,
		ctx:  ctx,
		objs: make(map[string]client.Object),
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
	for _, o := range objects {
		s.objs[o.GetName()] = o
		if o.GetNamespace() == "" {
			// an empty namespace may not be set during creation
			o.SetNamespace(defaultNamespace)
		}
	}

	ctx, cancel := context.WithTimeout(s.ctx, createTimeout)
	defer cancel()

	var g errgroup.Group
	for _, o := range objects {
		obj := o
		g.Go(func() error {
			defer s.recover()

			obj.SetResourceVersion("")
			obj.SetManagedFields(nil)

			// use server-side apply like kubectl with force to resolve conflicts
			return s.k8s.Patch(ctx, obj, client.Apply, &client.PatchOptions{
				FieldManager: "kubectl",
			})
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

	// The obj is empty, it doesn't have its kind and name set yet.
	obj.SetName(key.Name)
	obj.SetNamespace(key.Namespace)

	return retryForever(ctx, fmt.Sprintf("verify %s is running", objKey(obj)), func() (bool, error) {
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

		return !controllers.IsReadyToUse(obj), nil
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

func (s *session) Destroy(t testingT) {
	assert.NoError(t, s.DestroyError())
}

// DestroyError deletes all applied resources.
// Tolerant to "not found" error, because resource may have been deleted manually
func (s *session) DestroyError() (err error) {
	log.Printf("SESSION DESTROY: Starting cleanup of %d resources", len(s.objs))
	for name := range s.objs {
		log.Printf("SESSION DESTROY: Will delete resource: %s", objKey(s.objs[name]))
	}

	var wg sync.WaitGroup
	wg.Add(len(s.objs))
	for n := range s.objs {
		go func(n string) {
			defer wg.Done()
			defer s.recover()
			errDel := s.delete(s.objs[n])
			if errDel != nil && !isNotFound(errDel) {
				err = multierror.Append(err, errDel)
			}
		}(n)
	}
	wg.Wait()
	return err
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
	if avngen.IsNotFound(err) {
		return nil
	}
	return err
}

// delete deletes object from kube, and makes sure it is not there anymore
// Removes from applied list to not delete object on Destroy()
func (s *session) delete(o client.Object) error {
	_, ok := s.objs[o.GetName()]
	if !ok {
		return fmt.Errorf("resource %q not applied", objKey(o))
	}

	log.Printf("SESSION DELETE: Deleting resource %s (kind: %s, namespace: %s)",
		objKey(o), o.GetObjectKind().GroupVersionKind().Kind, o.GetNamespace())

	// Delete operation doesn't share the context,
	// because it shouldn't leave artifacts
	ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
	defer cancel()

	err := s.k8s.Delete(ctx, o)
	if err != nil {
		log.Printf("SESSION DELETE: Failed to delete %s: %v", objKey(o), err)
		return fmt.Errorf("kubernetes error: %w", err)
	}

	log.Printf("SESSION DELETE: Successfully deleted %s", objKey(o))

	// Waits being deleted from kube
	key := types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}
	return retryForever(ctx, fmt.Sprintf("delete %s", objKey(o)), func() (bool, error) {
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
	b := make([]byte, randIDSize)

	randomBytes := make([]byte, randIDSize)

	_, err := rand.Read(randomBytes)
	if err != nil {
		panic("failed to read random: " + err.Error())
	}

	for i, randomByte := range randomBytes {
		b[i] = randIDChars[int(randomByte)%len(randIDChars)]
	}

	return string(b)
}

func randName[T ~string](name T) string {
	s := strings.ToLower(fmt.Sprintf("test-%s-%s", randID(), name))
	if len(s) > nameMaxSize {
		panic(fmt.Sprintf("invalid name, max length %d: %q", nameMaxSize, s))
	}
	return s
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
			// an empty namespace may not be set during creation
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

// getAnyType an empty client.Object doesn't have its kind.
// Returns the type name of the object.
func getAnyType(o any) string {
	t := reflect.TypeOf(o)
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	}
	return t.Name()
}

func objKey(o client.Object) string {
	ns := o.GetNamespace()
	if ns == "" {
		ns = defaultNamespace
	}
	return filepath.Join(getAnyType(o), ns, o.GetName())
}
