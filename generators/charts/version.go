package main

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

type chartYaml struct {
	APIVersion  string `yaml:"apiVersion"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Version     string `yaml:"version"`
	AppVersion  string `yaml:"appVersion"`
	Maintainers []struct {
		Name string `yaml:"name"`
		URL  string `yaml:"url"`
	} `yaml:"maintainers"`
}

// updateVersion updates version in Chart.yaml files and the changelog
func updateVersion(version, operatorPath string, charts ...string) error {
	for _, c := range charts {
		err := updateChartVersion(version, path.Join(c, "Chart.yaml"))
		if err != nil {
			return err
		}
	}
	return updateChangelogVersion(version, path.Join(operatorPath, changelogFile))
}

// updateChartVersion replaces version in Chart.yaml
func updateChartVersion(version, chartPath string) error {
	file, err := os.ReadFile(chartPath)
	if err != nil {
		return err
	}

	c := new(chartYaml)
	err = yaml.Unmarshal(file, c)
	if err != nil {
		return err
	}

	// Replaces version with operator's version
	c.Version = version
	c.AppVersion = version

	b, err := marshalCompactYaml(c)
	if err != nil {
		return err
	}
	return writeFile(chartPath, b.Bytes())
}

// reVersionHeader finds headers
var reVersionHeader = regexp.MustCompile(`(?m)^(##.+?)$`)

const (
	headerVersionPlaceholder = "## [MAJOR.MINOR.PATCH] - YYYY-MM-DD"
	headerVersionDateFormat  = "2006-01-02"
)

// updateChangelogVersion updates CHANGELOG.md, sets version header.
// If version already exists, updates release date
func updateChangelogVersion(version, filePath string) error {
	f, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	changelog := string(f)
	alreadyExists := strings.Contains(changelog, "## "+version)
	versionHeader := fmt.Sprintf("## %s - %s", version, time.Now().Format(headerVersionDateFormat))

	changelog = reVersionHeader.ReplaceAllStringFunc(changelog, func(s string) string {
		// If a placeholder found adds version header
		if !alreadyExists && s == headerVersionPlaceholder {
			return fmt.Sprintf("%s\n\n%s", s, versionHeader)
		}

		// If existing version found, the date is updated
		if alreadyExists && strings.Contains(s, version) {
			return versionHeader
		}

		return s
	})

	return writeFile(filePath, []byte(changelog))
}
