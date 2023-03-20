package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	pgtestuserconfig "github.com/aiven/aiven-operator/generators/userconfigs/pg"
)

func TestNewUserConfigFile(t *testing.T) {
	src, err := os.ReadFile(`generator_test_source.yml`)
	assert.NoError(t, err)

	obj := new(object)
	err = yaml.Unmarshal(src, obj)
	assert.NoError(t, err)

	expected, err := os.ReadFile(`pg/pg.go`)
	assert.NoError(t, err)

	actual, err := newUserConfigFile("pg_test_user_config", obj)
	assert.NoError(t, err)

	// Leave the var for debugging with a break point
	expectedStr := string(expected)
	actualStr := string(actual)
	assert.Equal(t, expectedStr, actualStr)
}

func TestIpFilterString(t *testing.T) {
	var c pgtestuserconfig.PgTestUserConfig
	s := `{
		"ip_filter": [
			"foo",
			"bar"
		]
	}`

	err := json.Unmarshal([]byte(s), &c)
	assert.NoError(t, err)
	assert.Len(t, c.IpFilter, 2)
	assert.Equal(t, c.IpFilter[0].Network, "foo")
	assert.Nil(t, c.IpFilter[0].Description)
	assert.Equal(t, c.IpFilter[1].Network, "bar")
	assert.Nil(t, c.IpFilter[1].Description)
}

func TestIpFilterObjects(t *testing.T) {
	var c pgtestuserconfig.PgTestUserConfig
	s := `{
		"ip_filter": [
			{
				"network": "foo",
				"description": "foo description"
			},
			{
				"network": "bar"
			}
		]
	}`

	err := json.Unmarshal([]byte(s), &c)
	assert.NoError(t, err)
	assert.Len(t, c.IpFilter, 2)
	assert.Equal(t, c.IpFilter[0].Network, "foo")
	assert.Equal(t, *c.IpFilter[0].Description, "foo description")
	assert.Equal(t, c.IpFilter[1].Network, "bar")
	assert.Nil(t, c.IpFilter[1].Description)
}

func TestIpFilterEmpty(t *testing.T) {
	var c pgtestuserconfig.PgTestUserConfig
	s := `{
		"ip_filter": []
	}`

	err := json.Unmarshal([]byte(s), &c)
	assert.NoError(t, err)
	assert.Len(t, c.IpFilter, 0)
}
