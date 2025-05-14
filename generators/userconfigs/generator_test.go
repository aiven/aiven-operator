package main

import (
	"os"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserConfigFile(t *testing.T) {
	src, err := os.ReadFile(`generator_test_source.yml`)
	require.NoError(t, err)

	obj := new(object)
	err = yaml.Unmarshal(src, obj)
	require.NoError(t, err)

	expected, err := os.ReadFile(`pg/pg.go`)
	require.NoError(t, err)

	actual, err := newUserConfigFile("pg_test", obj)
	require.NoError(t, err)

	// Leave the var for debugging with a break point
	expectedStr := string(expected)
	actualStr := string(actual)
	assert.Equal(t, expectedStr, actualStr)
}
