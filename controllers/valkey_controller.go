// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=valkeys,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=valkeys/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=valkeys/finalizers,verbs=get;create;update

func newValkeyReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.Valkey] {
			return newServiceController(c, avnGen, valkeyKind)
		},
		nil,
	)
}

// valkeyKind is the serviceController definition for Valkey.
var valkeyKind = newServiceKind(serviceTypeValkey, valkeySecretData)

// valkeySecretData returns the prefixed connection secret keys.
func valkeySecretData(obj *v1alpha1.Valkey, s *service.ServiceGetOut) map[string]string {
	prefix := getSecretPrefix(obj)
	stringData := map[string]string{
		prefix + "HOST":     s.ServiceUriParams["host"],
		prefix + "PASSWORD": s.ServiceUriParams["password"],
		prefix + "PORT":     s.ServiceUriParams["port"],
		prefix + "SSL":      s.ServiceUriParams["ssl"],
		prefix + "USER":     s.ServiceUriParams["user"],
	}

	return stringData
}
