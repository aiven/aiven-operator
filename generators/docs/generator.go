package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

const (
	// maxPatternLength some patterns too long, looks not cool in docs, drops long patterns
	maxPatternLength = 42
	// minHeaderLevel doesn't let print headers less than
	minHeaderLevel = 2
)

// emptyLinesRe finds multiple newlines
var emptyLinesRe = regexp.MustCompile(`\n{2,}`)

// parseSchema creates documentation out of CRD
func parseSchema(srcFile, dstDir, examplesDir string) (*schemaType, error) {
	crdData, err := os.ReadFile(srcFile)
	if err != nil {
		return nil, err
	}

	var crd crdType
	err = yaml.Unmarshal(crdData, &crd)
	if err != nil {
		return nil, err
	}

	if l := len(crd.Spec.Versions); l != 1 {
		return nil, fmt.Errorf("version count must be exactly one, got %d", l)
	}

	// Root level object is composed of disparate fields
	kind := crd.Spec.Versions[0].Schema.OpenAPIV3Schema
	kind.Kind = crd.Spec.Names.Kind
	kind.Name = kind.Kind
	kind.Version = crd.Spec.Versions[0].Name
	kind.Group = crd.Spec.Group

	// Those fields are generic, but can have only explicit values
	kind.Properties["apiVersion"].Description = fmt.Sprintf("Value `%s/%s`", kind.Group, kind.Version)
	kind.Properties["kind"].Description = fmt.Sprintf("Value `%s`", kind.Kind)
	kind.Properties["metadata"].Description = "Data that identifies the object, including a `name` string and optional `namespace`"

	// Status is a meta field, brings nothing important to docs
	delete(kind.Properties, "status")
	kind.init()

	// Adds usage example
	// Mkdocs can embed files, but we use docker, we can include files only within the dir
	// So if they are moved out elsewhere, this will be broken
	// Ignores errors, because does "not exist" is not a problem
	examplePath := path.Join(examplesDir, strings.ToLower(kind.Kind+".yaml"))
	exampleData, _ := os.ReadFile(examplePath)
	if exampleData != nil {
		err = validateYAML(crdData, exampleData)
		if err != nil {
			return nil, err
		}
		kind.UsageExample = string(exampleData)
	}

	return kind, nil
}

func renderTemplate(tmpl string, in any) ([]byte, error) {
	var buf bytes.Buffer
	t := template.Must(template.New("").Funcs(templateFuncs).Parse(tmpl))
	err := t.Execute(&buf, in)
	if err != nil {
		return nil, err
	}
	return fixNewlines(buf.Bytes()), nil
}

// fixNewlines replaces multilines made by template engine
func fixNewlines(b []byte) []byte {
	return emptyLinesRe.ReplaceAll(b, []byte("\n\n"))
}

// crdType CRD doc type
type crdType struct {
	Spec struct {
		Group string `yaml:"group"`
		Names struct {
			Kind string `yaml:"kind"`
		} `yaml:"names"`
		Versions []struct {
			Name   string `yaml:"name"`
			Schema struct {
				OpenAPIV3Schema *schemaType `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

type schemaInternal struct {
	// Internal
	parent     *schemaType   // parent line
	properties []*schemaType // Sorted Properties
	isRequired bool
	level      int // For header level

	// Meta data for rendering
	Kind         string // CRD Kind
	Name         string // field name
	Version      string // API version, like v1alpha
	Group        string // API group, like aiven.io
	UsageExample string
}

type schemaType struct {
	schemaInternal

	// Yaml fields
	AdditionalProperties *struct {
		Type string `yaml:"type"`
	} `yaml:"additionalProperties"`
	Description            string                 `yaml:"description"`
	Properties             map[string]*schemaType `yaml:"properties"`
	Items                  *schemaType            `yaml:"items"`
	RequiredList           []string               `yaml:"required"`
	Type                   string                 `yaml:"type"`
	Format                 string                 `yaml:"format"`
	Enum                   []string               `yaml:"enum"`
	Pattern                string                 `yaml:"pattern"`
	Minimum                *float64               `yaml:"minimum"`
	Maximum                *float64               `yaml:"maximum"`
	MinItems               *int                   `yaml:"minItems"`
	MaxItems               *int                   `yaml:"maxItems"`
	MinLength              *int                   `yaml:"minLength"`
	MaxLength              *int                   `yaml:"maxLength"`
	XKubernetesValidations []struct {
		Rule string `yaml:"rule"`
	} `yaml:"x-kubernetes-validations"`
}

// propsType says if object property is optional or required
type propsType int

const (
	propsAll propsType = iota
	propsRequired
	propsOptional
)

func (s *schemaType) init() {
	if s.Items != nil && s.Items.IsNested() {
		// If item is an object, then it doesn't have a name
		// And we want to render it as topper level block
		s.Items.Name = s.Name
		s.Items.level = s.level
		s.Items.parent = s.parent
		s.Items.Description = s.Description
		s.Items.init()
	}

	// The field is required if:
	// - it is specified in required list
	// - it is the only prop object has
	// - it is root object 	https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields
	allRequired := len(s.Properties) == 1 || s.IsKind()
	for k, v := range s.Properties {
		v.Name = k
		v.level = s.level + 1
		v.parent = s
		v.isRequired = allRequired || slices.Contains(s.RequiredList, v.Name)
		v.init()
		s.properties = append(s.properties, v)
	}

	slices.SortFunc(s.properties, func(a, b *schemaType) bool {
		return a.Name < b.Name
	})

}

// ListProperties lists all object properties
func (s *schemaType) ListProperties(flag propsType) []*schemaType {
	if flag == propsAll {
		return s.properties
	}

	result := make([]*schemaType, 0)
	wantRequired := flag == propsRequired
	for _, v := range s.properties {
		if v.isRequired == wantRequired {
			result = append(result, v)
		}
	}
	return result
}

// ListRequired returns required field. If object has the only field, then it is required too
func (s *schemaType) ListRequired() []*schemaType {
	return s.ListProperties(propsRequired)
}

// ListOptional returns optional fields
func (s *schemaType) ListOptional() []*schemaType {
	return s.ListProperties(propsOptional)
}

func (s *schemaType) IsKind() bool {
	return s.parent == nil
}

// IsNested returns true if need to render nested block (an object)
func (s *schemaType) IsNested() bool {
	return len(s.Properties) > 0 || (s.Items != nil && s.Items.IsNested())
}

// GetDescription Returns description with a dot suffix. Fallbacks to items description
func (s *schemaType) GetDescription() string {
	if s.Description == "" {
		return ""
	}

	if strings.HasSuffix(s.Description, ".") {
		return s.Description
	}
	return s.Description + "."
}

// GetPropertyLink renders a link for property list
func (s *schemaType) GetPropertyLink() string {
	return fmt.Sprintf("[`%[1]s`](#%[2]s-property){: name='%[2]s-property'}", s.Name, s.GetID())
}

// GetParentLink returns a link to parent schema
func (s *schemaType) GetParentLink() string {
	if s.IsKind() {
		return ""
	}
	return fmt.Sprintf("_Appears on [`%[1]s`](#%[2]s)._", s.parent.GetID(), s.parent.GetID())
}

// GetHeader renders h2/h3/etc tag
func (s *schemaType) GetHeader() string {
	// Flattens TOC, puts first two levels on the same level
	// otherwise it starts with "spec" and then drills down to the atoms.
	// And that makes TOC navigation useless
	level := s.level
	if level < minHeaderLevel {
		level = minHeaderLevel
	}
	return fmt.Sprintf("%s %s {: #%s }", strings.Repeat("#", level), s.Name, s.GetID())
}

// GetID returns id/path of nested schema without Kind,
// because it is the document itself
func (s *schemaType) GetID() string {
	if s.IsKind() || s.parent.IsKind() {
		return s.Name
	}
	return fmt.Sprintf("%s.%s", s.parent.GetID(), s.Name)
}

func (s *schemaType) GetType() string {
	switch s.Type {
	case "array":
		return fmt.Sprintf("array of %ss", s.Items.GetType())
	default:
		return s.Type
	}
}

// GetDef returns field definition (types and constraints)
func (s *schemaType) GetDef() string {
	chunks := []string{s.GetType()}
	if len(s.Enum) != 0 {
		ems := make([]string, len(s.Enum))
		for i, e := range s.Enum {
			ems[i] = fmt.Sprintf("`%s`", e)
		}
		chunks = append(chunks, "Enum: "+strings.Join(ems, ", "))
	}
	for _, r := range s.XKubernetesValidations {
		if r.Rule == "self == oldSelf" {
			chunks = append(chunks, "Immutable")
		}
	}
	if s.Pattern != "" && len(s.Pattern) <= maxPatternLength {
		// Some patterns are too long
		chunks = append(chunks, fmt.Sprintf("Pattern: `%s`", s.Pattern))
	}
	if s.Minimum != nil {
		chunks = append(chunks, "Minimum: "+prettyDigit(s.Type, *s.Minimum))
	}
	if s.Maximum != nil {
		chunks = append(chunks, "Maximum: "+prettyDigit(s.Type, *s.Maximum))
	}
	if s.MinItems != nil {
		chunks = append(chunks, fmt.Sprintf("MinItems: %d", *s.MinItems))
	}
	if s.MaxItems != nil {
		chunks = append(chunks, fmt.Sprintf("MaxItems: %d", *s.MaxItems))
	}
	if s.MinLength != nil {
		chunks = append(chunks, fmt.Sprintf("MinLength: %d", *s.MinLength))
	}
	if s.MaxLength != nil {
		chunks = append(chunks, fmt.Sprintf("MaxLength: %d", *s.MaxLength))
	}
	if s.AdditionalProperties != nil {
		chunks = append(chunks, fmt.Sprintf("AdditionalProperties: %s", s.AdditionalProperties.Type))
	}

	return strings.Join(chunks, ", ")
}

func (s *schemaType) GetUsageExample() string {
	if s.UsageExample == "" {
		return ""
	}
	return fmt.Sprintf("```yaml\n%s```", s.UsageExample)
}

var templateFuncs = template.FuncMap{
	// Returns a seq of a given size
	"seq": func(i int) []int { return make([]int, i) },
}

var schemaTemplate = `---
title: "{{ .Kind }}"
---

{{ if .GetUsageExample }}
## Usage example

{{ .GetUsageExample }}
{{ end }}

{{- template "renderSchema" . }}

{{ define "renderSchema" }}
{{ if .Items }}
	{{- template "renderSchema" .Items }}
{{ else }}
{{ .GetHeader }}

{{ .GetParentLink }}

{{ .GetDescription }}
{{ end }}

{{ $req := .ListRequired }}
{{ if $req }}
**Required**

{{ range $req }}
	{{- template "renderProp" . }}
{{- end }}
{{ end }}

{{ $opt := .ListOptional }}
{{ if $opt }}
**Optional**

{{ range $opt }}
	{{- template "renderProp" . }}
{{- end }}
{{ end }}

{{ range .ListProperties 0 }}
	{{- if .IsNested }}
		{{- template "renderSchema" . }}
	{{- end }}
{{- end }}
{{ end }}

{{ define "renderProp" -}}
- {{ .GetPropertyLink }} ({{ .GetDef }}). {{ .GetDescription }}{{ if .IsNested }} See below for [nested schema](#{{ .GetID }}).{{ end }}
{{ end }}
`

var (
	// reTrailingZeros finds trailing zeros
	reTrailingZeros = regexp.MustCompile(`0+$`)
)

// prettyDigit formats floats without zero fraction
func prettyDigit(kind string, value float64) string {
	if kind == "integer" {
		return fmt.Sprintf("%d", int(value))
	}

	s := fmt.Sprintf("%.5f", value)
	s = reTrailingZeros.ReplaceAllString(s, "$1")
	return strings.TrimSuffix(s, ".")
}
