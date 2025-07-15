// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaconnect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// errSecretFetch returned when unable to fetch the secret, that is described in the connector UserConfig value.
const errSecretFetch = "unable to fetch secret"

// KafkaConnectorReconciler reconciles a KafkaConnector object
type KafkaConnectorReconciler struct {
	Controller
}

func newKafkaConnectorReconciler(c Controller) reconcilerType {
	return &KafkaConnectorReconciler{Controller: c}
}

type KafkaConnectorHandler struct {
	k8s client.Client
}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnectors/finalizers,verbs=get;create;update

func (r *KafkaConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaConnectorHandler{k8s: r.Client}, &v1alpha1.KafkaConnector{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *KafkaConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaConnector{}).
		Complete(r)
}

func (h KafkaConnectorHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) (bool, error) {
	conn, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	connCfg, err := h.buildConnectorConfig(ctx, conn)
	if err != nil {
		return false, fmt.Errorf("unable to build connector config: %w", err)
	}

	// Sometimes GET (ServiceKafkaConnectGetConnectorStatus) returns OK,
	// and POST (ServiceKafkaConnectCreateConnector) returns NotFound error.
	// So instead of asking Aiven API if the connector exists,
	// we try to create it.
	_, err = avnGen.ServiceKafkaConnectCreateConnector(ctx, conn.Spec.Project, conn.Spec.ServiceName, connCfg)
	exists := isAlreadyExists(err)
	if exists {
		_, err = avnGen.ServiceKafkaConnectEditConnector(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name, connCfg)
		if err != nil {
			return false, err
		}
	}

	switch {
	case isNotFound(err):
		// Means, the API isn't consistent yet,
		// because checkPreconditions() passed, yet we get 404.
		// Retry later.
		return false, nil
	case isServerError(err):
		// Service is not ready yet, retry later.
		return false, nil
	case err != nil:
		return false, err
	}

	return !exists, nil
}

// buildConnectorConfig joins mandatory fields with additional connector specific config
func (h KafkaConnectorHandler) buildConnectorConfig(ctx context.Context, conn *v1alpha1.KafkaConnector) (map[string]any, error) {
	const (
		configFieldConnectorName  = "name"
		configFieldConnectorClass = "connector.class"
	)
	var (
		templateFuncFromSecret = func(name, key string) (string, error) {
			var secret corev1.Secret
			objectKey := types.NamespacedName{Namespace: conn.GetNamespace(), Name: name}
			if err := h.k8s.Get(ctx, objectKey, &secret); err != nil {
				return "", fmt.Errorf("%s: %w", errSecretFetch, err)
			}
			v, ok := secret.Data[key]
			if !ok {
				return "", fmt.Errorf("no such key in secret '%s': '%s'", name, key)
			}
			return string(v), nil
		}

		funcMap = template.FuncMap{
			"fromSecret": templateFuncFromSecret,
		}
	)

	m := make(map[string]any)

	m[configFieldConnectorName] = conn.GetName()
	m[configFieldConnectorClass] = conn.Spec.ConnectorClass

	for k, v := range conn.Spec.UserConfig {
		t, err := template.New(k).Funcs(funcMap).Parse(v)
		if err != nil {
			return nil, fmt.Errorf("unable to parse template for key '%s': '%w'", k, err)
		}
		templateRes := new(bytes.Buffer)
		if err := t.Execute(templateRes, nil); err != nil {
			return nil, fmt.Errorf("unable to execute template for key '%s': '%w'", k, err)
		}
		m[k] = templateRes.String()
	}
	return m, nil
}

func (h KafkaConnectorHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	conn, err := h.convert(obj)
	if err != nil {
		return false, err
	}
	err = avnGen.ServiceKafkaConnectDeleteConnector(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name)
	if err != nil && !isNotFound(err) {
		return false, fmt.Errorf("unable to delete kafka connector: %w", err)
	}
	return true, nil
}

func (h KafkaConnectorHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	conn, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	connAtAiven, err := GetKafkaConnectorByName(ctx, avnGen, conn.Spec.Project, conn.Spec.ServiceName, conn.Name)
	if err != nil {
		return nil, err
	}
	conn.Status.PluginStatus = v1alpha1.KafkaConnectorPluginStatus{
		Author:  connAtAiven.Plugin.Author,
		Class:   connAtAiven.Plugin.Class,
		DocURL:  connAtAiven.Plugin.DocUrl,
		Title:   connAtAiven.Plugin.Title,
		Type:    string(connAtAiven.Plugin.Type),
		Version: connAtAiven.Plugin.Version,
	}

	connStat, err := avnGen.ServiceKafkaConnectGetConnectorStatus(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name)
	if err != nil {
		return nil, err
	}
	conn.Status.State = connStat.State
	conn.Status.TasksStatus = v1alpha1.KafkaConnectorTasksStatus{}
	for i := range connStat.Tasks {
		conn.Status.TasksStatus.Total++
		switch connStat.Tasks[i].State {
		case kafkaconnect.TaskStateTypeRunning:
			conn.Status.TasksStatus.Running++
		case kafkaconnect.TaskStateTypePaused:
			conn.Status.TasksStatus.Paused++
		case kafkaconnect.TaskStateTypeUnassigned:
			conn.Status.TasksStatus.Unassigned++
		case kafkaconnect.TaskStateTypeFailed:
			// in case we have a failed task, we just use the last failed stacktrace
			conn.Status.TasksStatus.Failed++
			conn.Status.TasksStatus.StackTrace = connStat.Tasks[i].Trace
		default:
			conn.Status.TasksStatus.Unknown++
		}
	}

	if connStat.State == kafkaconnect.ServiceKafkaConnectConnectorStateTypeRunning {
		meta.SetStatusCondition(&conn.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))
		metav1.SetMetaDataAnnotation(&conn.ObjectMeta, instanceIsRunningAnnotation, "true")
	}
	return nil, nil
}

func (h KafkaConnectorHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	conn, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&conn.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	// Check if the service is operational
	ok, err := checkServiceIsOperational(ctx, avnGen, conn.Spec.Project, conn.Spec.ServiceName)
	if !ok || err != nil {
		return ok, err
	}

	// Checks if the secret in the config is ready.
	// Instead of using error.Is() we check the error message,
	// because buildConnectorConfig() uses template engine, which might merge errors.
	_, err = h.buildConnectorConfig(ctx, conn)
	if err != nil && strings.Contains(err.Error(), errSecretFetch) {
		return false, nil
	}

	return err == nil, err
}

func (h KafkaConnectorHandler) convert(o client.Object) (*v1alpha1.KafkaConnector, error) {
	conn, ok := o.(*v1alpha1.KafkaConnector)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaConnector")
	}
	return conn, nil
}

func GetKafkaConnectorByName(ctx context.Context, avnGen avngen.Client, projectName, serviceName, name string) (*kafkaconnect.ConnectorOut, error) {
	list, err := avnGen.ServiceKafkaConnectList(ctx, projectName, serviceName)
	if err != nil {
		return nil, err
	}

	for _, v := range list {
		if v.Name == name {
			return &v, nil
		}
	}

	return nil, NewNotFound(fmt.Sprintf("Kafka connector with name %q not found", name))
}
