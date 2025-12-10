// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"slices"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
	"github.com/aiven/go-client-codegen/handler/service"
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
		newSecret: newSecret,
	}
}

func (r *ClickhouseUserControllerV2) Observe(ctx context.Context, user *v1alpha1.ClickhouseUser) (Observation, error) {
	svc, err := getServiceIfOperational(ctx, r.avnGen, user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return Observation{}, err
	}

	list, err := r.avnGen.ServiceClickHouseUserList(ctx, user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return Observation{}, fmt.Errorf("listing Clickhouse users: %w", err)
	}

	idx := slices.IndexFunc(list, func(u clickhouse.UserOut) bool {
		return u.Name == user.GetUsername()
	})
	if idx < 0 {
		return Observation{ResourceExists: false}, nil
	}

	u := list[idx]
	user.Status.UUID = u.Uuid

	var password string
	if user.Spec.ConnInfoSecretSource != nil {
		// External mode: the password from ConnInfoSecretSource fully defines the desired password and should be reflected in the connection secret
		var err error
		password, err = GetPasswordFromSecret(ctx, r.Client, user)
		if err != nil {
			return Observation{}, err
		}
	} else if u.Password != nil && *u.Password != "" {
		// Operator-managed mode: when ConnInfoSecretSource is not set, we treat the password returned by Aiven API (if any)
		// as the source of truth for the connection secret. If the API does not expose the password (e.g. it was changed directly in ClickHouse),
		// we leave password empty and do not touch password keys in the Secret.
		password = *u.Password
	}

	secretDetails := buildConnectionDetailsFromService(svc, user, password)

	return Observation{
		ResourceExists: true,
		// TODO: extend the logic with more checks if needed
		ResourceUpToDate: IsReadyToUse(user),
		SecretDetails:    secretDetails,
	}, nil
}

func (r *ClickhouseUserControllerV2) Create(ctx context.Context, user *v1alpha1.ClickhouseUser) (CreateResult, error) {
	password, err := GetPasswordFromSecret(ctx, r.Client, user)
	if err != nil {
		return CreateResult{}, err
	}

	req := clickhouse.ServiceClickHouseUserCreateIn{
		Name:     user.GetUsername(),
		Password: NilIfZero(password),
	}

	resp, err := r.avnGen.ServiceClickHouseUserCreate(ctx, user.Spec.Project, user.Spec.ServiceName, &req)
	if err != nil {
		return CreateResult{}, fmt.Errorf("creating Clickhouse user: %w", err)
	}
	user.Status.UUID = resp.Uuid

	if password == "" {
		if resp.Password != nil && *resp.Password != "" {
			password = *resp.Password
		} else {
			resetReq := clickhouse.ServiceClickHousePasswordResetIn{}
			password, err = r.avnGen.ServiceClickHousePasswordReset(ctx, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID, &resetReq)
			if err != nil {
				return CreateResult{}, fmt.Errorf("resetting Clickhouse user password: %w", err)
			}
		}
	}

	meta.SetStatusCondition(&user.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")

	secretDetails, err := r.buildConnectionDetails(ctx, user, password)
	if err != nil {
		return CreateResult{}, fmt.Errorf("building connection details: %w", err)
	}

	return CreateResult{SecretDetails: secretDetails}, nil
}

func (r *ClickhouseUserControllerV2) Update(ctx context.Context, user *v1alpha1.ClickhouseUser) (UpdateResult, error) {
	desiredPassword, err := GetPasswordFromSecret(ctx, r.Client, user)
	if err != nil {
		return UpdateResult{}, err
	}

	// External mode: when a ConnInfoSecretSource is configured, we actively enforce the password from that source via PasswordReset.
	//
	// Operator-managed mode (no ConnInfoSecretSource): we do not modify the ClickHouse password on updates.
	// Instead, we keep publishing connection details without touching the password in the connection Secret.
	password := ""
	if desiredPassword != "" {
		req := clickhouse.ServiceClickHousePasswordResetIn{
			Password: &desiredPassword,
		}

		password, err = r.avnGen.ServiceClickHousePasswordReset(ctx, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID, &req)
		if err != nil {
			return UpdateResult{}, fmt.Errorf("resetting Clickhouse user password: %w", err)
		}
	}

	meta.SetStatusCondition(&user.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")

	secretDetails, err := r.buildConnectionDetails(ctx, user, password)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("building connection details: %w", err)
	}

	return UpdateResult{SecretDetails: secretDetails}, nil
}

func (r *ClickhouseUserControllerV2) Delete(ctx context.Context, user *v1alpha1.ClickhouseUser) error {
	if user.Status.UUID == "" {
		return nil
	}

	if isBuiltInUser(user.Name) {
		return nil
	}

	err := r.avnGen.ServiceClickHouseUserDelete(ctx, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting Clickhouse user: %w", err)
	}

	return nil
}

func (r *ClickhouseUserControllerV2) buildConnectionDetails(ctx context.Context, user *v1alpha1.ClickhouseUser, password string) (SecretDetails, error) {
	s, err := r.avnGen.ServiceGet(ctx, user.Spec.Project, user.Spec.ServiceName, service.ServiceGetIncludeSecrets(true))
	if err != nil {
		return nil, fmt.Errorf("getting service details: %w", err)
	}

	details := buildConnectionDetailsFromService(s, user, password)
	return details, nil
}

func buildConnectionDetailsFromService(s *service.ServiceGetOut, user *v1alpha1.ClickhouseUser, password string) SecretDetails {
	prefix := getSecretPrefix(user)

	details := SecretDetails{
		prefix + "HOST":     s.ServiceUriParams["host"],
		prefix + "PORT":     s.ServiceUriParams["port"],
		prefix + "USERNAME": user.GetUsername(),
		// todo: remove in future releases
		"HOST":     s.ServiceUriParams["host"],
		"PORT":     s.ServiceUriParams["port"],
		"USERNAME": user.GetUsername(),
	}

	if password != "" {
		details[prefix+"PASSWORD"] = password
		details["PASSWORD"] = password
	}

	return details
}
