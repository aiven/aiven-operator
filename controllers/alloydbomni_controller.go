// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aiven/aiven-go-client/v2"
	alloydbomniUtils "github.com/aiven/aiven-operator/utils/alloydbomni"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/alloydbomni"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	alloydbomniUtils "github.com/aiven/aiven-operator/utils/alloydbomni"
)

// AlloyDBOmniReconciler reconciles a AlloyDBOmni object
type AlloyDBOmniReconciler struct {
	Controller
}

func newAlloyDBOmniReconciler(c Controller) reconcilerType {
	return &AlloyDBOmniReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=alloydbomnis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=alloydbomnis/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=alloydbomnis/finalizers,verbs=get;create;update

func (r *AlloyDBOmniReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newAlloyDBOmniAdapter), &v1alpha1.AlloyDBOmni{})
}

func (r *AlloyDBOmniReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AlloyDBOmni{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newAlloyDBOmniAdapter(_ *aiven.Client, object client.Object) (serviceAdapter, error) {
	adbo, ok := object.(*v1alpha1.AlloyDBOmni)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.AlloyDBOmni")
	}
	return &alloyDBOmniAdapter{adbo}, nil
}

// alloyDBOmniAdapter handles an Aiven AlloyDBOmni service
type alloyDBOmniAdapter struct {
	*v1alpha1.AlloyDBOmni
}

func (a *alloyDBOmniAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *alloyDBOmniAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *alloyDBOmniAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *alloyDBOmniAdapter) getUserConfig() any {
	return a.Spec.UserConfig
}

func (a *alloyDBOmniAdapter) newSecret(ctx context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
	stringData := map[string]string{
		"HOST":         s.ServiceUriParams["host"],
		"PORT":         s.ServiceUriParams["port"],
		"DATABASE":     s.ServiceUriParams["dbname"],
		"USER":         s.ServiceUriParams["user"],
		"PASSWORD":     s.ServiceUriParams["password"],
		"SSLMODE":      s.ServiceUriParams["sslmode"],
		"DATABASE_URI": s.ServiceUri,
	}

	return newSecret(a, stringData, true), nil
}

func (a *alloyDBOmniAdapter) getServiceType() string {
	return "alloydbomni"
}

func (a *alloyDBOmniAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *alloyDBOmniAdapter) performUpgradeTaskIfNeeded(ctx context.Context, avn avngen.Client, old *service.ServiceGetOut) error {
	return nil
}

func (a *alloyDBOmniAdapter) createOrUpdateServiceSpecific(ctx context.Context, avnGen avngen.Client, old *service.ServiceGetOut) error {
	// Get current credentials state
	currentCreds, err := avnGen.AlloyDbOmniGoogleCloudPrivateKeyIdentify(ctx, a.Spec.Project, a.Name)
	if err != nil {
		return fmt.Errorf("failed to get current credentials: %w", err)
	}

	// Handle removal case first
	if a.Spec.ServiceAccountCredentials == "" {
		if currentCreds.PrivateKeyId != "" {
			if _, err := avnGen.AlloyDbOmniGoogleCloudPrivateKeyRemove(ctx, a.Spec.Project, a.Name); err != nil {
				return fmt.Errorf("failed to remove credentials: %w", err)
			}
		}
		return nil
	}

	// Validate new credentials
	if err := validateCredentials(a.Spec.ServiceAccountCredentials); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	// Parse credentials to get key ID
	var credsMap map[string]interface{}
	if err := json.Unmarshal([]byte(a.Spec.ServiceAccountCredentials), &credsMap); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Update only if key ID changed
	if newKeyID := credsMap["private_key_id"].(string); newKeyID != currentCreds.PrivateKeyId {
		req := &alloydbomni.AlloyDbOmniGoogleCloudPrivateKeySetIn{
			PrivateKey: a.Spec.ServiceAccountCredentials,
		}
		if _, err := avnGen.AlloyDbOmniGoogleCloudPrivateKeySet(ctx, a.Spec.Project, a.Name, req); err != nil {
			return fmt.Errorf("failed to update credentials: %w", err)
		}
	}

	return nil
}

// Helper function to validate credentials
func validateCredentials(creds string) error {
	if err := alloydbomniUtils.ValidateServiceAccountCredentials(creds); err != nil {
		return err
	}
	return nil
}
