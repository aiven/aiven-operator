// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"text/template"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaconnect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// errSecretFetch returned when unable to fetch the secret, that is described in the connector UserConfig value.
var errSecretFetch = errors.New("unable to fetch secret")

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaconnectors/finalizers,verbs=get;create;update

// KafkaConnectorController reconciles a KafkaConnector object.
type KafkaConnectorController struct {
	client.Client
	avnGen avngen.Client
}

func newKafkaConnectorReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.KafkaConnector] {
			return &KafkaConnectorController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *KafkaConnectorController) Observe(ctx context.Context, conn *v1alpha1.KafkaConnector) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, conn.Spec.Project, conn.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	// A missing secret (errSecretFetch) is a transient precondition error.
	if _, err := r.buildConnectorConfig(ctx, conn); err != nil {
		if errors.Is(err, errSecretFetch) {
			return Observation{}, fmt.Errorf("%w: %w", errPreconditionNotMet, err)
		}
		return Observation{}, err
	}

	connAtAiven, err := GetKafkaConnectorByName(ctx, r.avnGen, conn.Spec.Project, conn.Spec.ServiceName, conn.Name)
	switch {
	case isNotFound(err):
		return Observation{ResourceExists: false}, nil
	case err != nil:
		return Observation{}, fmt.Errorf("describing kafka connector: %w", err)
	}

	conn.Status.PluginStatus = v1alpha1.KafkaConnectorPluginStatus{
		Author:  connAtAiven.Plugin.Author,
		Class:   connAtAiven.Plugin.Class,
		DocURL:  connAtAiven.Plugin.DocUrl,
		Title:   connAtAiven.Plugin.Title,
		Type:    string(connAtAiven.Plugin.Type),
		Version: connAtAiven.Plugin.Version,
	}

	connStat, err := r.avnGen.ServiceKafkaConnectGetConnectorStatus(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name)
	if err != nil {
		return Observation{}, fmt.Errorf("getting kafka connector status: %w", err)
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

	// Mark running only when the connector is actually RUNNING on Aiven side.
	if connStat.State == kafkaconnect.ServiceKafkaConnectConnectorStateTypeRunning {
		markInstanceRunning(conn)
	}

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: hasLatestGeneration(conn),
	}, nil
}

func (r *KafkaConnectorController) Create(ctx context.Context, conn *v1alpha1.KafkaConnector) (CreateResult, error) {
	delete(conn.GetAnnotations(), instanceIsRunningAnnotation)

	connCfg, err := r.buildConnectorConfig(ctx, conn)
	if err != nil {
		return CreateResult{}, fmt.Errorf("unable to build connector config: %w", err)
	}

	_, err = r.avnGen.ServiceKafkaConnectCreateConnector(ctx, conn.Spec.Project, conn.Spec.ServiceName, &connCfg)
	switch {
	case isAlreadyExists(err) || isNotFound(err) || isServerError(err):
		// Transient errors, just requeue.
		return CreateResult{}, fmt.Errorf("%w: %w", errPreconditionNotMet, err)
	case err != nil:
		return CreateResult{}, fmt.Errorf("cannot create kafka connector on Aiven side: %w", err)
	}

	const reason = "Created"
	meta.SetStatusCondition(&conn.Status.Conditions, getInitializedCondition(reason, "Successfully created the instance in Aiven"))
	meta.SetStatusCondition(&conn.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created the instance in Aiven, status remains unknown"))

	return CreateResult{}, nil
}

func (r *KafkaConnectorController) Update(ctx context.Context, conn *v1alpha1.KafkaConnector) (UpdateResult, error) {
	delete(conn.GetAnnotations(), instanceIsRunningAnnotation)

	connCfg, err := r.buildConnectorConfig(ctx, conn)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("unable to build connector config: %w", err)
	}

	_, err = r.avnGen.ServiceKafkaConnectEditConnector(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name, &connCfg)
	switch {
	case isNotFound(err) || isServerError(err):
		// Kafka Connect API not consistent yet (404) or service not ready (5xx). Requeue softly.
		return UpdateResult{}, fmt.Errorf("%w: %w", errPreconditionNotMet, err)
	case err != nil:
		return UpdateResult{}, fmt.Errorf("cannot update kafka connector on Aiven side: %w", err)
	}

	const reason = "Updated"
	meta.SetStatusCondition(&conn.Status.Conditions, getInitializedCondition(reason, "Successfully updated the instance in Aiven"))
	meta.SetStatusCondition(&conn.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully updated the instance in Aiven, status remains unknown"))

	return UpdateResult{}, nil
}

func (r *KafkaConnectorController) Delete(ctx context.Context, conn *v1alpha1.KafkaConnector) error {
	err := r.avnGen.ServiceKafkaConnectDeleteConnector(ctx, conn.Spec.Project, conn.Spec.ServiceName, conn.Name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("unable to delete kafka connector: %w", err)
	}
	return nil
}

// buildConnectorConfig joins mandatory fields with additional connector specific config
func (r *KafkaConnectorController) buildConnectorConfig(ctx context.Context, conn *v1alpha1.KafkaConnector) (map[string]string, error) {
	const (
		configFieldConnectorName  = "name"
		configFieldConnectorClass = "connector.class"
	)
	var (
		templateFuncFromSecret = func(name, key string) (string, error) {
			var secret corev1.Secret
			objectKey := types.NamespacedName{Namespace: conn.GetNamespace(), Name: name}
			if err := r.Get(ctx, objectKey, &secret); err != nil {
				return "", fmt.Errorf("%w: %w", errSecretFetch, err)
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

func GetKafkaConnectorByName(ctx context.Context, avnGen avngen.Client, projectName, serviceName, name string) (*kafkaconnect.ConnectorOut, error) {
	list, err := avnGen.ServiceKafkaConnectList(ctx, projectName, serviceName)
	if err != nil {
		return nil, err
	}

	for _, v := range list.Connectors {
		if v.Name == name {
			return &v, nil
		}
	}

	return nil, NewNotFound(fmt.Sprintf("Kafka connector with name %q not found", name))
}
