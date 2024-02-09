// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"text/template"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
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
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnectors/finalizers,verbs=get;list;watch;create;update;patch;delete

func (r *KafkaConnectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaConnectorHandler{k8s: r.Client}, &v1alpha1.KafkaConnector{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *KafkaConnectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaConnector{}).
		Complete(r)
}

func (h KafkaConnectorHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	conn, err := h.convert(obj)
	if err != nil {
		return err
	}

	exists, err := h.exists(ctx, avn, conn)
	if err != nil {
		return fmt.Errorf("unable to check if kafka connector exists: %w", err)
	}

	connCfg, err := h.buildConnectorConfig(conn)
	if err != nil {
		return fmt.Errorf("unable to build connector config: %w", err)
	}

	var reason string
	if !exists {
		err = avn.KafkaConnectors.Create(ctx, conn.Spec.Project, conn.Spec.ServiceName, connCfg)
		if err != nil && !aiven.IsAlreadyExists(err) {
			return err
		}
		reason = "Created"
	} else {
		_, err = avn.KafkaConnectors.Update(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name, connCfg)
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

// buildConnectorConfig joins mandatory fields with additional connector specific config
func (h KafkaConnectorHandler) buildConnectorConfig(conn *v1alpha1.KafkaConnector) (aiven.KafkaConnectorConfig, error) {
	const (
		configFieldConnectorName  = "name"
		configFieldConnectorClass = "connector.class"
	)
	var (
		templateFuncFromSecret = func(name, key string) (string, error) {
			var secret corev1.Secret

			if err := h.k8s.Get(context.Background(), types.NamespacedName{Namespace: conn.GetNamespace(), Name: name}, &secret); err != nil {
				return "", fmt.Errorf("unable to fetch secret: '%w'", err)
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

	m := make(map[string]string)

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

func (h KafkaConnectorHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	conn, err := h.convert(obj)
	if err != nil {
		return false, err
	}
	err = avn.KafkaConnectors.Delete(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name)
	if err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("unable to delete kafka connector: %w", err)
	}
	return true, nil
}

func (h KafkaConnectorHandler) exists(ctx context.Context, avn *aiven.Client, conn *v1alpha1.KafkaConnector) (bool, error) {
	connector, err := avn.KafkaConnectors.Status(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name)
	if err != nil && !aiven.IsNotFound(err) {
		return false, err
	}
	return connector != nil, nil
}

func (h KafkaConnectorHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	conn, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	connAtAiven, err := avn.KafkaConnectors.GetByName(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name)
	if err != nil {
		return nil, err
	}
	conn.Status.PluginStatus = v1alpha1.KafkaConnectorPluginStatus{
		Author:  connAtAiven.Plugin.Author,
		Class:   connAtAiven.Plugin.Class,
		DocURL:  connAtAiven.Plugin.DocumentationURL,
		Title:   connAtAiven.Plugin.Title,
		Type:    connAtAiven.Plugin.Type,
		Version: connAtAiven.Plugin.Version,
	}

	connStat, err := avn.KafkaConnectors.Status(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name)
	if err != nil {
		return nil, err
	}
	conn.Status.State = connStat.Status.State
	conn.Status.TasksStatus = v1alpha1.KafkaConnectorTasksStatus{}
	for i := range connStat.Status.Tasks {
		conn.Status.TasksStatus.Total++
		switch connStat.Status.Tasks[i].State {
		case "RUNNING":
			conn.Status.TasksStatus.Running++
		case "PAUSED":
			conn.Status.TasksStatus.Paused++
		case "UNASSIGNED":
			conn.Status.TasksStatus.Unassigned++
		case "FAILED":
			// in case we have a failed task, we just use the last failed stacktrace
			conn.Status.TasksStatus.Failed++
			conn.Status.TasksStatus.StackTrace = connStat.Status.Tasks[i].Trace
		default:
			conn.Status.TasksStatus.Unknown++
		}
	}

	if connStat.Status.State == "RUNNING" {
		meta.SetStatusCondition(&conn.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))
		metav1.SetMetaDataAnnotation(&conn.ObjectMeta, instanceIsRunningAnnotation, "true")
	}
	return nil, nil
}

func (h KafkaConnectorHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	conn, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&conn.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsRunning(ctx, avn, avnGen, conn.Spec.Project, conn.Spec.ServiceName)
}

func (h KafkaConnectorHandler) convert(o client.Object) (*v1alpha1.KafkaConnector, error) {
	conn, ok := o.(*v1alpha1.KafkaConnector)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaConnector")
	}
	return conn, nil
}
