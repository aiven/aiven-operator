package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	kafkaconnectuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/kafka_connect"
)

// TestCreateEmptyUserConfiguration shouldn't panic
func TestCreateEmptyUserConfiguration(t *testing.T) {
	var uc *kafkaconnectuserconfig.KafkaConnectUserConfig
	m, err := CreateUserConfiguration(uc)
	assert.Nil(t, m)
	assert.NoError(t, err)
}
