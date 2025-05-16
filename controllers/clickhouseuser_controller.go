// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ClickhouseUserReconciler reconciles a ClickhouseUser object
type ClickhouseUserReconciler struct {
	Controller
}

func newClickhouseUserReconciler(c Controller) reconcilerType {
	return &ClickhouseUserReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers/finalizers,verbs=get;create;update

func (r *ClickhouseUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, &clickhouseUserHandler{}, &v1alpha1.ClickhouseUser{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClickhouseUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClickhouseUser{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

type clickhouseUserHandler struct{}

func (h *clickhouseUserHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, _ avngen.Client, obj client.Object, _ []client.Object) error {
	user, err := h.convert(obj)
	if err != nil {
		return err
	}

	list, err := avn.ClickhouseUser.List(ctx, user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return err
	}

	var uuid string
	for _, u := range list.Users {
		if u.Name == user.GetUsername() {
			uuid = u.UUID
			break
		}
	}

	if uuid == "" {
		r, err := avn.ClickhouseUser.Create(ctx, user.Spec.Project, user.Spec.ServiceName, user.GetUsername())
		if err != nil {
			return err
		}

		uuid = r.User.UUID
	}

	user.Status.UUID = uuid

	meta.SetStatusCondition(&user.Status.Conditions,
		getInitializedCondition("Created",
			"Successfully created or updated the instance in Aiven"))

	metav1.SetMetaDataAnnotation(&user.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(user.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h *clickhouseUserHandler) delete(ctx context.Context, avn *aiven.Client, _ avngen.Client, obj client.Object) (bool, error) {
	user, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	// Not processed yet
	if user.Status.UUID == "" {
		return true, nil
	}

	err = avn.ClickhouseUser.Delete(ctx, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID)
	if !isNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h *clickhouseUserHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	user, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	s, err := avnGen.ServiceGet(ctx, user.Spec.Project, user.Spec.ServiceName, service.ServiceGetIncludeSecrets(true))
	if err != nil {
		return nil, err
	}

	// By design this handler can't create secret in createOrUpdate method,
	// while password is returned on create only.
	// And all other GET methods return empty password, even this one.
	// So the only way to have a secret here is to reset it manually
	password, err := avn.ClickhouseUser.ResetPassword(ctx, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID, nil)
	if err != nil {
		return nil, err
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

	secret := newSecret(user, stringData, false)

	meta.SetStatusCondition(&user.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")
	return secret, nil
}

func (h *clickhouseUserHandler) checkPreconditions(ctx context.Context, _ *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	user, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsOperational(ctx, avnGen, user.Spec.Project, user.Spec.ServiceName)
}

func (h *clickhouseUserHandler) convert(i client.Object) (*v1alpha1.ClickhouseUser, error) {
	user, ok := i.(*v1alpha1.ClickhouseUser)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ClickhouseUser")
	}
	return user, nil
}
