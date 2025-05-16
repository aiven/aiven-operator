package main

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
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

type validation struct {
	Message string `yaml:"message"`
	Rule    string `yaml:"rule"`
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
	Validations []validation       `yaml:"x-kubernetes-validations"`
}

func loadSchema(b []byte) (*schema, error) {
	crd := new(crdType)
	err := yaml.Unmarshal(b, crd)
	if err != nil {
		return nil, err
	}

	if crd.Spec.Versions == nil {
		return nil, nil
	}

	s := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	s.Kind = crd.Spec.Names.Kind
	return s, nil
}

// genChangelog generates changelog for given yamls
func genChangelog(wasBytes, hasBytes []byte) ([]changelog, error) {
	wasSchema, err := loadSchema(wasBytes)
	if err != nil {
		return nil, err
	}

	hasSchema, err := loadSchema(hasBytes)
	if err != nil {
		return nil, err
	}

	if wasSchema == nil || hasSchema == nil {
		c := changelog{value: fmt.Sprintf("`%s`", hasSchema.Kind)}

		if wasSchema == nil {
			c.title = "Add kind"
			return []changelog{c}, nil
		}

		if hasSchema == nil {
			c.title = "Remove kind"
			return []changelog{c}, nil
		}

		return nil, fmt.Errorf("empty schemas")
	}

	changes := cmpSchemas(hasSchema.Kind, "", wasSchema, hasSchema)
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].title < changes[j].title
	})

	return changes, nil
}

type changelog struct {
	title string
	value string
}

func cmpSchemas(kind, parent string, wasSpec, hasSpec *schema) []changelog {
	if cmp.Equal(wasSpec, hasSpec) {
		return nil
	}

	changes := make([]changelog, 0)
	for _, k := range mergedKeys(wasSpec.Properties, hasSpec.Properties) {
		ov, oOk := wasSpec.Properties[k]
		nv, nOk := hasSpec.Properties[k]

		fieldPath := k
		if parent != "" {
			fieldPath = fmt.Sprintf("%s.%s", parent, k)
		}

		switch {
		case !nOk:
			title := fmt.Sprintf("Remove `%s` field `%s`, type `%s`", kind, fieldPath, ov.Type)
			changes = append(changes, changelog{title: title, value: shortDescription(ov.Description)})
		case !oOk:
			title := fmt.Sprintf("Add `%s` field `%s`, type `%s`", kind, fieldPath, nv.Type)
			changes = append(changes, changelog{title: title, value: shortDescription(nv.Description)})
		case !cmp.Equal(ov, nv):
			c := fmtChanges(ov, nv)
			if c != "" {
				title := fmt.Sprintf("Change `%s` field `%s`", kind, fieldPath)
				changes = append(changes, changelog{title: title, value: c})
			}
			if ov.Type == "object" {
				changes = append(changes, cmpSchemas(kind, fieldPath, ov, nv)...)
			}
		}
	}
	return changes
}

const (
	immutableRule = "self == oldSelf"
)

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
	changes[fmtChange("immutable", hasRule(was, immutableRule), hasRule(has, immutableRule))] = true

	if c := cmpList(was.Enum, has.Enum); c != "" {
		changes["enum "+c] = true
	}

	if !isDeprecated(was.Description) && isDeprecated(has.Description) {
		changes[deprecatedMark] = true
	}

	delete(changes, "")
	return joinSorted(maps.Keys(changes))
}

// fmtChange returns a string like: foo ~~`0`~~ → `1` or empty string
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
		return fmt.Sprintf("~~%s~~", was)
	}
	return fmt.Sprintf("~~%s~~%s%s", was, sep, has)
}

func strSlice(src []any) string {
	result := make([]string, len(src))
	for i, v := range src {
		result[i] = fmt.Sprint(v)
	}
	s := joinSorted(result)
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

func addChanges(body []byte, changes []changelog) string {
	strBody := string(body)
	lines := strings.Split(strBody, "\n")

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
outer:
	for _, s := range changes {
		line := softWrapLine(fmt.Sprintf("- %s: %s", s.title, s.value), "\n  ", lineWidth)
		// Replaces a change with the same title to store latest change only
		for index, old := range items {
			if strings.Contains(old, s.title) {
				items[index] = line
				continue outer
			}
		}
		items = append(items, line)
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
		changes := make([]changelog, 0)
		for _, k := range sortedKeys(hasFiles) {
			kindChanges, err := genChangelog(wasFiles[k], hasFiles[k])
			if err != nil {
				return err
			}
			changes = append(changes, kindChanges...)
		}

		if len(changes) == 0 {
			return nil
		}

		// Reads changelogFile
		changelogPath := path.Join(operatorPath, changelogFile)
		changelogBody, err := os.ReadFile(changelogPath)
		if err != nil {
			return err
		}

		// Writes changes to changelogFile
		changelogUpdated := addChanges(changelogBody, changes)
		return os.WriteFile(changelogPath, []byte(changelogUpdated), 0o644)
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

func hasRule(s *schema, rule string) *bool {
	for _, v := range s.Validations {
		if v.Rule == rule {
			t := true
			return &t
		}
	}
	return nil
}

func cmpList[T any](was, have []T) string {
	const (
		remove int = 1 << iota
		add
	)

	seen := make(map[string]int)
	for _, v := range was {
		seen[fmt.Sprintf("`%v`", v)] = remove
	}

	for _, v := range have {
		k := fmt.Sprintf("`%v`", v)
		seen[k] |= add
	}

	var added, removed []string
	for k, v := range seen {
		switch v {
		case add:
			added = append(added, k)
		case remove:
			removed = append(removed, k)
		}
	}

	result := make([]string, 0)
	if s := joinSorted(added); s != "" {
		result = append(result, "add "+s)
	}

	if s := joinSorted(removed); s != "" {
		result = append(result, "remove "+s)
	}

	return joinSorted(result)
}

func joinSorted(args []string) string {
	sort.Strings(args)
	return strings.Join(args, ", ")
}
