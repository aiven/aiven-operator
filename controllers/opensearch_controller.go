// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=opensearches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=opensearches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=opensearches/finalizers,verbs=get;create;update

func newOpenSearchReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.OpenSearch] {
			return newServiceController(c, avnGen, openSearchKind)
		},
		nil,
	)
}

// openSearchKind is the serviceController definition for OpenSearch.
var openSearchKind = newServiceKind(serviceTypeOpenSearch, openSearchSecretData)

// openSearchSecretData returns the prefixed connection secret keys.
func openSearchSecretData(obj *v1alpha1.OpenSearch, s *service.ServiceGetOut) map[string]string {
	prefix := getSecretPrefix(obj)
	stringData := map[string]string{
		prefix + "URI":      s.ServiceUri,
		prefix + "HOST":     s.ServiceUriParams["host"],
		prefix + "PASSWORD": s.ServiceUriParams["password"],
		prefix + "PORT":     s.ServiceUriParams["port"],
		prefix + "USER":     s.ServiceUriParams["user"],
		// todo: remove in future releases
		"HOST":     s.ServiceUriParams["host"],
		"PASSWORD": s.ServiceUriParams["password"],
		"PORT":     s.ServiceUriParams["port"],
		"USER":     s.ServiceUriParams["user"],
	}

	return stringData
}
