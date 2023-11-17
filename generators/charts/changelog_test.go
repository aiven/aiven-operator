package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoftWrapLine(t *testing.T) {
	src := "Add `Kafka.userConfig.kafka.sasl_oauthbearer_expected_audience`: The (optional) comma-delimited setting for the broker to use to verify that the JWT was issued for one of the expected audiences."
	expect := "Add `Kafka.userConfig.kafka.sasl_oauthbearer_expected_audience`: The (optional) comma-delimited setting\n  for the broker to use to verify that the JWT was issued for one of the expected audiences."
	assert.Equal(t, expect, softWrapLine(src, "\n  ", 100))
}

const testOldCRD = `
spec:
  names:
    kind: Kafka
  versions:
  - schema:
      openAPIV3Schema:
        properties:
          spec:
            properties:
              cloudName:
                description: Cloud the service runs in.
                maxLength: 120
                type: string
                enum: [foo, bar]
                minimum: 1.0
                maximum: 2.0
              karapace:
                description: Switch the service to use Karapace for schema registry and REST proxy
                type: boolean
              topic_name:
                description: The durable single partition topic
                format: ^[1-9]*(GiB|G)*
                maxLength: 111
                type: string
`

const testNewCRD = `
spec:
  names:
    kind: Kafka
  versions:
  - schema:
      openAPIV3Schema:
        properties:
          spec:
            properties:
              cloudName:
                description: Cloud the service runs in.
                maxLength: 256
                type: string
                enum: [foo, bar, baz]
                minimum: 3.0
                maximum: 4.0
              disk_space:
                description: The disk space of the service.
                type: string
              topic_name:
                description: The durable single partition topic
                format: ^[1-9][0-9]*(GiB|G)*
                maxLength: 249
                minLength: 1
                minimum: 1
                maximum: 1000000
                enum: [foo, bar, baz]
`

func TestGenChangelog(t *testing.T) {
	changes, err := genChangelog([]byte(testOldCRD), []byte(testNewCRD))
	require.NoError(t, err)

	expect := []string{"Add `Kafka` field `disk_space`, type `string`: The disk space of the service",
		"Change `Kafka` field `cloudName`: enum ~~`[bar, foo]`~~ → `[bar, baz, foo]`, maxLength ~~`120`~~ → `256`, maximum ~~`2`~~ → `4`, minimum ~~`1`~~ → `3`",
		"Change `Kafka` field `topic_name`: enum `[bar, baz, foo]`, format ~~`^[1-9]*(GiB|G)*`~~ → `^[1-9][0-9]*(GiB|G)*`, maxLength ~~`111`~~ → `249`, maximum `1000000`, minLength `1`, minimum `1`",
		"Remove `Kafka` field `karapace`, type `boolean`: Switch the service to use Karapace for schema registry and REST proxy",
	}

	assert.Equal(t, expect, changes)
}

func TestAddChanges(t *testing.T) {
	changes := []string{"a change", "another change"}
	cases := []struct {
		name   string
		source string
		expect string
	}{
		{
			name: "add changes to empty list",
			source: `## [MAJOR.MINOR.PATCH] - YYYY-MM-DD

## v0.13.0 - 2023-08-18
`,
			expect: `## [MAJOR.MINOR.PATCH] - YYYY-MM-DD

- a change
- another change

## v0.13.0 - 2023-08-18
`,
		},
		{
			name: "add changes to existing list",
			source: `## [MAJOR.MINOR.PATCH] - YYYY-MM-DD

- new go version

## v0.13.0 - 2023-08-18
`,
			expect: `## [MAJOR.MINOR.PATCH] - YYYY-MM-DD

- new go version
- a change
- another change

## v0.13.0 - 2023-08-18
`,
		},
	}

	for _, opt := range cases {
		t.Run(opt.name, func(t *testing.T) {
			actual := addChanges([]byte(opt.source), changes)
			assert.Equal(t, opt.expect, actual)
		})
	}
}
