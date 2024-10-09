package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/stoewer/go-strcase"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

const yamlBufferSize = 100

// whManifest represents webhook/manifests.yaml
type whManifest struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		CreationTimestamp interface{} `yaml:"creationTimestamp"`
		Name              string      `yaml:"name"`
	} `yaml:"metadata"`

	whManifestProp `yaml:",inline"`
}

// whManifestProp separate struct to render Webhooks prop only
type whManifestProp struct {
	Webhooks []struct {
		AdmissionReviewVersions []string `yaml:"admissionReviewVersions"`
		ClientConfig            struct {
			Service struct {
				Name      string `yaml:"name"`
				Namespace string `yaml:"namespace"`
				Path      string `yaml:"path"`
			} `yaml:"service"`
		} `yaml:"clientConfig"`
		FailurePolicy string `yaml:"failurePolicy"`
		Name          string `yaml:"name"`
		Rules         []struct {
			APIGroups   []string `yaml:"apiGroups"`
			APIVersions []string `yaml:"apiVersions"`
			Operations  []string `yaml:"operations"`
			Resources   []string `yaml:"resources"`
		} `yaml:"rules"`
		SideEffects       string      `yaml:"sideEffects"`
		NamespaceSelector interface{} `yaml:"namespaceSelector"`
	} `yaml:"webhooks"`
}

// updateWebhooks creates charts for webhooks using operators files
// as a result it creates two separate "charts" for mutating and validating webhooks
func updateWebhooks(operatorPath, operatorCharts string) error {
	file, err := os.ReadFile(path.Join(operatorPath, "config/webhook/manifests.yaml"))
	if err != nil {
		return err
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(file), yamlBufferSize)
	for {
		var wh whManifest
		err = decoder.Decode(&wh)
		if err != nil {
			break
		}

		b, err := marshalCompactYaml(&wh.whManifestProp)
		if err != nil {
			return err
		}

		// Renders manifest template
		data := fmt.Sprintf(manifestTemplate, wh.Kind, wh.Metadata.Name, b.String())

		// Replaces name, namespace and namespaceSelector with Helm template inclusions
		data = strings.ReplaceAll(data, `name: webhook-service`, `name: {{ include "aiven-operator.fullname" . }}-webhook-service`)
		data = strings.ReplaceAll(data, `namespace: system`, `namespace: {{ include "aiven-operator.namespace" . }}`)
		// Since kubebuilder:webhook does not support the `namespaceSelector` field, we replace its null occurrences
		data = strings.ReplaceAll(data, `namespaceSelector: null`, `{{- include "aiven-operator.webhookNamespaceSelector" . | indent 4 }}`)

		// Creates files according to Metadata.Name in snake case
		filePath := path.Join(operatorCharts, "templates", strcase.SnakeCase(wh.Metadata.Name)+".yaml")
		err = writeFile(filePath, []byte(data))
		if err != nil {
			return err
		}
	}

	return nil
}

var manifestTemplate = `{{- if .Values.webhooks.enabled }}
apiVersion: admissionregistration.k8s.io/v1
kind: %s
metadata:
  annotations:
{{- include "aiven-operator.ca_injection_annotation" . | nindent 4 }}
  name: {{ include "aiven-operator.fullname" . }}-%s
  namespace: {{ include "aiven-operator.namespace" . }}
  labels:
{{- include "aiven-operator.labels" . | nindent 4 }}
%s
{{- end }}
`
