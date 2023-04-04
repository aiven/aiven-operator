// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	clickhouseuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/clickhouse"
)

func chSpec(serviceName, namespace string) *v1alpha1.Clickhouse {
	return &v1alpha1.Clickhouse{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Clickhouse",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.ClickhouseSpec{
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "business-16",
				CloudName: "google-europe-west1",
				Tags:      map[string]string{"key1": "value1"},
			},
			UserConfig: &clickhouseuserconfig.ClickhouseUserConfig{
				IpFilter: []*clickhouseuserconfig.IpFilter{
					{
						Network: "10.20.0.0/16",
					},
					{
						Network:     "0.0.0.0",
						Description: anyPointer("whatever"),
					},
				},
			},
			AuthSecretRef: &v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
