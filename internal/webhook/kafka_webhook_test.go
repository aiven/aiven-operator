// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package webhook

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestKafkaWebhookProjectVPCConflict(t *testing.T) {
	t.Parallel()

	// projectVpcId set to anything, including "", together with projectVPCRef.
	for name, id := range map[string]*string{"id": new("vpc-1"), "empty": new("")} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := &v1alpha1.Kafka{}
			in.Spec.ProjectVPCID = id
			in.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "vpc"}
			h := &KafkaWebhook{}

			_, err := h.ValidateCreate(t.Context(), in)
			require.ErrorContains(t, err, "not both")
			_, err = h.ValidateUpdate(t.Context(), nil, in)
			require.ErrorContains(t, err, "not both")
			// A spec conflict is no reason to keep the resource.
			_, err = h.ValidateDelete(t.Context(), in)
			require.NoError(t, err)
		})
	}
}
