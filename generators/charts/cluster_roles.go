package main

import (
	"fmt"
	"os"
	"path"

	"github.com/goccy/go-yaml"
)

type clusterRoleYaml struct {
	Rules []struct {
		APIGroups []string `yaml:"apiGroups,omitempty"`
		Resources []string `yaml:"resources,omitempty"`
		Verbs     []string `yaml:"verbs,omitempty"`
	} `yaml:"rules,omitempty"`
}

func updateClusterRole(operatorPath, crdCharts string) error {
	srcPath := path.Join(operatorPath, "config/rbac/role.yaml")

	updated := new(clusterRoleYaml)
	f, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(f, updated)
	if err != nil {
		return err
	}

	result, err := marshalCompactYaml(updated)
	if err != nil {
		return err
	}
	content := fmt.Sprintf(clusterRoleTmpl, result)
	dstPath := path.Join(crdCharts, "templates/cluster_role.yaml")
	return writeFile(dstPath, []byte(content))
}

var clusterRoleTmpl = `{{- if .Values.clusterRole.create -}}
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ include "aiven-operator.fullname" . }}-role
  namespace: {{ include "aiven-operator.namespace" . }}
  labels:
    {{- include "aiven-operator.labels" . | nindent 4 }}
%s{{- end }}
`
