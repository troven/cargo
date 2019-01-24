package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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

type Cargo struct {
	Version          string
	ContextCreatedAt time.Time
}

func (c TemplateContext) LengthOf(selector string) int {
	v, ok := structwalk.FieldValue(selector, c)
	if !ok {
		// no such field
		return 0
	}
	collectionV := reflect.ValueOf(v)
	collectionT := reflect.TypeOf(v)
	switch collectionT.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map:
		return collectionV.Len()
	default:
		return 0
	}
}

func (c TemplateContext) ItemAt(selector string, idx int) TemplateContext {
	v, ok := structwalk.FieldValue(selector, c)
	if !ok {
		// no such field
		c["Current"] = nil
		return c
	}
	collectionV := reflect.ValueOf(v)
	collectionT := reflect.TypeOf(v)
	switch collectionT.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map:
		if idx < 0 || idx >= collectionV.Len() {
			// not indexable - out of bounds
			c["Current"] = nil
			return c
		}
		if v := collectionV.Index(idx); v.CanInterface() {
			// set the value index is pointing to
			c["Current"] = v.Interface()
			return c
		}
	default:
		// not indexable
		c["Current"] = nil
		return c
	}
}

func (c TemplateContext) LoadFromJSON(data []byte) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for name, v := range fields {
		if strings.ToLower(name) == "cargo" || stings.ToLower(name) == "current" {
			err := errors.New("root field names 'cargo' and 'current' are reserved")
			return err
		}
		c[name] = v
	}
	return nil
}

func (c TemplateContext) LoadFromYAML(data []byte) error {
	var fields map[string]interface{}
	if err := yaml.Unmarshal(data, &fields); err != nil {
		return err
	}
	for name, v := range fields {
		if strings.ToLower(name) == "cargo" || stings.ToLower(name) == "current" {
			err := errors.New("root field names 'cargo' and 'current' are reserved")
			return err
		}
		c[name] = v
	}
	return nil
}
