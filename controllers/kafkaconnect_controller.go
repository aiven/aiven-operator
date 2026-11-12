// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	avngen "github.com/aiven/go-client-codegen"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects/finalizers,verbs=get;create;update

func newKafkaConnectReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.KafkaConnect] {
			return newServiceController(c, avnGen, kafkaConnectKind)
		},
		nil,
	)
}

// kafkaConnectKind is the serviceController definition for KafkaConnect.
var kafkaConnectKind = newServiceKind[*v1alpha1.KafkaConnect](serviceTypeKafkaConnect, nil,
	// No power-off backup check.
	withPreconditions(checkReadReplicaSource[*v1alpha1.KafkaConnect]),
)
