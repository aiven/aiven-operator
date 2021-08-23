// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
)

// KafkaConnectorReconciler reconciles a KafkaConnector object
type KafkaConnectorReconciler struct {
	Controller
}

type KafkaConnectorHandler struct {
	k8s client.Client
}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnectors/status,verbs=get;update;patch

func (r *KafkaConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaConnectorHandler{k8s: r.Client}, &v1alpha1.KafkaConnector{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *KafkaConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaConnector{}).
		Complete(r)
}

func (h KafkaConnectorHandler) createOrUpdate(avn *aiven.Client, o client.Object) error {
	conn, err := h.convert(o)
	if err != nil {
		return err
	}

	exists, err := h.exists(avn, conn)
	if err != nil {
		return fmt.Errorf("unable to check if kafka connector exists: %w", err)
	}

	connCfg, err := h.buildConnectorConfig(conn)
	if err != nil {
		return fmt.Errorf("unable to build connector config: %w", err)
	}

	var reason string
	if !exists {
		err = avn.KafkaConnectors.Create(conn.Spec.Project, conn.Spec.ServiceName, connCfg)
		if err != nil && !aiven.IsAlreadyExists(err) {
			return err
		}
		reason = "Created"
	} else {
		_, err := avn.KafkaConnectors.Update(conn.Spec.Project, conn.Spec.ServiceName, conn.Name, connCfg)
		if err != nil {
			return err
		}
		reason = "Updated"

	}

	meta.SetStatusCondition(&conn.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&conn.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&conn.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(conn.GetGeneration(), formatIntBaseDecimal))

	return nil
}

// buildConnectorConfig joins mandatory fields with additional conncetor specific config
func (h KafkaConnectorHandler) buildConnectorConfig(conn *v1alpha1.KafkaConnector) (aiven.KafkaConnectorConfig, error) {
	const (
		configFieldConnectorName  = "name"
		configFieldConnectorClass = "connector.class"

		configMarkerFromSecret         = "secretRef"
		configSecretSeperator          = ":"
		configExpectedFieldsWithSecret = 3
	)
	m := make(map[string]string)

	m[configFieldConnectorName] = conn.GetName()
	m[configFieldConnectorClass] = conn.Spec.ConnectorClass

	for k, v := range conn.Spec.ConnectorSpecificConfig {
		if !strings.HasPrefix(v, configMarkerFromSecret) {
			m[k] = v
		} else {
			fields := strings.Split(v, configSecretSeperator)
			if len(fields) != configExpectedFieldsWithSecret {
				return nil, fmt.Errorf("bad config key '%s': unexpected number of secret fields: %d", k, len(fields))
			}
			sn, sk := fields[1], fields[2]

			var secret corev1.Secret
			if err := h.k8s.Get(context.Background(), types.NamespacedName{Namespace: conn.GetNamespace(), Name: sn}, &secret); err != nil {
				return nil, fmt.Errorf("unable to fetch secret for config key '%s': %w", k, err)
			}
			m[k] = string(secret.Data[sk])
		}
	}
	return aiven.KafkaConnectorConfig(m), nil
}

func (h KafkaConnectorHandler) delete(avn *aiven.Client, o client.Object) (bool, error) {
	conn, err := h.convert(o)
	if err != nil {
		return false, err
	}
	if err := avn.KafkaConnectors.Delete(conn.Spec.Project, conn.Spec.ServiceName, conn.Name); err != nil {
		if !aiven.IsNotFound(err) {
			return false, fmt.Errorf("unable to delete kafka connector: %w", err)
		}
	}
	return true, nil
}

func (h KafkaConnectorHandler) exists(avn *aiven.Client, conn *v1alpha1.KafkaConnector) (bool, error) {
	connectorsAtAiven, err := avn.KafkaConnectors.List(conn.Spec.Project, conn.Spec.ServiceName)
	if err != nil {
		return false, ignoreAivenNotFound(err)
	}

	for i := range connectorsAtAiven.Connectors {
		if connectorsAtAiven.Connectors[i].Name == conn.Name {
			return true, nil
		}
	}

	return false, nil
}

func (h KafkaConnectorHandler) get(avn *aiven.Client, o client.Object) (*corev1.Secret, error) {
	conn, err := h.convert(o)
	if err != nil {
		return nil, err
	}

	if exists, err := h.exists(avn, conn); err != nil {
		return nil, err
	} else if exists {
		// TODO: for now we set the connector to running once it is created. This is not true as
		// Tasks may have failed. This will be eventually handled by a status observer reconciler
		// see https://github.com/aiven/aiven-kubernetes-operator/issues/133 for details.
		meta.SetStatusCondition(&conn.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&conn.ObjectMeta, instanceIsRunningAnnotation, "true")
	}
	return nil, nil
}

func (h KafkaConnectorHandler) checkPreconditions(avn *aiven.Client, o client.Object) (bool, error) {
	const (
		configKeyKafkaConnect = "kafka_connect"
	)
	conn, err := h.convert(o)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&conn.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	kafka, err := avn.Services.Get(conn.Spec.Project, conn.Spec.ServiceName)
	if err != nil {
		return false, err
	}
	if kafka.State != "RUNNING" {
		return false, nil
	}
	hasConnectEnabledMapVal, ok := kafka.UserConfig[configKeyKafkaConnect]
	if !ok {
		return false, nil
	}
	hasConnectEnabled, ok := hasConnectEnabledMapVal.(bool)
	if !ok {
		return false, nil
	}
	return hasConnectEnabled, nil
}

func (h KafkaConnectorHandler) convert(o client.Object) (*v1alpha1.KafkaConnector, error) {
	conn, ok := o.(*v1alpha1.KafkaConnector)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaConnector")
	}
	return conn, nil
}
