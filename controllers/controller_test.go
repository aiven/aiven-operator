// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestUserConfigurationToAPI(t *testing.T) {
	var tempFileLimit int64
	var publicAccessPg bool
	type args struct {
		c interface{}
	}

	tempFileLimit = -1
	publicAccessPg = true

	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			name: "basic",
			args: args{
				c: v1alpha1.PostgreSQLUserconfig{
					PgVersion: "12",
					Pg: v1alpha1.PostgreSQLSubUserConfig{
						Timezone:      "CEST",
						TempFileLimit: &tempFileLimit,
					},
					PublicAccess: v1alpha1.PublicAccessUserConfig{
						Pg: &publicAccessPg,
					},
				},
			},
			want: map[string]interface{}{
				"pg_version": "12",
				"pg": map[string]interface{}{
					"temp_file_limit": int64(-1),
					"timezone":        "CEST",
				},
				"public_access": map[string]interface{}{
					"pg": true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UserConfigurationToAPI(tt.args.c)
			assert.Equal(t, got, tt.want)
		})
	}
}

func Test_ensureSecretDataIsNotEmpty(t *testing.T) {
	type args struct {
		log logr.Logger
		s   *corev1.Secret
	}
	tests := []struct {
		name string
		args args
		want *corev1.Secret
	}{
		{
			"basic",
			args{
				log: nil,
				s: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-name",
						Namespace: "some-namespace",
					},
					StringData: map[string]string{
						"PGHOST":       "host",
						"PGPORT":       "port",
						"PGDATABASE":   "db",
						"PGUSER":       "user",
						"PGPASSWORD":   "pass",
						"PGSSLMODE":    "mode",
						"DATABASE_URI": "uri",
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-name",
					Namespace: "some-namespace",
				},
				StringData: map[string]string{
					"PGHOST":       "host",
					"PGPORT":       "port",
					"PGDATABASE":   "db",
					"PGUSER":       "user",
					"PGPASSWORD":   "pass",
					"PGSSLMODE":    "mode",
					"DATABASE_URI": "uri",
				},
			},
		},
		{
			"one-empty",
			args{
				log: nil,
				s: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-name",
						Namespace: "some-namespace",
					},
					StringData: map[string]string{
						"PGHOST":       "",
						"PGPORT":       "port",
						"PGDATABASE":   "db",
						"PGUSER":       "user",
						"PGPASSWORD":   "pass",
						"PGSSLMODE":    "mode",
						"DATABASE_URI": "uri",
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-name",
					Namespace: "some-namespace",
				},
				StringData: map[string]string{
					"PGPORT":       "port",
					"PGDATABASE":   "db",
					"PGUSER":       "user",
					"PGPASSWORD":   "pass",
					"PGSSLMODE":    "mode",
					"DATABASE_URI": "uri",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ensureSecretDataIsNotEmpty(tt.args.log, tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ensureSecretDataIsNotEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
