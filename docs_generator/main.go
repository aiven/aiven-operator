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
	dstDirPath = "./docs/docs/api-reference/"
	// maxPatternLength some patterns too long, looks not cool in docs, drops long patterns
	maxPatternLength = 42
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

		err = generate(path.Join(srcDir, entry.Name()), dstDir)
		if err != nil {
			return err
		}
	}

	return nil
}

// emptyLinesRe finds multiple newlines
var emptyLinesRe = regexp.MustCompile(`\n{2,}`)

// generate Creates documentation out of CRDs
func generate(srcFile, dstDir string) error {
	b, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}

	var doc crdType
	err = yaml.Unmarshal(b, &doc)
	if err != nil {
		return err
	}

	if l := len(doc.Spec.Versions); l != 1 {
		return fmt.Errorf("version count must be exactly one, got %d", l)
	}

	kind := doc.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties.Spec
	kind.Name = doc.Spec.Names.Kind
	kind.Version = doc.Spec.Versions[0].Name
	kind.Group = doc.Spec.Group

	var buf bytes.Buffer
	t := template.Must(template.New("schema").Funcs(templateFuncs).Parse(schemaTemplate))
	err = t.Execute(&buf, kind)
	if err != nil {
		return err
	}

	// Replaces multilines made by template engine
	dest := path.Join(dstDir, strings.ToLower(kind.Name)+".md")
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
				OpenAPIV3Schema struct {
					Properties struct {
						Spec *schemaType `yaml:"spec"`
					} `yaml:"properties"`
				} `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

type schemaType struct {
	// Internal
	Name    string
	Version string
	Group   string
	Level   int

	// Yaml fields
	Description            string                 `yaml:"description"`
	Properties             map[string]*schemaType `yaml:"properties"`
	Items                  *schemaType            `yaml:"items"`
	Required               []string               `yaml:"required"`
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

// ListProperties lists all object properties
func (s *schemaType) ListProperties(flag propsType) []*schemaType {
	result := make([]*schemaType, 0)
	for k, v := range s.Properties {
		// The filed is required if:
		// - it is specified in required list
		// - it is the only prop object has
		if flag == propsAll || (flag == propsRequired) == (len(s.Properties) == 1 || slices.Contains(s.Required, k)) {
			v.Name = k
			v.Level = s.Level + 1
			result = append(result, v)
		}
	}

	slices.SortFunc(result, func(a, b *schemaType) bool {
		return a.Name < b.Name
	})
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

// IsNested returns if need to render nested block (an object)
func (s *schemaType) IsNested() bool {
	return len(s.Properties) > 0 || (s.Items != nil && s.Items.IsNested())
}

// GetDescription Returns description with a dot suffix. Fallbacks to items description
func (s *schemaType) GetDescription() string {
	if s.Description == "" {
		if s.Items != nil {
			return s.Items.GetDescription()
		}
		return ""
	}

	if strings.HasSuffix(s.Description, ".") {
		return s.Description
	}
	return s.Description + "."
}

// GetNameLink renders a link with a name of if
func (s *schemaType) GetNameLink() string {
	return fmt.Sprintf("[`%[1]s`](#%[1]s){: name='%[1]s'}", s.Name)
}

// GetDef returns field definition (types and constraints)
func (s *schemaType) GetDef() string {
	chunks := []string{s.Type}
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
	return strings.Join(chunks, ", ")
}

var templateFuncs = template.FuncMap{
	// Returns a seq of a given size
	"seq": func(i int) []int { return make([]int, i) },
}

var schemaTemplate = `---
title: "{{ .Name }}"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| {{ .Group }}/{{ .Version }} | {{ .Name }} |

{{ .GetDescription }}

{{ range .ListProperties 0 }}
	{{- template "renderProp" . }}
{{- end }}

{{ range .ListProperties 0 }}
	{{- if .IsNested }}
		{{- template "renderSchema" . }}
	{{- end }}
{{- end }}

{{ define "renderSchema" }}
{{ if .Name }}
#{{ range seq .Level }}#{{ end }} {{ .Name }} {: #{{ .Name }} }
{{ end }}

{{/* This description may go recursively, so we don't render it if it has items  */}}
{{ if .Items }}
	{{- if .Items.IsNested }}
		{{- template "renderSchema" .Items }}
	{{- end }}
{{ else }}
	{{- .GetDescription }}
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
- {{ .GetNameLink }} ({{ .GetDef }}). {{ .GetDescription }} {{ if .IsNested }}See [below for nested schema](#{{ .Name }}).{{ end }}
{{ end }}
`
