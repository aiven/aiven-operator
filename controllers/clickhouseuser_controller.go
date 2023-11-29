// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
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

//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers/finalizers,verbs=get;list;watch;create;update;patch;delete

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

func (h *clickhouseUserHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, obj client.Object, refs []client.Object) error {
	user, err := h.convert(obj)
	if err != nil {
		return err
	}

	r, err := avn.ClickhouseUser.Create(ctx, user.Spec.Project, user.Spec.ServiceName, user.Name)
	if err != nil {
		return err
	}

	user.Status.UUID = r.User.UUID

	meta.SetStatusCondition(&user.Status.Conditions,
		getInitializedCondition("Created",
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&user.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "Created",
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&user.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(user.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h *clickhouseUserHandler) delete(ctx context.Context, avn *aiven.Client, obj client.Object) (bool, error) {
	user, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	// Not processed yet
	if user.Status.UUID == "" {
		return true, nil
	}

	err = avn.ClickhouseUser.Delete(ctx, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID)
	if !aiven.IsNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h *clickhouseUserHandler) get(ctx context.Context, avn *aiven.Client, obj client.Object) (*corev1.Secret, error) {
	user, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	s, err := avn.Services.Get(ctx, user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return nil, err
	}

	// By design this handler can't create secret in createOrUpdate method,
	// while password is returned on create only.
	// And all other GET methods return empty password, even this one.
	// So the only way to have a secret here is to reset it manually
	password := randPassword(maxUserPasswordLength)
	_, err = avn.ClickhouseUser.ResetPassword(ctx, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID, password)
	if err != nil {
		return nil, err
	}

	prefix := getSecretPrefix(user)
	stringData := map[string]string{
		prefix + "HOST":     s.URIParams["host"],
		prefix + "PORT":     s.URIParams["port"],
		prefix + "PASSWORD": password,
		prefix + "USERNAME": user.Name,
		// todo: remove in future releases
		"HOST":     s.URIParams["host"],
		"PORT":     s.URIParams["port"],
		"PASSWORD": password,
		"USERNAME": user.Name,
	}

	secret := newSecret(user, stringData, false)

	meta.SetStatusCondition(&user.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")
	return secret, nil
}

func (h *clickhouseUserHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, obj client.Object) (bool, error) {
	user, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsRunning(ctx, avn, user.Spec.Project, user.Spec.ServiceName)
}

func (h *clickhouseUserHandler) convert(i client.Object) (*v1alpha1.ClickhouseUser, error) {
	user, ok := i.(*v1alpha1.ClickhouseUser)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ClickhouseUser")
	}
	return user, nil
}

const maxUserPasswordLength = 24

func randPassword(len int) string {
	buff := make([]byte, len)
	_, _ = rand.Read(buff)
	str := base64.StdEncoding.EncodeToString(buff)
	return str[:len]
}
