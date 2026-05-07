// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafka"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaquotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaquotas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaquotas/finalizers,verbs=get;create;update

// KafkaQuotaController reconciles a KafkaQuota object.
type KafkaQuotaController struct {
	client.Client
	avnGen avngen.Client
}

func newKafkaQuotaReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.KafkaQuota] {
			return &KafkaQuotaController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *KafkaQuotaController) Observe(ctx context.Context, q *v1alpha1.KafkaQuota) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, q.Spec.Project, q.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	got, err := r.avnGen.ServiceKafkaQuotaDescribe(ctx, q.Spec.Project, q.Spec.ServiceName, quotaSelector(q)...)
	switch {
	case isNotFound(err):
		return Observation{ResourceExists: false}, nil
	case err != nil:
		return Observation{}, fmt.Errorf("describing Kafka quota: %w", err)
	}

	// The describe endpoint returns an empty payload when no matching quota exists.
	// Treat that as "not found" so the reconciler issues a Create.
	if got.User == nil && got.ClientId == nil {
		return Observation{ResourceExists: false}, nil
	}

	meta.SetStatusCondition(&q.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&q.ObjectMeta, instanceIsRunningAnnotation, "true")

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: hasLatestGeneration(q) && quotaMatchesSpec(got, q),
	}, nil
}

func (r *KafkaQuotaController) Create(ctx context.Context, q *v1alpha1.KafkaQuota) (CreateResult, error) {
	if err := r.applyQuota(ctx, q); err != nil {
		return CreateResult{}, err
	}

	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(&q.Status.Conditions, getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&q.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))

	return CreateResult{}, nil
}

func (r *KafkaQuotaController) Update(ctx context.Context, q *v1alpha1.KafkaQuota) (UpdateResult, error) {
	delete(q.GetAnnotations(), instanceIsRunningAnnotation)
	if err := r.applyQuota(ctx, q); err != nil {
		return UpdateResult{}, err
	}

	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(&q.Status.Conditions, getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&q.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))

	return UpdateResult{}, nil
}

func (r *KafkaQuotaController) Delete(ctx context.Context, q *v1alpha1.KafkaQuota) error {
	err := r.avnGen.ServiceKafkaQuotaDelete(ctx, q.Spec.Project, q.Spec.ServiceName, quotaSelector(q)...)
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

// applyQuota upserts a Kafka quota. The API's endpoint is idempotent.
func (r *KafkaQuotaController) applyQuota(ctx context.Context, q *v1alpha1.KafkaQuota) error {
	in := &kafka.ServiceKafkaQuotaCreateIn{
		User:              NilIfZero(q.Spec.User),
		ClientId:          NilIfZero(q.Spec.ClientID),
		ConsumerByteRate:  int64ToFloatPtr(q.Spec.ConsumerByteRate),
		ProducerByteRate:  int64ToFloatPtr(q.Spec.ProducerByteRate),
		RequestPercentage: q.Spec.RequestPercentage,
	}

	if err := r.avnGen.ServiceKafkaQuotaCreate(ctx, q.Spec.Project, q.Spec.ServiceName, in); err != nil {
		return fmt.Errorf("cannot create or update Kafka quota: %w", err)
	}

	return nil
}

// quotaSelector builds the query parameters for describe/delete based on which identifiers are set.
func quotaSelector(q *v1alpha1.KafkaQuota) [][2]string {
	var params [][2]string
	if q.Spec.User != "" {
		params = append(params, kafka.ServiceKafkaQuotaDescribeUser(q.Spec.User))
	}
	if q.Spec.ClientID != "" {
		params = append(params, kafka.ServiceKafkaQuotaDescribeClientId(q.Spec.ClientID))
	}
	return params
}

// quotaMatchesSpec returns true if the remote quota values match the desired spec values.
func quotaMatchesSpec(remote *kafka.ServiceKafkaQuotaDescribeOut, q *v1alpha1.KafkaQuota) bool {
	return cmp.Equal(remote.ConsumerByteRate, int64ToFloatPtr(q.Spec.ConsumerByteRate)) &&
		cmp.Equal(remote.ProducerByteRate, int64ToFloatPtr(q.Spec.ProducerByteRate)) &&
		cmp.Equal(remote.RequestPercentage, q.Spec.RequestPercentage)
}

// int64ToFloatPtr converts the byte-rate spec fields to the float64 the Aiven API expects.
// The values are capped at 2^30 by validation markers, well within float64's exact-integer range (2^53).
func int64ToFloatPtr(v *int64) *float64 {
	if v == nil {
		return nil
	}
	return new(float64(*v))
}
