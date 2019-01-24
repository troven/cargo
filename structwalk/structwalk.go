package structwalk

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type StructWalkFunc func(path string, in interface{}) (v interface{}, found bool)

func FieldValue(path string, in interface{}) (v interface{}, found bool) {
	defer func() {
		if x := recover(); x != nil {
			return
		}
	}()

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, false
	}
	cur := reflect.ValueOf(in)
	for i, part := range parts {
		for {
			if cur.Kind() == reflect.Ptr || cur.Kind() == reflect.Interface {
				cur = cur.Elem()
				continue
			}
			break
		}
		if cur.Kind() == reflect.Struct {
			cur = cur.FieldByNameFunc(func(name string) bool {
				return strings.ToLower(name) == strings.ToLower(part)
			})
		} else if i != len(parts)-1 {
			// not last, but already has no deep
			return nil, false
		}
	}
	v = cur.Interface()
	found = true
	return
}

func GetterValue(path string, in interface{}) (v interface{}, found bool) {
	defer func() {
		if x := recover(); x != nil {
			return
		}
	}()

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, false
	}
	var parent reflect.Value
	var lastPart string
	cur := reflect.ValueOf(in)
	for i, part := range parts {
		m := cur.MethodByName(part)
		typ := m.Type()
		if typ.NumIn() != 0 || typ.NumOut() != 1 {
			continue
		}
		// consider it a getter
		out := m.Call(nil)
		parent = cur
		cur = out[0]
		if cur.NumMethod() == 0 && i != len(parts)-1 {
			// not last, but already has no deep
			return nil, false
		}
		lastPart = part
	}
	if cur.Kind() == reflect.String {
		m := parent.MethodByName(lastPart + "Bytes")
		if m.IsValid() {
			out := m.Call(nil)
			if vv := out[0]; vv.CanInterface() {
				v = vv.Interface()
				found = true
				return
			}
		}
	}
	v = cur.Interface()
	found = true
	return
}

func FieldList(in interface{}) []string {
	defer func() {
		if x := recover(); x != nil {
			return
		}
	}()

	t := reflect.TypeOf(in)
	for {
		if t.Kind() == reflect.Ptr ||
			t.Kind() == reflect.Interface {
			t = t.Elem()
			continue
		}
		break
	}
	flatList := make([]string, 0, t.NumField())
	flatList = traverseFields("", flatList, t)
	sort.Strings(flatList)
	return flatList
}

func traverseFields(prefix string, flatList []string, t reflect.Type) []string {
	n := t.NumField()
	for i := 0; i < n; i++ {
		f := t.Field(i).Type
		for {
			if f.Kind() == reflect.Ptr || f.Kind() == reflect.Interface {
				f = f.Elem()
				continue
			}
			break
		}
		fPrefix := t.Field(i).Name
		if len(prefix) > 0 {
			fPrefix = fmt.Sprintf("%s.%s", prefix, fPrefix)
		}

		if f.Kind() == reflect.Struct {
			flatList = traverseFields(fPrefix, flatList, f)
			continue
		}
		flatList = append(flatList, fPrefix)
	}
	return flatList
}

func GetterList(in interface{}) []string {
	defer func() {
		if x := recover(); x != nil {
			return
		}
	}()

	t := reflect.TypeOf(in)
	flatList := make([]string, 0, t.NumMethod())
	flatList = traverseGetters("", flatList, t, reflect.ValueOf(in))
	sort.Strings(flatList)
	return flatList
}

func traverseGetters(prefix string, flatList []string,
	t reflect.Type, v reflect.Value) []string {
	n := t.NumMethod()
	for i := 0; i < n; i++ {
		m := t.Method(i).Type
		if m.NumIn() != 1 || m.NumOut() != 1 {
			continue
		}
		mPrefix := t.Method(i).Name
		if len(prefix) > 0 {
			mPrefix = fmt.Sprintf("%s.%s", prefix, mPrefix)
		}

		out := v.Method(i).Call(nil)
		if out[0].Kind() == reflect.Struct {
			flatList = traverseGetters(mPrefix, flatList, out[0].Type(), out[0])
			continue
		}
		flatList = append(flatList, mPrefix)
	}
	return flatList
}
