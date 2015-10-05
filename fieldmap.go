package sqlbind

import (
	"errors"
	"reflect"
	"sort"
	"strings"
)

var (
	fieldMap = struct {
		index map[reflect.Type]map[string][]int
		names map[reflect.Type][]string
	}{
		index: map[reflect.Type]map[string][]int{},
		names: map[reflect.Type][]string{},
	}

	ErrNoPointerToField = errors.New("Cannot get pointer to field")
	ErrFieldNotFound    = errors.New("Field not found")
)

// Register is not safe. Do not use concurently.
func Register(l ...interface{}) {
	for _, i := range l {
		t := reflect.Indirect(reflect.ValueOf(i)).Type()

		is := map[string][]int{}
		buildIndexes(t, []int{}, is)
		fieldMap.index[t] = is

		fieldMap.names[t] = buildNames(t)
	}
}

func names(arg interface{}) []string {
	if arg == nil {
		return []string{}
	}
	if m, ok := arg.(map[string]interface{}); ok {
		names := make(sort.StringSlice, 0, len(m))
		for name := range m {
			names = append(names, name)
		}
		sort.Sort(&names)
		return []string(names)
	} else if v := reflect.Indirect(reflect.ValueOf(arg)); v.Type().Kind() == reflect.Struct {
		if names, found := fieldMap.names[v.Type()]; found {
			return filterMissing(names, v)
		}
		return filterMissing(buildNames(v.Type()), v)
	}
	return []string{}
}

type WillUpdater interface {
	WillUpdate() bool
}

func filterMissing(names []string, v reflect.Value) []string {
	n := make([]string, 0, len(names))
	for _, name := range names {
		fv, ok := field(v, name)
		if !ok {
			continue
		}
		if i, ok := fv.Interface().(WillUpdater); ok && i.WillUpdate() == false {
			continue
		}
		if fv.Kind() == reflect.Ptr && fv.IsNil() {
			continue
		}
		n = append(n, name)
	}
	return n
}

// TODO : remove missing fields from names
// func missing()

func value(arg interface{}, key string) interface{} {
	if m, ok := arg.(map[string]interface{}); ok {
		return m[key]
	} else if v := reflect.Indirect(reflect.ValueOf(arg)); v.Type().Kind() == reflect.Struct {
		if fv, found := field(v, key); found {
			return fv.Interface()
		}
	}
	return nil
}

func pointerto(arg interface{}, key string) (interface{}, error) {
	if v := reflect.Indirect(reflect.ValueOf(arg)); v.Type().Kind() == reflect.Struct {
		if fv, found := field(v, key); found {
			if !fv.CanAddr() {
				return nil, ErrNoPointerToField
			}
			return fv.Addr().Interface(), nil
		}
	}
	return nil, ErrFieldNotFound
}

func field(v reflect.Value, key string) (reflect.Value, bool) {
	is, found := fieldMap.index[v.Type()]
	if !found {
		is = map[string][]int{}
		buildIndexes(v.Type(), []int{}, is)
	}
	if i, found := is[key]; found {
		return v.FieldByIndex(i), true
	}
	return reflect.Value{}, false
}

func buildNames(t reflect.Type) []string {
	names := make(sort.StringSlice, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("db")
		if tag == "-" {
			continue
		}
		if tag == "" && f.Type.Kind() == reflect.Struct {
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
		tag := f.Tag.Get("db")
		if tag == "-" {
			continue
		}

		nidx := make([]int, len(idx), cap(idx))
		copy(nidx, idx)
		nidx = append(nidx, i)

		if tag == "" && f.Type.Kind() == reflect.Struct {
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
