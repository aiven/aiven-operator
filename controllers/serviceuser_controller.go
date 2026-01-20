// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/avast/retry-go"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func newServiceUserReconciler(c Controller) reconcilerType {
	return &Reconciler[*v1alpha1.ServiceUser]{
		Controller:              c,
		newAivenGeneratedClient: NewAivenGeneratedClient,
		newObj: func() *v1alpha1.ServiceUser {
			return &v1alpha1.ServiceUser{}
		},
		newController: func(avnGen avngen.Client) AivenController[*v1alpha1.ServiceUser] {
			return &ServiceUserController{
				Client: c.Client,
				avnGen: avnGen,
			}
		},
		newSecret: newSecret,
	}
}

//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers,verbs=update;get;list;watch;create;delete
//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers/finalizers,verbs=get;create;update

// ServiceUserController reconciles a ServiceUser object
type ServiceUserController struct {
	client.Client
	avnGen avngen.Client
}

func (r *ServiceUserController) Observe(ctx context.Context, user *v1alpha1.ServiceUser) (Observation, error) {
	details, err := r.fetchSecretDetails(ctx, user)
	if err != nil {
		// ServiceUser 404 means "user doesn't exist yet" and should trigger Create.
		// But service/project 404 is wrapped as errPreconditionNotMet in getServiceIfOperational
		// and must be treated as a soft precondition failure.
		if isNotFound(err) && !errors.Is(err, errPreconditionNotMet) {
			return Observation{ResourceExists: false}, nil
		}
		return Observation{}, err
	}

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: IsReadyToUse(user),
		SecretDetails:    details,
	}, nil
}

func (r *ServiceUserController) Create(ctx context.Context, user *v1alpha1.ServiceUser) (CreateResult, error) {
	password, err := GetPasswordFromSecret(ctx, r.Client, user)
	if err != nil {
		return CreateResult{}, err
	}

	u, err := r.avnGen.ServiceUserCreate(
		ctx,
		user.Spec.Project,
		user.Spec.ServiceName,
		&service.ServiceUserCreateIn{Username: user.Name},
	)
	if err != nil && !isAlreadyExists(err) {
		return CreateResult{}, fmt.Errorf("creating service user: %w", err)
	}

	if err := r.enforceExternalPassword(ctx, user, password); err != nil {
		return CreateResult{}, err
	}

	if u != nil {
		user.Status.Type = u.Type
	}

	meta.SetStatusCondition(&user.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")

	details, err := r.fetchSecretDetails(ctx, user)
	if err != nil {
		return CreateResult{}, fmt.Errorf("building connection details: %w", err)
	}

	return CreateResult{SecretDetails: details}, nil
}

func (r *ServiceUserController) Update(ctx context.Context, user *v1alpha1.ServiceUser) (UpdateResult, error) {
	password, err := GetPasswordFromSecret(ctx, r.Client, user)
	if err != nil {
		return UpdateResult{}, err
	}

	if err := r.enforceExternalPassword(ctx, user, password); err != nil {
		return UpdateResult{}, err
	}

	meta.SetStatusCondition(&user.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")

	details, err := r.fetchSecretDetails(ctx, user)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("building connection details: %w", err)
	}

	return UpdateResult{SecretDetails: details}, nil
}

func (r *ServiceUserController) Delete(ctx context.Context, user *v1alpha1.ServiceUser) error {
	// skip deletion for built-in users that cannot be deleted
	if isBuiltInUser(user.Name) {
		// built-in users like avnadmin cannot be deleted, this is expected behavior
		return nil
	}

	err := r.avnGen.ServiceUserDelete(ctx, user.Spec.Project, user.Spec.ServiceName, user.Name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting service user: %w", err)
	}

	return nil
}

func (r *ServiceUserController) enforceExternalPassword(ctx context.Context, user *v1alpha1.ServiceUser, password string) error {
	if password == "" {
		return nil
	}

	if _, err := r.avnGen.ServiceUserCredentialsModify(
		ctx,
		user.Spec.Project,
		user.Spec.ServiceName,
		user.Name,
		&service.ServiceUserCredentialsModifyIn{
			NewPassword: &password,
			Operation:   service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
		},
	); err != nil {
		return fmt.Errorf("modifying service user credentials: %w", err)
	}

	return nil
}

func (r *ServiceUserController) fetchSecretDetails(ctx context.Context, user *v1alpha1.ServiceUser) (SecretDetails, error) {
	svc, err := getServiceIfOperational(ctx, r.avnGen, user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return nil, err
	}

	// errEmptyPassword password is not received from the API:
	// 1. it was changed by TF but the API did not return it
	// 2. user has changed it in PG directly, so the API does not have it
	errEmptyPassword := errors.New("received empty password from the API")
	const (
		emptyPasswordRetryAttempts = 10
		emptyPasswordRetryDelay    = 5 * time.Second
	)

	// Retries empty password up to ~1m.
	// It should be enough to get the backend to a consistent state.
	// Though if user has changed the password in PG directly,
	// the API will never return the password.
	var u *service.ServiceUserGetOut
	err = retry.Do(
		func() error {
			u, err = r.avnGen.ServiceUserGet(ctx, user.Spec.Project, user.Spec.ServiceName, user.Name)
			if err == nil && u.Password == "" {
				err = errEmptyPassword
			}
			return err
		},
		retry.Context(ctx),
		// Retries errEmptyPassword only.
		// The rest is retried by the client itself and the outer controller.
		retry.RetryIf(func(err error) bool {
			return errors.Is(err, errEmptyPassword)
		}),
		// ≈1m total wait time
		retry.Attempts(emptyPasswordRetryAttempts),
		retry.Delay(emptyPasswordRetryDelay),
		// retry.Do returns a custom list of errors.
		// Outer controller must be able to detect error types like "server error".
		retry.LastErrorOnly(true),
	)
	if err != nil {
		return nil, err
	}

	idx := slices.IndexFunc(svc.Components, func(c service.ComponentOut) bool {
		return c.Component == svc.ServiceType || (svc.ServiceType == "alloydbomni" && c.Component == "pg")
	})
	if idx < 0 {
		return nil, fmt.Errorf("service component %q not found", svc.ServiceType)
	}
	component := &svc.Components[idx]

	caCert, err := r.avnGen.ProjectKmsGetCA(ctx, user.Spec.Project)
	if err != nil {
		return nil, fmt.Errorf("aiven client error %w", err)
	}

	return buildSecretDetailsFromComponent(component, user, u, caCert), nil
}

func buildSecretDetailsFromComponent(
	component *service.ComponentOut,
	user *v1alpha1.ServiceUser,
	u *service.ServiceUserGetOut,
	caCert string,
) SecretDetails {
	prefix := getSecretPrefix(user)
	details := SecretDetails{
		prefix + "HOST":        component.Host,
		prefix + "PORT":        fmt.Sprintf("%d", component.Port),
		prefix + "USERNAME":    u.Username,
		prefix + "PASSWORD":    u.Password,
		prefix + "ACCESS_CERT": fromAnyPointer(u.AccessCert),
		prefix + "ACCESS_KEY":  fromAnyPointer(u.AccessKey),
		prefix + "CA_CERT":     caCert,
	}

	return details
}
