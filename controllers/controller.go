package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type Controller struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	AivenClient *aiven.Client
}

// InitAivenClient retrieves an Aiven client
func (c *Controller) InitAivenClient(req ctrl.Request, ctx context.Context, log logr.Logger) error {
	if c.AivenClient != nil {
		return nil
	}

	var token string

	// Check if aiven-token secret exists
	secret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: "aiven-token", Namespace: req.Namespace}, secret)
	if err != nil {
		log.Error(err, "aiven-token secret is missing, required by the Aiven client")
		return err
	}

	if v, ok := secret.Data["token"]; ok {
		token = string(v)
	} else {
		return fmt.Errorf("cannot initialize Aiven client, kubernetes secret has no `token` key")
	}

	c.AivenClient, err = aiven.NewTokenClient(token, "k8s-operator/")
	if err != nil {
		return err
	}

	return nil
}

// contains checks if string slice contains an element
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// UserConfigurationToAPI converts UserConfiguration options structure
// to Aiven API compatible map[string]interface{}
func UserConfigurationToAPI(c interface{}) interface{} {
	result := make(map[string]interface{})

	v := reflect.ValueOf(c)

	// if its a pointer, resolve its value
	if v.Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}

	if v.Kind() != reflect.Struct {
		switch v.Kind() {
		case reflect.Int64:
			return *c.(*int64)
		case reflect.Bool:
			return *c.(*bool)
		default:
			return c
		}
	}

	structType := v.Type()

	// convert UserConfig structure to a map
	for i := 0; i < structType.NumField(); i++ {
		name := strings.ReplaceAll(structType.Field(i).Tag.Get("json"), ",omitempty", "")

		if structType.Kind() == reflect.Struct {
			result[name] = UserConfigurationToAPI(v.Field(i).Interface())
		} else {
			result[name] = v.Elem().Field(i).Interface()
		}
	}

	// remove all the nil and empty map data
	for key, val := range result {
		if val == nil || isNil(val) || val == "" {
			delete(result, key)
		}

		if reflect.TypeOf(val).Kind() == reflect.Map {
			if len(val.(map[string]interface{})) == 0 {
				delete(result, key)
			}
		}
	}

	return result
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}

func toOptionalStringPointer(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

func stringPointerToString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
