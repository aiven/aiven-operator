// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	pguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/pg"
)

func pgSpec(serviceName, namespace string) *v1alpha1.PostgreSQL {
	return &v1alpha1.PostgreSQL{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "PostgreSQL",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.PostgreSQLSpec{
			DiskSpace: "100Gib",
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "business-4",
				CloudName: "google-europe-west1",
				Tags:      map[string]string{"key1": "value1"},
			},
			UserConfig: &pguserconfig.PgUserConfig{
				// YAML converts string integers to real integers,
				// Then it fails with the validation as invalid choice
				PgVersion: anyPointer("14"),
				PublicAccess: &pguserconfig.PublicAccess{
					Pg:         anyPointer(true),
					Prometheus: anyPointer(true),
				},
				Pg: &pguserconfig.Pg{
					IdleInTransactionSessionTimeout: anyPointer(900),
				},
			},
			AuthSecretRef: &v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
