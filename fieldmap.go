package sqlbind

import (
	"reflect"
	"sort"
	"strings"
	"sync"
)

var fieldMap = struct {
	sync.Mutex
	index map[reflect.Type]map[string][]int
	names map[reflect.Type][]string
}{
	index: map[reflect.Type]map[string][]int{},
	names: map[reflect.Type][]string{},
}

func names(arg interface{}) []string {
	if m, ok := arg.(map[string]interface{}); ok {
		names := make(sort.StringSlice, 0, len(m))
		for name := range m {
			names = append(names, name)
		}
		sort.Sort(&names)
		return []string(names)
	} else if v := reflect.Indirect(reflect.ValueOf(arg)); v.Type().Kind() == reflect.Struct {
		if names, found := fieldMap.names[v.Type()]; found {
			return names
		}
		return buildNames(v.Type())
	}
	return []string{}
}

// TODO : remove missing fields from names
// func missing()

func value(arg interface{}, key string) interface{} {
	if m, ok := arg.(map[string]interface{}); ok {
		return m[key]
	} else if v := reflect.Indirect(reflect.ValueOf(arg)); v.Type().Kind() == reflect.Struct {
		is, found := fieldMap.index[v.Type()]
		if !found {
			is = map[string][]int{}
			buildIndexes(v.Type(), []int{}, is)
		}
		if i, found := is[key]; found {
			return v.FieldByIndex(i).Interface()
		}
	}
	return nil
}

func buildNames(t reflect.Type) []string {
	names := make(sort.StringSlice, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("sqlbind")
		if tag == "-" {
			continue
		}
		if f.Type.Kind() == reflect.Struct {
			add := buildNames(f.Type)
			names = append(names, add...)
		} else {
			name, opt := parseTag(tag)
			if opt == "omit" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			names = append(names, name)
		}
	}
	sort.Sort(&names)
	return []string(names)
}

func buildIndexes(t reflect.Type, idx []int, m map[string][]int) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("sqlbind")
		if tag == "-" {
			continue
		}

		nidx := make([]int, len(idx), cap(idx))
		copy(nidx, idx)
		nidx = append(nidx, i)

		if f.Type.Kind() == reflect.Struct {
			buildIndexes(f.Type, nidx, m)
		} else {
			name, _ := parseTag(tag)
			if name == "" {
				name = f.Name
			}
			m[name] = nidx
		}
	}
}

func parseTag(tag string) (string, string) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tag[idx+1:]
	}
	return tag, ""
}
