package main

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

const (
	lineWidth      = 100
	deprecatedMark = "deprecated"
	changelogFile  = "CHANGELOG.md"
)

// crdType kubernetes CRD representation
type crdType struct {
	Spec struct {
		Names struct {
			Kind string `yaml:"kind"`
		} `yaml:"names"`
		Versions []struct {
			Schema struct {
				OpenAPIV3Schema *schema `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

type schema struct {
	Kind        string             `yaml:"-"`
	Properties  map[string]*schema `yaml:"properties"`
	Type        string             `yaml:"type"`
	Description string             `yaml:"description"`
	Enum        []any              `yaml:"enum"`
	Pattern     string             `yaml:"pattern"`
	Format      string             `yaml:"format"`
	MinItems    *int               `yaml:"minItems"`
	MaxItems    *int               `yaml:"maxItems"`
	MinLength   *int               `yaml:"minLength"`
	MaxLength   *int               `yaml:"maxLength"`
	Minimum     *float64           `yaml:"minimum"`
	Maximum     *float64           `yaml:"maximum"`
}

func loadSchema(b []byte) (*schema, error) {
	crd := new(crdType)
	err := yaml.Unmarshal(b, crd)
	if err != nil {
		return nil, err
	}

	if crd.Spec.Versions == nil {
		return nil, fmt.Errorf("empty schema for kind %s", crd.Spec.Names.Kind)
	}

	s := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	s.Kind = crd.Spec.Names.Kind
	return s, nil
}

// genChangelog generates changelog for given yamls
func genChangelog(wasBytes, hasBytes []byte) ([]string, error) {
	wasSchema, err := loadSchema(wasBytes)
	if err != nil {
		return nil, err
	}

	hasSchema, err := loadSchema(hasBytes)
	if err != nil {
		return nil, err
	}

	changes := cmpSchemas(hasSchema.Kind, "", wasSchema, hasSchema)
	sort.Slice(changes, func(i, j int) bool {
		return changes[i][0] < changes[j][0]
	})

	return changes, nil
}

func cmpSchemas(kind, parent string, wasSpec, hasSpec *schema) []string {
	if cmp.Equal(wasSpec, hasSpec) {
		return nil
	}

	changes := make([]string, 0)
	for _, k := range mergedKeys(wasSpec.Properties, hasSpec.Properties) {
		ov, oOk := wasSpec.Properties[k]
		nv, nOk := hasSpec.Properties[k]

		fieldPath := k
		if parent != "" {
			fieldPath = fmt.Sprintf("%s.%s", parent, k)
		}

		switch {
		case !nOk:
			changes = append(changes, fmt.Sprintf("Remove `%s` field `%s`, type `%s`: %s", kind, fieldPath, ov.Type, shortDescription(ov.Description)))
		case !oOk:
			changes = append(changes, fmt.Sprintf("Add `%s` field `%s`, type `%s`: %s", kind, fieldPath, nv.Type, shortDescription(nv.Description)))
		case !cmp.Equal(ov, nv):
			switch ov.Type {
			case "object":
				changes = append(changes, cmpSchemas(kind, fieldPath, ov, nv)...)
			default:
				c := fmtChanges(ov, nv)
				if c != "" {
					changes = append(changes, fmt.Sprintf("Change `%s` field `%s`: %s", kind, fieldPath, c))
				}
			}
		}
	}
	return changes
}

func fmtChanges(was, has *schema) string {
	changes := make(map[string]bool)
	changes[fmtChange("pattern", &was.Pattern, &has.Pattern)] = true
	changes[fmtChange("format", &was.Format, &has.Format)] = true
	changes[fmtChange("minItems", was.MinItems, has.MinItems)] = true
	changes[fmtChange("maxItems", was.MaxItems, has.MaxItems)] = true
	changes[fmtChange("minLength", was.MinLength, has.MinLength)] = true
	changes[fmtChange("maxLength", was.MaxLength, has.MaxLength)] = true
	changes[fmtChange("minimum", was.Minimum, has.Minimum)] = true
	changes[fmtChange("maximum", was.Maximum, has.Maximum)] = true
	changes[fmtChange("enum", &was.Enum, &has.Enum)] = true

	if !isDeprecated(was.Description) && isDeprecated(has.Description) {
		changes[deprecatedMark] = true
	}

	delete(changes, "")
	return strings.Join(sortedKeys(changes), ", ")
}

// fmtChange returns a string like: foo ~`0`~ → `1` or empty string
func fmtChange[T any](title string, was, has *T) string {
	if cmp.Equal(was, has) {
		return ""
	}

	var w, h string
	if was != nil {
		if v := strAny(*was); v != "" {
			w = fmt.Sprintf("`%v`", v)
		}
	}
	if has != nil {
		if v := strAny(*has); v != "" {
			h = fmt.Sprintf("`%v`", v)
		}
	}

	return fmt.Sprintf("%s %s", title, fmtWasHas(" → ", w, h))
}

func fmtWasHas(sep, was, has string) string {
	switch "" {
	case was:
		return has
	case has:
		return fmt.Sprintf("~%s~", was)
	}
	return fmt.Sprintf("~%s~%s%s", was, sep, has)
}

func strSlice(src []any) string {
	result := make([]string, len(src))
	for i, v := range src {
		result[i] = fmt.Sprint(v)
	}
	slices.Sort(result)
	s := strings.Join(result, ", ")
	if s != "" {
		s = fmt.Sprintf("[%s]", s)
	}
	return s
}

func strAny(a any) string {
	switch v := a.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case []any:
		return strSlice(v)
	default:
		return fmt.Sprint(v)
	}
}

func isDeprecated(s string) bool {
	return strings.HasPrefix(strings.ToLower(s), deprecatedMark)
}

// shortDescription returns a string shorter than lineWidth when possible.
func shortDescription(s string) string {
	chunks := strings.Split(s, ". ")
	description := chunks[0]
	for i := 1; i < len(chunks); i++ {
		d := fmt.Sprintf("%s. %s", description, chunks[i])
		if len(d) > lineWidth {
			break
		}
		description = d
	}
	return strings.TrimSuffix(description, ".")
}

// softWrapLineToleranceFactor a line shorter than this factor won't be wrapped
// to not break it right before a small word
const softWrapLineToleranceFactor = 1.1

// softWrapLine wraps long lines
func softWrapLine(src, linebreak string, n int) string {
	if int(float64(n)*softWrapLineToleranceFactor) > len(src) {
		return src
	}

	line := 1 // line number
	for i := 0; i < len(src); {
		s := src[i]
		// 32 ASCII number for space
		if i >= n*line && i%n >= 0 && s == 32 {
			src = fmt.Sprintf("%s%s%s", src[:i], linebreak, src[i+1:])
			i += len(linebreak)
			line++
			continue
		}
		i++
	}
	return src
}

func addChanges(body []byte, changes []string) string {
	lines := strings.Split(string(body), "\n")

	i := 0
	headerAt := -1
	for ; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "##") {
			if headerAt != -1 {
				i--
				break
			}
			headerAt = i
		}
	}

	items := make([]string, 0)
	items = append(items, lines[:i]...)
	if i-headerAt == 1 {
		// adds extra line between headers (no changes in the changelog)
		items = append(items, "")
	}
	for _, s := range changes {
		items = append(items, softWrapLine("- "+s, "\n  ", lineWidth))
	}
	items = append(items, lines[i:]...)
	return strings.Join(items, "\n")
}

// updateChangelog updates changelog with CRD changes.
// To do so, it reads into memory files before they changed.
// Then compares with updates and finds the changes.
func updateChangelog(operatorPath, crdCharts string) (func() error, error) {
	crdDir := path.Join(crdCharts, crdDestinationDir)
	wasFiles, err := readFiles(crdDir)
	if err != nil {
		return nil, err
	}

	return func() error {
		hasFiles, err := readFiles(crdDir)
		if err != nil {
			return err
		}

		// Finds changes per Kind
		changes := make([]string, 0)
		for _, k := range sortedKeys(hasFiles) {
			kindChanges, err := genChangelog(wasFiles[k], hasFiles[k])
			if err != nil {
				return err
			}
			changes = append(changes, kindChanges...)
		}

		// Reads changelogFile
		changelogPath := path.Join(operatorPath, changelogFile)
		changelogBody, err := os.ReadFile(changelogPath)
		if err != nil {
			return err
		}

		// Writes changes to changelogFile
		changelogUpdated := addChanges(changelogBody, changes)
		return os.WriteFile(changelogPath, []byte(changelogUpdated), 0644)
	}, nil
}

func readFiles(p string) (map[string][]byte, error) {
	files, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		b, err := os.ReadFile(path.Join(p, file.Name()))
		if err != nil {
			return nil, err
		}
		result[file.Name()] = b
	}

	return result, nil
}

// sortedKeys returns map's keys sorted to have predictable output
func sortedKeys[K constraints.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// mergedKeys returns merged keys of multiple maps
func mergedKeys[K constraints.Ordered, V any](maps ...map[K]V) []K {
	unique := make(map[K]bool)
	for _, m := range maps {
		for k := range m {
			unique[k] = true
		}
	}
	return sortedKeys[K](unique)
}
