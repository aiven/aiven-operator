// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/avast/retry-go"
	"github.com/go-logr/logr"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
)

func newServiceUserReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.ServiceUser] {
			return &ServiceUserController{
				Client: c.Client,
				avnGen: avnGen,
			}
		},
		nil,
	)
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
	u, details, err := r.fetchUser(ctx, user, false)
	if err != nil {
		// ServiceUser 404 means "user doesn't exist yet" and should trigger Create.
		// But service/project 404 is wrapped as errPreconditionNotMet in getServiceIfOperational
		// and must be treated as a soft precondition failure.
		if isNotFound(err) && !errors.Is(err, errPreconditionNotMet) {
			return Observation{ResourceExists: false}, nil
		}
		return Observation{}, err
	}

	// Empty password with a source secret: Update so the declared value gets pushed back to Aiven.
	// Empty without a source: nothing to enforce, user may change it bypassing the API.
	if u.Password == "" && user.Spec.ConnInfoSecretSource != nil {
		return Observation{ResourceExists: true, ResourceUpToDate: false, SecretDetails: details}, nil
	}

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: IsReadyToUse(user) && accessControlMatches(user.Spec.AccessControl, u.AccessControl),
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
		&service.ServiceUserCreateIn{
			Username:      user.Name,
			AccessControl: buildServiceUserAccessControlIn(user.Spec.AccessControl),
		},
	)
	if err != nil {
		return CreateResult{}, fmt.Errorf("creating service user: %w", err)
	}

	if _, err := r.setAivenPasswordIfProvided(ctx, user, password); err != nil {
		return CreateResult{}, err
	}

	if u != nil {
		user.Status.Type = u.Type
	}

	meta.SetStatusCondition(&user.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")

	// Retry to absorb the potential API lag.
	_, details, err := r.fetchUser(ctx, user, true)
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

	if ac := buildServiceUserAccessControlIn(user.Spec.AccessControl); ac != nil {
		if _, err := r.avnGen.ServiceUserCredentialsModify(
			ctx,
			user.Spec.Project,
			user.Spec.ServiceName,
			user.Name,
			&service.ServiceUserCredentialsModifyIn{
				AccessControl: ac,
				Operation:     service.ServiceUserCredentialsModifyOperationTypeSetAccessControl,
			},
		); err != nil {
			return UpdateResult{}, fmt.Errorf("modifying service user access control: %w", err)
		}
	} else {
		logr.FromContextOrDiscard(ctx).V(1).Info("skipping access control update since it isn't provided in the spec")
	}

	wrotePassword, err := r.setAivenPasswordIfProvided(ctx, user, password)
	if err != nil {
		return UpdateResult{}, err
	}

	meta.SetStatusCondition(&user.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")

	// Retry empty-password only when a password was actually updated during this cycle.
	_, details, err := r.fetchUser(ctx, user, wrotePassword)
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

// setAivenPasswordIfProvided pushes the password to Aiven if non-empty.
// Returns true when the password was actually updated.
func (r *ServiceUserController) setAivenPasswordIfProvided(ctx context.Context, user *v1alpha1.ServiceUser, password string) (bool, error) {
	if password == "" {
		return false, nil
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
		return false, fmt.Errorf("modifying service user credentials: %w", err)
	}

	return true, nil
}

// fetchUser gets the ServiceUser from Aiven and builds the secret details.
//
// retryEmptyPassword absorbs the eventual-consistency window where
// ServiceUserGet returns an empty password right after the password change.
func (r *ServiceUserController) fetchUser(
	ctx context.Context,
	user *v1alpha1.ServiceUser,
	retryEmptyPassword bool,
) (*service.ServiceUserGetOut, SecretDetails, error) {
	svc, err := getServiceIfOperational(ctx, r.avnGen, user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return nil, nil, err
	}

	const (
		emptyPasswordExtraRetries = 2
		emptyPasswordRetryDelay   = 5 * time.Second
		notFoundRetryAttempts     = 5
		notFoundRetryDelay        = 1 * time.Second
	)

	running := hasIsRunningAnnotation(user)

	getUser := func() (*service.ServiceUserGetOut, error) {
		attempts := uint(notFoundRetryAttempts)
		if !running {
			attempts = 1
		}

		var u *service.ServiceUserGetOut
		// ServiceUsers that are already marked running also retry transient 404s
		// for a short time. Aiven may briefly report the user as missing immediately
		// after create or update, even though a follow-up get returns it.
		// Treating that 404 as "absent" too early pushes reconcile down the Create -> 409 path.
		if err := retry.Do(
			func() error {
				u, err = r.avnGen.ServiceUserGet(ctx, user.Spec.Project, user.Spec.ServiceName, user.Name)
				return err
			},
			retry.Context(ctx),
			retry.RetryIf(isNotFound),
			retry.Attempts(attempts),
			retry.Delay(notFoundRetryDelay),
			retry.LastErrorOnly(true),
		); err != nil {
			return nil, fmt.Errorf("getting service user: %w", err)
		}

		return u, nil
	}

	u, err := getUser()
	if err != nil {
		return nil, nil, err
	}

	if retryEmptyPassword && u.Password == "" {
		errEmptyPassword := errors.New("received empty password from the API")
		err = retry.Do(
			func() error {
				u, err = getUser()
				if err != nil {
					return err
				}
				if u.Password == "" {
					return errEmptyPassword
				}
				return nil
			},
			retry.Context(ctx),
			retry.RetryIf(func(err error) bool {
				return errors.Is(err, errEmptyPassword)
			}),
			retry.Attempts(emptyPasswordExtraRetries),
			retry.Delay(emptyPasswordRetryDelay),
			retry.LastErrorOnly(true),
		)
		// Real API errors propagate. Exhausted empty-password retries returns as precondition.
		switch {
		case errors.Is(err, errEmptyPassword):
			return nil, nil, fmt.Errorf("%w: ServiceUser password not yet available from API", errPreconditionNotMet)
		case err != nil:
			return nil, nil, err
		}
	}

	idx := slices.IndexFunc(svc.Components, func(c service.ComponentOut) bool {
		return c.Component == svc.ServiceType
	})
	if idx < 0 {
		return nil, nil, fmt.Errorf("service component %q not found", svc.ServiceType)
	}
	component := &svc.Components[idx]

	caCert, err := r.avnGen.ProjectKmsGetCA(ctx, user.Spec.Project)
	if err != nil {
		return nil, nil, fmt.Errorf("aiven client error %w", err)
	}

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

	if svc.ServiceType != string(serviceTypeKafka) {
		return u, details, nil
	}

	addKafkaEndpointDetails(details, svc.Components, prefix)

	var kafkaConfig struct {
		KafkaAuthenticationMethods *kafkauserconfig.KafkaAuthenticationMethods `json:"kafka_authentication_methods,omitempty"`
		SchemaRegistry             *bool                                       `json:"schema_registry,omitempty"`
	}
	if err := decodeMapInto(svc.UserConfig, &kafkaConfig); err != nil {
		logr.FromContextOrDiscard(ctx).Error(err, "unable to decode Kafka user config, keeping existing optional Kafka endpoint keys")
		return u, details, nil
	}

	// Temporary workaround until connection secret publishing removes stale keys.
	if kafkaConfig.KafkaAuthenticationMethods != nil &&
		kafkaConfig.KafkaAuthenticationMethods.Sasl != nil &&
		!*kafkaConfig.KafkaAuthenticationMethods.Sasl {
		details[prefix+"SASL_HOST"] = ""
		details[prefix+"SASL_PORT"] = ""
	}

	if kafkaConfig.SchemaRegistry != nil && !*kafkaConfig.SchemaRegistry {
		details[prefix+"SCHEMA_REGISTRY_HOST"] = ""
		details[prefix+"SCHEMA_REGISTRY_PORT"] = ""
	}

	return u, details, nil
}

func buildServiceUserAccessControlIn(ac *v1alpha1.ServiceUserAccessControl) *service.AccessControlIn {
	if ac == nil {
		return nil
	}

	ptr := func(in []string) *[]string {
		out := make([]string, len(in))
		copy(out, in)
		return &out
	}

	return &service.AccessControlIn{
		ValkeyAclKeys:       ptr(ac.ValkeyACLKeys),
		ValkeyAclCommands:   ptr(ac.ValkeyACLCommands),
		ValkeyAclCategories: ptr(ac.ValkeyACLCategories),
		ValkeyAclChannels:   ptr(ac.ValkeyACLChannels),
	}
}

func accessControlMatches(desired *v1alpha1.ServiceUserAccessControl, actual *service.AccessControlOut) bool {
	if desired == nil {
		return true
	}

	if actual == nil {
		return len(desired.ValkeyACLKeys) == 0 &&
			len(desired.ValkeyACLCommands) == 0 &&
			len(desired.ValkeyACLCategories) == 0 &&
			len(desired.ValkeyACLChannels) == 0
	}

	return lo.ElementsMatch(desired.ValkeyACLKeys, actual.ValkeyAclKeys) &&
		slices.Equal(desired.ValkeyACLCommands, actual.ValkeyAclCommands) &&
		slices.Equal(desired.ValkeyACLCategories, actual.ValkeyAclCategories) &&
		lo.ElementsMatch(desired.ValkeyACLChannels, actual.ValkeyAclChannels)
}

// TODO: Consider whether github.com/go-viper/mapstructure is a better fit if map-to-struct decoding grows.
func decodeMapInto(in map[string]any, out any) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}
