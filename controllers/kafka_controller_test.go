// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
)

func kafkaSpec(serviceName, namespace string) *v1alpha1.Kafka {
	return &v1alpha1.Kafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Kafka",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.KafkaSpec{
			DiskSpace: "600Gib",
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "business-4",
				CloudName: "google-europe-west1",
				Tags:      map[string]string{"key1": "value1"},
			},
			UserConfig: &kafkauserconfig.KafkaUserConfig{
				KafkaRest:      anyPointer(true),
				KafkaConnect:   anyPointer(true),
				SchemaRegistry: anyPointer(true),
				Kafka: &kafkauserconfig.Kafka{
					GroupMaxSessionTimeoutMs: anyPointer(70000),
					LogRetentionBytes:        anyPointer(1000000000),
				},
			},
			AuthSecretRef: &v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
