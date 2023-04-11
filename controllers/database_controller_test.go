// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func databaseSpec(service, database, namespace string) *v1alpha1.Database {
	return &v1alpha1.Database{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Database",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      database,
			Namespace: namespace,
		},
		Spec: v1alpha1.DatabaseSpec{
			Project:     os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName: service,
			LcCtype:     "en_US.UTF-8",
			LcCollate:   "en_US.UTF-8",
			AuthSecretRef: &v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
