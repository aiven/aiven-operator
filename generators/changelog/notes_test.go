package main

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shaped like the real file: placeholder on top, releases newest first, and v0.4.10 sitting above v0.4.1.
const testChangelog = `# Changelog

` + placeholderHeader + `

- an unreleased entry

## v0.44.0 - 2026-08-11

- Add kind: ` + "`OrganizationProject`" + `
- Fix multi-line
  entries that wrap

## v0.4.10 - 2024-01-02

- ten

## v0.4.1 - 2024-01-01

- one

## v0.3.0 - 2023-12-01
`

func TestExtractNotes(t *testing.T) {
	cases := []struct {
		name      string
		version   string
		changelog string
		expect    string
		expectErr error
	}{
		{
			name:      "reads a section, header and blank lines stripped",
			version:   "v0.44.0",
			changelog: testChangelog,
			expect: "- Add kind: `OrganizationProject`\n" +
				"- Fix multi-line\n" +
				"  entries that wrap",
		},
		{
			name:      "v0.4.1 does not match the v0.4.10 header",
			version:   "v0.4.1",
			changelog: testChangelog,
			expect:    "- one",
		},
		{
			name:      "v0.4.10 reads its own section",
			version:   "v0.4.10",
			changelog: testChangelog,
			expect:    "- ten",
		},
		{
			// Dots must not act as wildcards.
			name:      "dots are literal, not any-character",
			version:   "v0x44x0",
			changelog: testChangelog,
			expectErr: errNoSection,
		},
		{
			name:      "the placeholder terminates the section above it",
			version:   "v0.44.0",
			changelog: "## v0.44.0 - 2026-08-11\n\n- kept\n\n" + placeholderHeader + "\n\n- not kept\n",
			expect:    "- kept",
		},
		{
			name:      "last section in the file, no following header",
			version:   "v0.3.0",
			changelog: "## v0.3.0 - 2023-12-01\n\n- final\n",
			expect:    "- final",
		},
		{
			name:      "missing version",
			version:   "v9.9.9",
			changelog: testChangelog,
			expectErr: errNoSection,
		},
		{
			name:      "header present but nothing under it",
			version:   "v0.45.0",
			changelog: "## v0.45.0 - 2026-08-13\n\n## v0.44.0 - 2026-08-11\n\n- old\n",
			expectErr: errEmptySection,
		},
		{
			name:      "whitespace-only section counts as empty",
			version:   "v0.45.0",
			changelog: "## v0.45.0 - 2026-08-13\n\n   \n\t\n\n## v0.44.0 - 2026-08-11\n",
			expectErr: errEmptySection,
		},
		{
			name:      "the placeholder's own entries are readable by its header",
			version:   "[MAJOR.MINOR.PATCH]",
			changelog: testChangelog,
			expect:    "- an unreleased entry",
		},
	}

	for _, opt := range cases {
		t.Run(opt.name, func(t *testing.T) {
			actual, err := extractNotes([]byte(opt.changelog), opt.version)
			if opt.expectErr != nil {
				require.ErrorIs(t, err, opt.expectErr)
				assert.Empty(t, actual)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, opt.expect, actual)
		})
	}
}

func TestParsesWhatTheChartsGeneratorWrites(t *testing.T) {
	written := "# Changelog\n\n" + placeholderHeader + "\n\n## v1.2.3 - 2026-08-13\n\n- a new entry\n\n## v1.2.2 - 2020-01-01\n\n- older\n"

	notes, err := extractNotes([]byte(written), "v1.2.3")
	require.NoError(t, err)
	assert.Equal(t, "- a new entry", notes)

	// The placeholder survives the bump, empty, ready for the next cycle.
	_, err = extractNotes([]byte(written), "[MAJOR.MINOR.PATCH]")
	require.ErrorIs(t, err, errEmptySection)
}

func TestReadNotes(t *testing.T) {
	file := path.Join(t.TempDir(), changelogFile)
	require.NoError(t, os.WriteFile(file, []byte(testChangelog), 0o644))

	notes, err := readNotes(file, "v0.44.0")
	require.NoError(t, err)
	assert.Contains(t, notes, "OrganizationProject")

	_, err = readNotes(file, "v9.9.9")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no "## v9.9.9" header`)
	assert.Contains(t, err.Error(), "did the version bump run")

	_, err = readNotes(path.Join(t.TempDir(), "missing.md"), "v1.0.0")
	require.ErrorIs(t, err, os.ErrNotExist)
}
