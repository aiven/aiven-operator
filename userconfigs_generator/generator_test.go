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

	expected, err := os.ReadFile(`generator_test_expected.go`)
	assert.NoError(t, err)

	file, err := newUserConfigFile("pg_user_config", obj)
	assert.NoError(t, err)

	// Result is a go file, but marked to be ignored
	ignore := "//go:build exclude\n"
	expectedStr := string(expected)[len(ignore):]

	// Leave the var for debugging with a break point
	actual := file.GoString()
	assert.Equal(t, expectedStr, actual)
}
