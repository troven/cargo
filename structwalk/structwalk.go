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
				return name == part
			})
		} else if cur.Kind() == reflect.Map {
			keys := cur.MapKeys()
			var keyFound bool
			for _, k := range keys {
				if k.String() == part {
					cur = cur.MapIndex(k)
					keyFound = true
					break
				}
			}
			if !keyFound {
				return nil, false
			}
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
	v := reflect.ValueOf(in)
	for {
		if t.Kind() == reflect.Ptr ||
			t.Kind() == reflect.Interface {
			t = t.Elem()
			v = v.Elem()
			continue
		}
		break
	}
	var flatList []string
	if t.Kind() == reflect.Struct {
		flatList = make([]string, 0, t.NumField())
		flatList = traverseFields("", flatList, t, v)
	} else if t.Kind() == reflect.Map {
		flatList = make([]string, 0, len(v.MapKeys()))
		flatList = traverseMap("", flatList, t, v)
	}
	sort.Strings(flatList)
	return flatList
}

func traverseFields(prefix string, flatList []string, t reflect.Type, v reflect.Value) []string {
	n := t.NumField()
	for i := 0; i < n; i++ {
		var field reflect.Value
		if v.IsValid() {
			field = v.Field(i)
		}
		fieldType := t.Field(i).Type
		for {
			if fieldType.Kind() == reflect.Ptr || fieldType.Kind() == reflect.Interface {
				fieldType = fieldType.Elem()
				if field.IsValid() {
					field = field.Elem()
				}
				continue
			}
			break
		}
		fieldPrefix := t.Field(i).Name
		if len(prefix) > 0 {
			fieldPrefix = fmt.Sprintf("%s.%s", prefix, fieldPrefix)
		}

		if fieldType.Kind() == reflect.Struct {
			flatList = traverseFields(fieldPrefix, flatList, fieldType, field)
			continue
		} else if fieldType.Kind() == reflect.Map {
			flatList = traverseMap(fieldPrefix, flatList, fieldType, field)
			continue
		}
		flatList = append(flatList, fieldPrefix)
	}
	return flatList
}

func traverseMap(prefix string, flatList []string, t reflect.Type, v reflect.Value) []string {
	for _, key := range v.MapKeys() {
		var field reflect.Value
		if v.IsValid() {
			field = v.MapIndex(key)
		}
		fieldType := field.Type()
		for {
			if fieldType.Kind() == reflect.Ptr || fieldType.Kind() == reflect.Interface {
				fieldType = fieldType.Elem()
				if field.IsValid() {
					field = field.Elem()
				}
				continue
			}
			break
		}
		fieldPrefix := key.String()
		if len(prefix) > 0 {
			fieldPrefix = fmt.Sprintf("%s.%s", prefix, key.String())
		}
		if fieldType.Kind() == reflect.Struct {
			flatList = traverseFields(fieldPrefix, flatList, fieldType, field)
			continue
		} else if fieldType.Kind() == reflect.Map {
			flatList = traverseMap(fieldPrefix, flatList, fieldType, field)
			continue
		}
		flatList = append(flatList, fieldPrefix)
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
