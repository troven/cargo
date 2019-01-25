package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/troven/cargo/structwalk"
	"github.com/troven/cargo/version"
)

// TemplateContext defines the context root that is going to supply the data to templates
// it is always a map, hovewer its items can be of any type. TemplateContext always has "Cargo" field.
type TemplateContext map[string]interface{}

func NewTemplateContext() TemplateContext {
	return TemplateContext{
		"Cargo": &Cargo{
			Version:          fmt.Sprintf("%s (commmit %s)", version.Version, version.GitCommit),
			ContextCreatedAt: time.Now(),
		},
	}
}

// Cargo keeps context fields related to the renderer.
type Cargo struct {
	Version          string
	ContextCreatedAt time.Time
}

// LengthOf returns length of a collection specified by selector. If there is no
// field matching selector, or its value is not indexable, it will return false.
func (c TemplateContext) LengthOf(selector string) (int, bool) {
	v, ok := structwalk.FieldValue(selector, c)
	if !ok {
		// no such field
		return 0, false
	}
	collectionV := reflect.ValueOf(v)
	collectionT := reflect.TypeOf(v)
	switch collectionT.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map:
		return collectionV.Len(), true
	default:
		return 0, false
	}
}

// CurrentAt returns a shallow copy of TemplateContext, with "Current" root field
// set to the current item in the collection, at index idx. If there is no item,
// or the collection is not indexable, it sets "Current" to nil.
func (c TemplateContext) CurrentAt(selector string, idx int) TemplateContext {
	view := make(TemplateContext, len(c))
	for k, v := range c {
		view[k] = v
		view["Current"] = nil
	}
	v, ok := structwalk.FieldValue(selector, c)
	if !ok {
		// no such field
		return view
	}
	collectionV := reflect.ValueOf(v)
	collectionT := reflect.TypeOf(v)
	switch collectionT.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map:
		if idx < 0 || idx >= collectionV.Len() {
			// not indexable - out of bounds
			return view
		}
		if v := collectionV.Index(idx); v.CanInterface() {
			// set the value index is pointing to
			view["Current"] = v.Interface()
			return view
		}
	}
	// not indexable
	return view
}

// CurrentItem returns the value of a matching field from Current context.
func (c TemplateContext) CurrentItem(selector string) (interface{}, bool) {
	return structwalk.FieldValue(selector, c["Current"])
}

// Item returns the value of a matching field from Template context.
func (c TemplateContext) Item(selector string) (interface{}, bool) {
	return structwalk.FieldValue(selector, c)
}

// LoadFromJSON parses a JSON source and builds context from that, setting it to
// the root field of context specified by name.
func (c TemplateContext) LoadFromJSON(name string, data []byte) error {
	var fields interface{}
	if existingFields, ok := c[name]; ok {
		fields = existingFields
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	c[name] = fields
	return nil
}

// LoadFromYAML parses a YAML source and builds context from that, setting it to
// the root field of context specified by name.
func (c TemplateContext) LoadFromYAML(name string, data []byte) error {
	var fields interface{}
	if existingFields, ok := c[name]; ok {
		fields = existingFields
	}
	if err := yaml.Unmarshal(data, &fields); err != nil {
		return err
	}
	return nil
}

// LoadEnvVars fills context "ENV" environment variables map.
func (c TemplateContext) LoadEnvVars() error {
	pairs := os.Environ()
	envVars := make(map[string]string)
	for _, pair := range pairs {
		nameVal := strings.Split(pair, "=")
		if len(nameVal) == 2 {
			envVars[nameVal[0]] = nameVal[1]
		}
	}
	c["ENV"] = envVars
	return nil
}

// LoadOsVars fills context "OS" with some variables from OS.
func (c TemplateContext) LoadOsVars() error {
	osVars := make(map[string]string)
	osVars["PathSeparator"] = string(os.PathSeparator)
	osVars["PathListSeparator"] = string(os.PathListSeparator)
	osVars["WorkDir"], _ = os.Getwd()
	osVars["Hostname"], _ = os.Hostname()
	osVars["Executable"], _ = os.Executable()
	osVars["ARCH"] = runtime.GOARCH
	osVars["OS"] = runtime.GOOS
	c["OS"] = osVars
	return nil
}
