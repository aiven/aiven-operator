package controllers

import (
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
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
				c: k8soperatorv1alpha1.PGUserConfig{
					PgVersion: "12",
					Pg: k8soperatorv1alpha1.PGSubPGUserConfig{
						Timezone:      "CEST",
						TempFileLimit: &tempFileLimit,
					},
					PublicAccess: k8soperatorv1alpha1.PublicAccessUserConfig{
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
