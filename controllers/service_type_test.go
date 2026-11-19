// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceVersion(t *testing.T) {
	t.Parallel()

	metadata := map[string]any{
		"valkey_version": "9.1.1",
		"plan_type":      "plan",
		"engine_version": 2, // non-string values must not panic
	}

	assert.Equal(t, "9.1.1", serviceVersion(serviceTypeValkey, metadata))
	assert.Empty(t, serviceVersion(serviceTypePostgreSQL, metadata))
	assert.Empty(t, serviceVersion(serviceTypeKafkaConnect, metadata))
	assert.Empty(t, serviceVersion(serviceTypeValkey, nil))
}
