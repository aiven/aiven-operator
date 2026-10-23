// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestKafkaWebhookValidateDelete(t *testing.T) {
	t.Parallel()

	t.Run("rejects projectVpcId together with projectVPCRef", func(t *testing.T) {
		t.Parallel()
		for name, id := range map[string]*string{"id": new("vpc-1"), "empty": new("")} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				in := &v1alpha1.Kafka{}
				in.Spec.ProjectVPCID = id
				in.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "vpc"}
				_, err := (&KafkaWebhook{}).ValidateDelete(t.Context(), in)
				require.ErrorContains(t, err, "cannot use both projectVpcId and projectVPCRef")
			})
		}
	})

	t.Run("allows either alone", func(t *testing.T) {
		t.Parallel()
		in := &v1alpha1.Kafka{}
		in.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "vpc"}
		_, err := (&KafkaWebhook{}).ValidateDelete(t.Context(), in)
		require.NoError(t, err)
	})
}
