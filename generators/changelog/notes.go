package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	changelogFile     = "CHANGELOG.md"
	placeholderHeader = "## [MAJOR.MINOR.PATCH] - YYYY-MM-DD"
)

var (
	errNoSection    = errors.New("no changelog section for this version")
	errEmptySection = errors.New("changelog section is empty")
)

// extractNotes returns the body of version's section, without its header.
func extractNotes(body []byte, version string) (string, error) {
	reHeader := regexp.MustCompile(`^## ` + regexp.QuoteMeta(version) + ` `)

	var section []string
	found := false

	for line := range strings.SplitSeq(string(body), "\n") {
		if !found {
			found = reHeader.MatchString(line)
			continue
		}

		// Any subsequent header ends the section: the next release or the placeholder.
		if strings.HasPrefix(line, "## ") {
			break
		}

		section = append(section, line)
	}

	if !found {
		return "", errNoSection
	}

	notes := strings.Trim(strings.Join(section, "\n"), "\n")
	if strings.TrimSpace(notes) == "" {
		return "", errEmptySection
	}

	return notes, nil
}

// readNotes reads filePath and returns version's section.
func readNotes(filePath, version string) (string, error) {
	body, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	notes, err := extractNotes(body, version)
	switch {
	case errors.Is(err, errNoSection):
		return "", fmt.Errorf("no %q header in %s, did the version bump run", "## "+version, filePath)
	case errors.Is(err, errEmptySection):
		return "", fmt.Errorf(
			"changelog section for %s is empty, add entries under the %q placeholder in %s before releasing",
			version, placeholderHeader, filePath,
		)
	case err != nil:
		return "", err
	}

	return notes, nil
}
