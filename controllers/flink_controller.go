// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=flinks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=flinks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=flinks/finalizers,verbs=get;create;update

func newFlinkReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.Flink] {
			return newServiceController(c, avnGen, flinkKind)
		},
		nil,
	)
}

// flinkKind is the serviceController definition for Flink.
var flinkKind = newServiceKind(serviceTypeFlink, flinkSecretData)

// flinkSecretData returns the prefixed connection secret keys.
func flinkSecretData(obj *v1alpha1.Flink, s *service.ServiceGetOut) map[string]string {
	stringData := map[string]string{
		"HOST":     s.ServiceUriParams["host"],
		"USER":     s.ServiceUriParams["user"],
		"PASSWORD": s.ServiceUriParams["password"],
		"URI":      s.ServiceUri,
		"HOSTS":    strings.Join(s.ConnectionInfo.Flink, ","),
	}

	return prefixKeys(getSecretPrefix(obj), stringData)
}
