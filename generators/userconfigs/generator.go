package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/google/go-cmp/cmp"
	"github.com/stoewer/go-strcase"
	"golang.org/x/exp/slices"
	"golang.org/x/tools/imports"
	"gopkg.in/yaml.v3"
)

// generate writes to file a service user config for a given serviceList
func generate(dstDir string, serviceTypes []byte, serviceList []string) error {
	// root level object
	var root map[string]*object

	err := yaml.Unmarshal(serviceTypes, &root)
	if err != nil {
		return err
	}

	done := make([]string, 0, len(serviceList))
	for _, k := range serviceList {
		v, ok := root[k]
		if !ok {
			continue
		}

		dirPath := filepath.Join(dstDir, k)
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			return err
		}

		b, err := newUserConfigFile(k, v)
		if err != nil {
			log.Println(err)
			continue
		}

		filePath := filepath.Join(dirPath, k+".go")
		err = os.WriteFile(filePath, b, 0644)
		if err != nil {
			return err
		}

		if slices.Contains(done, k) {
			return fmt.Errorf("%q is a duplicate name on the list", k)
		}
		done = append(done, k)
	}

	if d := cmp.Diff(serviceList, done); d != "" {
		return fmt.Errorf("not all services are generated: %s", d)
	}
	return nil
}

// newUserConfigFile generates jennifer file from the root object
func newUserConfigFile(name string, obj *object) ([]byte, error) {
	// User config has UserConfig suffix
	configName := name + "_user_config"
	root := toCamelCase(configName)
	obj.init(root) // Cascade init from the top

	// Package naming convention doesn't allow snake case
	packageName := strings.ReplaceAll(strings.ToLower(configName), "_", "")

	// packageName won't be used as a file name, as we return its data, not dumping to disk
	pack := jen.NewFile(packageName)
	pack.HeaderComment("Code generated by user config generator. DO NOT EDIT.")

	// Makes kubebuilder generate DeepCopy method for the package
	pack.HeaderComment("// +kubebuilder:object:generate=true")
	err := addObject(pack, obj)
	if err != nil {
		return nil, err
	}

	// Jenifer won't use imports from code chunks added with Op()
	// Even calling explicit import won't work,
	// cause if module is not used, then it's dropped
	// Calls goimports which fixes missing imports
	b, err := imports.Process("", []byte(pack.GoString()), nil)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// objectType json object types
type objectType string

const (
	objectTypeObject  objectType = "object"
	objectTypeArray   objectType = "array"
	objectTypeString  objectType = "string"
	objectTypeBoolean objectType = "boolean"
	objectTypeInteger objectType = "integer"
	objectTypeNumber  objectType = "number"
)

// ObjectInternal internal fields for object
type objectInternal struct {
	jsonName   string // original name from json spec
	structName string // go struct name in CamelCase
	index      int    // field order in object.Properties
}

// object represents OpenApi object
type object struct {
	objectInternal

	Enum []*struct {
		Value string `yaml:"value"`
	} `yaml:"enum"`
	Pattern   string `yaml:"pattern"`
	MinItems  *int   `yaml:"min_items"`
	MaxItems  *int   `yaml:"max_items"`
	MinLength *int   `yaml:"min_length"`
	MaxLength *int   `yaml:"max_length"`
	// Store both int and float
	Minimum *float64 `yaml:"minimum"`
	Maximum *float64 `yaml:"maximum"`

	// OpenAPI Spec
	Type           objectType         `yaml:"-"`
	OrigType       interface{}        `yaml:"type"`
	Format         string             `yaml:"format"`
	Title          string             `yaml:"title"`
	Description    string             `yaml:"description"`
	Properties     map[string]*object `yaml:"properties"`
	ArrayItems     *object            `yaml:"items"`
	RequiredFields []string           `yaml:"required"`
	CreateOnly     bool               `yaml:"create_only"`
	Required       bool               `yaml:"-"`
	// Go doesn't support nullable scalar types, e.g.:
	// type Foo struct {
	//     Foo *bool `json:"foo,omitempty"
	// }
	// To be able to send "false" we use pointer. So "nil" becomes "empty"
	// Then if we need to send "nil", we remove "omitempty"
	//     Foo *bool `json:"foo"
	// Now it is possible to send [null, true, false]
	// But the field becomes required, and it's mandatory to have it manifest
	// We can mark field as "optional" for builder:
	//     // +optional
	//     Foo *bool `json:"foo,omitempty"
	// That means that KubeAPI won't require this field on request.
	// But that would send explicit "nil" to Aiven API.
	// Now you need "default" value to send it instead of "nil", if default is not nil.
	// Adding `+nullable` will fail on API call for the same reason (pointer vs omitempty vs default value)
	// Another reason is that spec is mostly invalid, and nullable fields are not so.
	// So for simplicity this generator doesn't support nullable values.
	Nullable bool `yaml:"-"` // Not really used for now

}

// init initiates object after it gets values from OpenAPI spec
func (o *object) init(name string) {
	o.jsonName = name
	o.structName = toCamelCase(name)

	// Sorts properties so they keep order on each generation
	keys := make([]string, 0, len(o.Properties))
	for k := range o.Properties {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	required := make(map[string]bool, len(o.RequiredFields))
	for _, k := range o.RequiredFields {
		required[k] = true
	}

	for i, k := range keys {
		child := o.Properties[k]
		child.index = i
		child.Required = required[k]
		child.init(k)
	}

	if o.ArrayItems != nil {
		o.ArrayItems.init(name)
		// Slice items always Required, but for GO struct pointers are better
		o.ArrayItems.Required = o.ArrayItems.Type != objectTypeObject
		// Slice items can't be null, if so it is invalid spec
		o.ArrayItems.Nullable = false
	}

	// Types can be list of strings, or a string
	if v, ok := o.OrigType.(string); ok {
		o.Type = objectType(v)
	} else if v, ok := o.OrigType.([]interface{}); ok {
		o.Type = objectType(v[0].(string))
		for _, t := range v {
			switch s := t.(string); s {
			case "null":
				// Enums can't be nullable
				o.Nullable = len(o.Enum) == 0
			case "string":
				o.Type = objectType(s)
			default:
				// Sets if not empty, string is priority
				if o.Type != "" {
					o.Type = objectType(s)
				}
			}
		}
	}
}

// addObject adds object to jen.File
func addObject(file *jen.File, obj *object) error {
	// We need to iterate over fields by index,
	// so new structs and properties are ordered
	// Or we will get diff everytime we generate files
	keyOrder := make([]string, len(obj.Properties))
	for key, child := range obj.Properties {
		keyOrder[child.index] = key
	}

	fields := make([]jen.Code, len(obj.Properties))
	for _, key := range keyOrder {
		child := obj.Properties[key]
		f, err := addField(file, jen.Id(child.structName), child)
		if err != nil {
			return fmt.Errorf("%s: %s", key, err)
		}
		fields[child.index] = f
	}

	// Creates struct and adds fmtComment if available
	s := jen.Type().Id(obj.structName).Struct(fields...)
	if c := fmtComment(obj); c != "" {
		s = jen.Comment(fmtComment(obj)).Line().Add(s)
	}

	file.Add(s)
	return nil
}

func addField(file *jen.File, s *jen.Statement, obj *object) (*jen.Statement, error) {
	s, err := addFieldType(file, s, obj)
	if err != nil {
		return nil, err
	}

	s = addFieldComments(s, obj)
	s = addFieldTags(s, obj)
	return s.Line(), nil
}

func addFieldType(file *jen.File, s *jen.Statement, obj *object) (*jen.Statement, error) {
	if !obj.Required {
		// Adds to all types, except arrays, which are of pointer type in go
		if obj.Type != objectTypeArray {
			s = s.Op("*")
		}
	}

	switch obj.Type {
	case objectTypeObject:
		err := addObject(file, obj)
		if err != nil {
			return nil, err
		}
		s = s.Id(obj.structName)
	case objectTypeArray:
		return addFieldType(file, s.Index(), obj.ArrayItems)
	case objectTypeString:
		s = s.String()
	case objectTypeBoolean:
		s = s.Bool()
	case objectTypeInteger:
		s = s.Int()
	case objectTypeNumber:
		s = s.Float64()
	default:
		return nil, fmt.Errorf("unknown type %q", obj.Type)
	}
	return s, nil
}

// addFieldTags adds tags for marshal/unmarshal
// with `groups` tag it is possible to mark "create only" fields, like `admin_password`
func addFieldTags(s *jen.Statement, obj *object) *jen.Statement {
	tags := map[string]string{
		"json":   obj.jsonName,
		"groups": "create",
	}

	if !obj.Required {
		tags["json"] += ",omitempty"
	}

	// CreatOnly can't be updated
	if !obj.CreateOnly {
		tags["groups"] += ",update"
	}
	return s.Tag(tags)
}

// addFieldComments add validation markers and doc string
func addFieldComments(s *jen.Statement, obj *object) *jen.Statement {
	c := make([]string, 0)

	// Sets min-max if they are not equal (both empty or equal to zero)
	min := objMinimum(obj)
	max := objMaximum(obj)
	if min != max {
		if min != "" {
			c = append(c, "// +kubebuilder:validation:Minimum="+min)
		}
		if max != "" {
			c = append(c, "// +kubebuilder:validation:Maximum="+max)
		}
	}

	if obj.MinLength != nil {
		c = append(c, fmt.Sprintf("// +kubebuilder:validation:MinLength=%d", *obj.MinLength))
	}
	if obj.MaxLength != nil {
		c = append(c, fmt.Sprintf("// +kubebuilder:validation:MaxLength=%d", *obj.MaxLength))
	}
	if obj.MinItems != nil {
		c = append(c, fmt.Sprintf("// +kubebuilder:validation:MinItems=%d", *obj.MinItems))
	}
	if obj.MaxItems != nil {
		c = append(c, fmt.Sprintf("// +kubebuilder:validation:MaxItems=%d", *obj.MaxItems))
	}
	if obj.Pattern != "" {
		_, err := regexp.Compile(obj.Pattern)
		if err != nil {
			log.Printf("can't compile field %q regex `%s`: %s", obj.jsonName, obj.Pattern, err)
		} else {
			c = append(c, fmt.Sprintf("// +kubebuilder:validation:Pattern=`%s`", obj.Pattern))
		}
	}
	if len(obj.Enum) != 0 {
		enum := make([]string, len(obj.Enum))
		for i, s := range obj.Enum {
			if obj.Type == objectTypeString {
				enum[i] = fmt.Sprintf("%q", s.Value)
			} else {
				enum[i] = s.Value
			}
		}
		c = append(c, fmt.Sprintf("// +kubebuilder:validation:Enum=%s", strings.Join(enum, ";")))
	}
	if obj.CreateOnly {
		c = append(c, `// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"`)
	}

	doc := fmtComment(obj)
	if doc != "" {
		c = append(c, doc)
	}

	if len(c) != 0 {
		s = jen.Comment(strings.Join(c, "\n")).Line().Add(s)
	}

	return s
}

// fmtComment creates nice comment from object.Title or object.Description (takes the longest string)
func fmtComment(obj *object) string {
	d := ""
	if len(obj.Description) > len(obj.Title) {
		d = obj.Description
	} else {
		d = obj.Title
	}

	if d == "" {
		return d
	}
	// Do not add field/struct name into the comment.
	// Otherwise, generated manifests and docs will have those as a part of the description
	return strings.ReplaceAll("// "+d, "\n", " ")
}

// toCamelCase some fields has dots within, makes cleaner camelCase
func toCamelCase(s string) string {
	return strcase.UpperCamelCase(strings.ReplaceAll(s, ".", "_"))
}

// maxSafeInt float64 mantissa size is 53 bits (52 explicitly stored) - 1
// - Can be exactly represented as an IEEE-754 double precision number, and
// - IEEE-754 representation cannot be the result of rounding any other integer to fit the IEEE-754 representation.
const maxSafeInt = float64(1<<53 - 1)

func objMaximum(obj *object) string {
	if obj.Maximum == nil {
		return ""
	}

	f := *obj.Maximum
	if f > maxSafeInt {
		return ""
	}

	if obj.Type == objectTypeInteger {
		return fmt.Sprint(int(f))
	}
	return fmt.Sprint(f)
}

func objMinimum(obj *object) string {
	if obj.Minimum == nil {
		return ""
	}

	f := *obj.Minimum
	if f < -maxSafeInt {
		return ""
	}

	if obj.Type == objectTypeInteger {
		return fmt.Sprint(int(f))
	}
	return fmt.Sprint(f)
}
