package main

import (
	"fmt"
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

	// Result is a go file, but marked to be ignored.
	// Empty lines add to make IDE formatter be happy with that.
	ignore := "//go:build exclude\n\n"
	expectedStr := string(expected)[len(ignore):]

	// Leave the var for debugging with a break point
	actual := file.GoString()
	assert.Equal(t, expectedStr, actual)
}

func TestSafeEnumKeepsOriginal(t *testing.T) {
	cases := []string{
		"1",
		"foo",
		"foo_bar",
		"foo-bar",
		"Foo",
		"foo123",
	}
	for _, s := range cases {
		t.Run(s, func(t *testing.T) {
			assert.Equal(t, s, safeEnum(s))
		})
	}
}

func TestSafeEnumAddsQuotes(t *testing.T) {
	cases := []string{
		"foo%p",
		"foo{}",
		"[foo]",
		"foo bar",
		"foo,bar",
	}
	for _, s := range cases {
		t.Run(s, func(t *testing.T) {
			assert.Equal(t, fmt.Sprintf("%q", s), safeEnum(s))
		})
	}
}
