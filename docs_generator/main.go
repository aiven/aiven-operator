package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

const (
	// srcDirPath CRDs location
	srcDirPath = "./config/crd/bases/"
	// dstDirPath CRDs docs location to export to
	dstDirPath  = "./docs/docs/api-reference/"
	examplesDir = dstDirPath + "examples"
	// maxPatternLength some patterns too long, looks not cool in docs, drops long patterns
	maxPatternLength = 42
	// minHeaderLevel h2
	minHeaderLevel = 2
)

func main() {
	err := exec(srcDirPath, dstDirPath)
	if err != nil {
		log.Fatal(err)
	}
}

func exec(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		err = generate(path.Join(srcDir, entry.Name()), dstDir, examplesDir)
		if err != nil {
			return fmt.Errorf("%q generation error: %w", entry.Name(), err)
		}
	}

	return nil
}

// emptyLinesRe finds multiple newlines
var emptyLinesRe = regexp.MustCompile(`\n{2,}`)

// generate Creates documentation out of CRDs
func generate(srcFile, dstDir, examplesDir string) error {
	crdData, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}

	var crd crdType
	err = yaml.Unmarshal(crdData, &crd)
	if err != nil {
		return err
	}

	if l := len(crd.Spec.Versions); l != 1 {
		return fmt.Errorf("version count must be exactly one, got %d", l)
	}

	// Root level object is composed of disparate fields
	kind := crd.Spec.Versions[0].Schema.OpenAPIV3Schema
	kind.Kind = crd.Spec.Names.Kind
	kind.Version = crd.Spec.Versions[0].Name
	kind.Group = crd.Spec.Group
	kind.Name = "Schema"

	// Those fields are generic, but can have only explicit values
	kind.Properties["apiVersion"].Description = fmt.Sprintf("Must be equal to `%s/%s`", kind.Group, kind.Version)
	kind.Properties["kind"].Description = fmt.Sprintf("Must be equal to `%s`", kind.Kind)
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
			return err
		}
		kind.UsageExample = string(exampleData)
	}

	var buf bytes.Buffer
	t := template.Must(template.New("schema").Funcs(templateFuncs).Parse(schemaTemplate))
	err = t.Execute(&buf, kind)
	if err != nil {
		return err
	}

	// Replaces multilines made by template engine
	dest := path.Join(dstDir, strings.ToLower(kind.Kind)+".md")
	data := emptyLinesRe.ReplaceAll(buf.Bytes(), []byte("\n\n"))
	err = os.WriteFile(dest, data, 0644)
	if err != nil {
		return err
	}
	return nil
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
	parent     *schemaType   // Parent line
	properties []*schemaType // Sorted Properties
	isRequired bool

	// Meta data for rendering
	Kind         string // CRD Kind
	Name         string // field name
	Version      string // API version, like v1alpha
	Group        string // API group, like aiven.io
	Level        int    // For header level
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
	Minimum                *int                   `yaml:"minimum"`
	Maximum                *int                   `yaml:"maximum"`
	MinItems               *int                   `yaml:"minItems"`
	MaxItems               *int                   `yaml:"maxItems"`
	MinLength              *int                   `yaml:"minLength"`
	MaxLength              *int                   `yaml:"maxLength"`
	XKubernetesValidations []struct {
		Rule string `yaml:"rule"`
	} `yaml:"x-kubernetes-validations"`
}

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
		s.Items.Level = s.Level
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
		v.Level = s.Level + 1
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
	return s.Kind != ""
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

// GetLinkTag renders a link with a name of if
func (s *schemaType) GetLinkTag() string {
	return fmt.Sprintf("[`%[1]s`](#%[2]s-property){: name='%[2]s-property'}", s.Name, s.GetID())
}

func (s *schemaType) GetHeader() string {
	// Flattens TOC, puts first two levels on the same level
	// otherwise it starts with "spec" and then drills down to the atoms.
	// And that makes TOC navigation useless
	level := s.Level
	if level < minHeaderLevel {
		level = minHeaderLevel
	}
	return fmt.Sprintf("%s %s {: #%s }", strings.Repeat("#", level), s.Name, s.GetID())
}

func (s *schemaType) GetID() string {
	if s.parent == nil || s.parent.IsKind() {
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
		chunks = append(chunks, fmt.Sprintf("Minimum: %d", *s.Minimum))
	}
	if s.Maximum != nil {
		chunks = append(chunks, fmt.Sprintf("Maximum: %d", *s.Maximum))
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
- {{ .GetLinkTag }} ({{ .GetDef }}). {{ .GetDescription }}{{ if .IsNested }} See below for [nested schema](#{{ .GetID }}).{{ end }}
{{ end }}
`
