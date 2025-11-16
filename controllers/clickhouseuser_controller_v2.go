// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers/finalizers,verbs=get;create;update

// ClickhouseUserControllerV2 reconciles a ClickhouseUser object
type ClickhouseUserControllerV2 struct {
	client.Client
	avnGen avngen.Client
}

func newClickhouseUserReconcilerV2(c Controller) reconcilerType {
	return &Reconciler[*v1alpha1.ClickhouseUser]{
		Controller:              c,
		newAivenGeneratedClient: NewAivenGeneratedClient,
		newObj: func() *v1alpha1.ClickhouseUser {
			return &v1alpha1.ClickhouseUser{}
		},
		newController: func(avnGen avngen.Client) AivenController[*v1alpha1.ClickhouseUser] {
			return &ClickhouseUserControllerV2{
				Client: c.Client,
				avnGen: avnGen,
			}
		},
	}
}

func (r *ClickhouseUserControllerV2) Observe(ctx context.Context, user *v1alpha1.ClickhouseUser) (Observation, error) {
	obs := Observation{
		ResourceExists:   false,
		ResourceUpToDate: false,
		SecretDetails:    nil,
	}

	if err := checkServiceIsOperational2(ctx, r.avnGen, user.Spec.Project, user.Spec.ServiceName); err != nil {
		return obs, err
	}

	list, err := r.avnGen.ServiceClickHouseUserList(ctx, user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return obs, err
	}
	for _, u := range list {
		if u.Name == user.GetUsername() {
			obs.ResourceExists = true
			user.Status.UUID = u.Uuid
			break
		}
	}

	if obs.ResourceExists {
		// TODO: extend the logic with more checks if needed
		obs.ResourceUpToDate = IsReadyToUse(user)
	}

	return obs, nil
}

// Create implements AivenClient for ClickhouseUser.
func (r *ClickhouseUserControllerV2) Create(ctx context.Context, user *v1alpha1.ClickhouseUser) (CreateResult, error) {
	logr.FromContextOrDiscard(ctx).Info("generation wasn't processed, creation or updating instance on aiven side")

	// Validates the secret password if it exists
	_, err := GetPasswordFromSecret(ctx, r.Client, user)
	if err != nil {
		return CreateResult{}, fmt.Errorf("failed to get password from secret: %w", err)
	}

	if user.Status.UUID == "" {
		req := clickhouse.ServiceClickHouseUserCreateIn{
			Name: user.GetUsername(),
		}
		rsp, err := r.avnGen.ServiceClickHouseUserCreate(ctx, user.Spec.Project, user.Spec.ServiceName, &req)
		if err != nil {
			logr.FromContextOrDiscard(ctx).Info(
				"unable to create or update instance, retrying",
				"kind", user.GetObjectKind().GroupVersionKind().Kind,
				"namespace", user.GetNamespace(),
				"name", user.GetName(),
				"error", err,
			)
			return CreateResult{}, err
		}
		user.Status.UUID = rsp.Uuid
	}

	logr.FromContextOrDiscard(ctx).Info(
		"processed instance, updating annotations",
		"generation", user.GetGeneration(),
		"annotations", user.GetAnnotations(),
	)

	return CreateResult{}, nil
}

// Update implements AivenClient for ClickhouseUser.
func (r *ClickhouseUserControllerV2) Update(ctx context.Context, user *v1alpha1.ClickhouseUser) (UpdateResult, error) {
	logr.FromContextOrDiscard(ctx).Info("checking if instance is ready")

	s, err := r.avnGen.ServiceGet(ctx, user.Spec.Project, user.Spec.ServiceName, service.ServiceGetIncludeSecrets(true))
	if err != nil {
		return UpdateResult{}, err
	}

	// User can set password in the secret
	secretPassword, err := GetPasswordFromSecret(ctx, r.Client, user)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("failed to get password from secret: %w", err)
	}

	// By design, this handler can't create secret in createOrUpdate method, while the password is returned on create only.
	// The only way to have a secret here is to reset it manually
	req := clickhouse.ServiceClickHousePasswordResetIn{}
	if secretPassword != "" {
		req.Password = &secretPassword
	}

	password, err := r.avnGen.ServiceClickHousePasswordReset(ctx, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID, &req)
	if err != nil {
		return UpdateResult{}, err
	}

	prefix := getSecretPrefix(user)
	stringData := map[string]string{
		prefix + "HOST":     s.ServiceUriParams["host"],
		prefix + "PORT":     s.ServiceUriParams["port"],
		prefix + "PASSWORD": password,
		prefix + "USERNAME": user.GetUsername(),
		// todo: remove in future releases
		"HOST":     s.ServiceUriParams["host"],
		"PORT":     s.ServiceUriParams["port"],
		"PASSWORD": password,
		"USERNAME": user.GetUsername(),
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")

	result := UpdateResult{
		SecretDetails: make(map[string][]byte, len(stringData)),
	}
	for k, v := range stringData {
		result.SecretDetails[k] = []byte(v)
	}

	return result, nil
}

// Delete implements AivenClient for ClickhouseUser.
// It mirrors the current delete behaviour used in reconcile2.
func (r *ClickhouseUserControllerV2) Delete(ctx context.Context, user *v1alpha1.ClickhouseUser) error {
	// Not processed yet
	if user.Status.UUID == "" {
		return nil
	}

	// skip deletion for built-in users that cannot be deleted
	if isBuiltInUser(user.Name) {
		// built-in users like 'default' cannot be deleted, this is expected behavior
		// we consider this a successful deletion since we can't and shouldn't delete built-in users
		return nil
	}

	err := r.avnGen.ServiceClickHouseUserDelete(ctx, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID)
	if !isNotFound(err) {
		return err
	}

	return nil
}
