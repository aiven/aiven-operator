package main

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
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

// reEmptyLines finds multiple newlines
var reEmptyLines = regexp.MustCompile(`\n{2,}`)

// parseSchema creates documentation out of CRD
func parseSchema(crdData []byte) (*schemaType, error) {
	var crd crdType
	err := yaml.Unmarshal(crdData, &crd)
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
	kind.Plural = crd.Spec.Names.Plural
	kind.Columns = crd.Spec.Versions[0].AdditionalPrinterColumns

	// Those fields are generic, but can have only explicit values
	kind.Properties["apiVersion"].Description = fmt.Sprintf("Value `%s/%s`", kind.Group, kind.Version)
	kind.Properties["kind"].Description = fmt.Sprintf("Value `%s`", kind.Kind)
	kind.Properties["metadata"].Description = "Data that identifies the object, including a `name` string and optional `namespace`"

	// Status is a meta field, brings nothing important to docs
	delete(kind.Properties, "status")
	kind.init()

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
	return reEmptyLines.ReplaceAll(b, []byte("\n\n"))
}

// crdType CRD doc type
type crdType struct {
	Spec struct {
		Group string `yaml:"group"`
		Names struct {
			Kind   string `yaml:"kind"`
			Plural string `yaml:"plural"`
		} `yaml:"names"`
		Versions []struct {
			Name   string `yaml:"name"`
			Schema struct {
				OpenAPIV3Schema *schemaType `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
			AdditionalPrinterColumns []specTableColumn `yaml:"additionalPrinterColumns"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

type usageExample struct {
	Title, Value string
}

// specTableColumn the column fields printed on "kubectl get foo name" command
type specTableColumn struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Path string `yaml:"jsonPath"`
}

type schemaInternal struct {
	// Internal
	parent     *schemaType   // parent line
	properties []*schemaType // Sorted Properties
	isRequired bool
	level      int // For header level

	// Meta data for rendering
	Kind          string // CRD Kind
	Name          string // field name
	Version       string // API version, like v1alpha
	Group         string // API group, like aiven.io
	Plural        string
	Columns       []specTableColumn
	UsageExamples []usageExample
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
		if s.Items.Description == "" {
			// Takes parent description if doesn't have one
			s.Items.Description = s.Description
		}
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

	sort.Slice(s.properties, func(i, j int) bool {
		return s.properties[i].Name < s.properties[j].Name
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

var (
	// reInlineCode to wrap inline code into backticks
	reInlineCode = regexp.MustCompile(`'(\S+)'`)

	// reAdmonitions to render markdown admonitions
	reAdmonitions = regexp.MustCompile(`(?mi)^(note|tip|info|warning|danger|bug|success|example)( "[^"]+")*:\s*(.+)$`)
)

// GetDescription Returns description with a dot suffix. Fallbacks to items description
func (s *schemaType) GetDescription() string {
	d := s.Description
	if d == "" {
		return ""
	}

	// Adds trailing dot to the description
	if !strings.HasSuffix(s.Description, ".") {
		d += "."
	}

	// Wraps code chunks with backticks
	d = fmtAdmonitions(d)
	return reInlineCode.ReplaceAllString(d, "`$1`")
}

// fmtAdmonitions formats https://squidfunk.github.io/mkdocs-material/reference/admonitions/
func fmtAdmonitions(src string) string {
	// Marks admonitions
	marked := ReplaceAllStringSubmatchFunc(reAdmonitions, src, func(parts []string) string {
		// i.e.: !!! note
		p := "!!! " + parts[1]

		// If it has a non-empty title, adds it
		if parts[2] != `""` {
			p += parts[2]
		}

		// If an admonition is followed with body, adds a newline
		if parts[3] != "" {
			p += "\n" + parts[3]
		}
		return p
	})

	// Indents body lines
	lines := strings.Split(marked, "\n")
	start := -1
	for i, o := range lines {
		if start < 0 {
			// Finds a block beginning
			if strings.HasPrefix(o, "!!!") {
				start = i
				// Wraps the headline with newlines
				lines[i] = fmt.Sprintf("\n%s\n", o)
			}
			continue
		}

		trimmed := strings.TrimSpace(o)
		if trimmed == "" {
			// End of block
			start = -1
			continue
		}

		// Adds left padding
		lines[i] = "    " + trimmed
	}
	return strings.Join(lines, "\n")
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
	if s.Format != "" && s.Type == "string" {
		chunks = append(chunks, fmt.Sprintf("Format: `%s`", s.Format))
	}
	if s.AdditionalProperties != nil {
		chunks = append(chunks, fmt.Sprintf("AdditionalProperties: %s", s.AdditionalProperties.Type))
	}

	return strings.Join(chunks, ", ")
}

// exampleType example.yaml files model
type exampleType struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	}
	Spec   map[string]any       `yaml:"spec"`
	Table  []exampleTableColumn `yaml:"-"`
	Secret struct {
		Name string
		Keys []string
	} `yaml:"-"`
}

// exampleTableColumn columns and values from the exampleType
type exampleTableColumn struct {
	Title, Value string
}

var reExampleSecretKeys = regexp.MustCompile("(?:`)([A-Z_]+)")

// GetExample returns exampleType with formed output table
func (s *schemaType) GetExample() *exampleType {
	var example *exampleType
	for _, e := range loadYAMLs[exampleType]([]byte(s.UsageExamples[0].Value)) {
		if e.Kind == s.Kind {
			example = e
			break
		}
	}

	if example == nil {
		return nil
	}

	// Adds the Name column
	example.Table = append(example.Table, exampleTableColumn{Title: "Name", Value: example.Metadata.Name})

	// Takes columns from the spec and values from the example
	for _, c := range s.Columns {
		column := exampleTableColumn{}
		if strings.HasPrefix(c.Path, ".spec.") {
			k := strings.TrimPrefix(c.Path, ".spec.")
			column.Value = fmt.Sprintf("%v", example.Spec[k])
		} else {
			switch c.Path {
			case ".status.state":
				column.Value = "RUNNING"
			default:
				column.Value = fmt.Sprintf("<%s>", strings.TrimPrefix(c.Path, ".status."))
			}
		}

		switch column.Value {
		case "", "<nil>":
			// Not set
			continue
		}

		column.Title = c.Name
		example.Table = append(example.Table, column)
	}

	if secret, ok := example.Spec["connInfoSecretTarget"]; ok {
		// The secret's kube name
		example.Secret.Name = secret.(map[string]any)["name"].(string)

		// Secret keys
		for _, m := range reExampleSecretKeys.FindAllStringSubmatch(s.Description, -1) {
			example.Secret.Keys = append(example.Secret.Keys, m[1])
		}
	}

	return example
}

// rfill adds right padding
func rfill(x, y string) string {
	return fmt.Sprintf("%-*s", max(len(x), len(y)), x)
}

var reIndent = regexp.MustCompile("(?m)^")

var templateFuncs = template.FuncMap{
	"indent": func(i int, src string) string {
		return reIndent.ReplaceAllString(src, strings.Repeat(" ", i))
	},
	"code": func(s string) string {
		return fmt.Sprintf("`%s`", s)
	},
	"codeblock": func() string {
		// we can't use backticks in go strings, so we render them
		return "```"
	},
	"rfill": rfill,
}

const schemaTemplate = `---
title: "{{ .Kind }}"
---

{{ if .UsageExamples }}
## Usage example{{ if ne (len .UsageExamples) 1 }}s{{ end }}

{{ range .UsageExamples }}
??? example {{ if .Title }}"{{ .Title }}"{{ end }}
    {{ codeblock }}yaml
{{ .Value | indent 4 }}
    {{ codeblock }}
{{ end }}

{{ $example := .GetExample }}
{{ if $example }}

!!! info
	To create this resource, a {{ code "Secret" }} containing Aiven token must be [created](/aiven-operator/authentication.html) first.

Apply the resource with:

{{ codeblock }}shell
kubectl apply -f example.yaml
{{ codeblock }}

Verify the newly created {{ code .Kind }}:

{{ codeblock }}shell
kubectl get {{ .Plural }} {{ $example.Metadata.Name }}
{{ codeblock }}

The output is similar to the following:
{{ codeblock }}shell
{{ range $example.Table }}{{ rfill .Title .Value }}    {{ end }}
{{ range $example.Table }}{{ rfill .Value .Title }}    {{ end }}
{{ codeblock }}

{{ if $example.Secret.Name }}
To view the details of the {{ code "Secret" }}, use the following command:
{{ codeblock }}shell
kubectl describe secret {{ $example.Secret.Name }}
{{ codeblock }}

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the {{ code "Secret" }}:

{{ codeblock }}shell
kubectl get secret {{ $example.Secret.Name }} -o json | jq '.data | map_values(@base64d)'
{{ codeblock }}

The output is similar to the following:

{{ codeblock }}{ .json .no-copy }
{
	{{- range $example.Secret.Keys  }}
	"{{ . }}": "<secret>",{{ end }}
}
{{ codeblock }}

{{ end }}

{{ end }}
{{ end }}

{{- template "renderSchema" . -}}

{{ define "renderSchema" -}}
{{ if .Items }}
    {{- template "renderSchema" .Items -}}
{{ else }}
{{ .GetHeader }}

{{ .GetParentLink }}

{{ .GetDescription }}
{{ end }}

{{ $req := .ListRequired }}
{{- if $req }}
**Required**

{{ range $req }}
    {{- template "renderProp" . -}}
{{- end -}}
{{ end -}}

{{- $opt := .ListOptional }}
{{- if $opt }}
**Optional**

{{ range $opt }}
    {{- template "renderProp" . -}}
{{- end -}}
{{ end -}}

{{ range .ListProperties 0 -}}
    {{- if .IsNested -}}
        {{- template "renderSchema" . -}}
    {{- end -}}
{{- end -}}
{{ end -}}

{{ define "renderProp" -}}
- {{ .GetPropertyLink }} ({{ .GetDef }}).{{ if .GetDescription }} {{ .GetDescription }}{{ end }}{{ if .IsNested }} See below for [nested schema](#{{ .GetID }}).{{ end }}
{{ end -}}
`

// reTrailingZeros finds trailing zeros
var reTrailingZeros = regexp.MustCompile(`0+$`)

// prettyDigit formats floats without zero fraction
func prettyDigit(kind string, value float64) string {
	if kind == "integer" {
		return fmt.Sprintf("%d", int(value))
	}

	s := fmt.Sprintf("%.5f", value)
	s = reTrailingZeros.ReplaceAllString(s, "$1")
	return strings.TrimSuffix(s, ".")
}

// loadYAMLs loads a list of yamls
func loadYAMLs[T any](b []byte) []*T {
	decoder := yaml.NewDecoder(bytes.NewReader(b))
	list := make([]*T, 0)
	for {
		doc := new(T)
		err := decoder.Decode(&doc)
		if err != nil {
			break
		}

		list = append(list, doc)
	}
	return list
}

// ReplaceAllStringSubmatchFunc Credits https://gist.github.com/elliotchance/d419395aa776d632d897
func ReplaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func([]string) string) string {
	result := ""
	lastIndex := 0

	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		var groups []string
		for i := 0; i < len(v); i += 2 {
			if v[i] == -1 || v[i+1] == -1 {
				groups = append(groups, "")
			} else {
				groups = append(groups, str[v[i]:v[i+1]])
			}
		}

		result += str[lastIndex:v[0]] + repl(groups)
		lastIndex = v[1]
	}

	return result + str[lastIndex:]
}
