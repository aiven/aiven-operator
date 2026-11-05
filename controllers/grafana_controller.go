// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=grafanas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=grafanas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=grafanas/finalizers,verbs=get;create;update

func newGrafanaReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.Grafana] {
			return newServiceController(c, avnGen, grafanaKind)
		},
		nil,
	)
}

// grafanaKind is the serviceController definition for Grafana.
var grafanaKind = newServiceKind(serviceTypeGrafana, grafanaSecretData)

// grafanaSecretData returns the prefixed connection secret keys.
func grafanaSecretData(obj *v1alpha1.Grafana, s *service.ServiceGetOut) map[string]string {
	stringData := map[string]string{
		"HOST":     s.ServiceUriParams["host"],
		"PORT":     s.ServiceUriParams["port"],
		"USER":     s.ServiceUriParams["user"],
		"PASSWORD": s.ServiceUriParams["password"],
		"URI":      s.ServiceUri,
		"HOSTS":    strings.Join(s.ConnectionInfo.Grafana, ","),
	}

	return prefixKeys(getSecretPrefix(obj), stringData)
}
