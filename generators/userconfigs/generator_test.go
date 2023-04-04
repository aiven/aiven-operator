package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestNewUserConfigFile(t *testing.T) {
	src, err := os.ReadFile(`generator_test_source.yml`)
	assert.NoError(t, err)

	obj := new(object)
	err = yaml.Unmarshal(src, obj)
	assert.NoError(t, err)

	expected, err := os.ReadFile(`pg/pg.go`)
	assert.NoError(t, err)

	actual, err := newUserConfigFile("pg_test", obj)
	assert.NoError(t, err)

	// Leave the var for debugging with a break point
	expectedStr := string(expected)
	actualStr := string(actual)
	assert.Equal(t, expectedStr, actualStr)
}
