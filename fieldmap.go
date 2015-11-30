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

// Register registers a type to be used Register is not safe. Do not use concurently.
func Register(l ...interface{}) struct{} {
	for _, i := range l {
		t := reflect.Indirect(reflect.ValueOf(i)).Type()

		is := map[string][]int{}
		buildIndexes(t, []int{}, is)
		fieldMap.index[t] = is

		fieldMap.names[t] = buildNames(t)
	}

	return struct{}{}
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
		fv, ok := field(name, v)
		if !ok || !fv.CanInterface() {
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

func value(key string, arg interface{}, args ...interface{}) (interface{}, bool) {
	nilfound := false
	if m, ok := arg.(map[string]interface{}); ok {
		if val, found := m[key]; found {
			if val == nil {
				nilfound = true
			} else {
				return val, true
			}
		}
	} else if v := reflect.Indirect(reflect.ValueOf(arg)); v.Type().Kind() == reflect.Struct {
		if fv, found := field(key, v); found && fv.CanInterface() {
			if i, ok := fv.Interface().(WillUpdater); ok && i.WillUpdate() == false {
				nilfound = true
			} else if !fv.IsValid() || fv.Interface() == nil {
				nilfound = true
			} else {
				return fv.Interface(), true
			}
		}
	}
	for _, arg := range args {
		if val, found := value(key, arg); found {
			return val, found
		}
	}
	return nil, nilfound
}

func pointerto(key string, arg interface{}) (interface{}, error) {
	if v := reflect.Indirect(reflect.ValueOf(arg)); v.Type().Kind() == reflect.Struct {
		if fv, found := field(key, v); found {
			if !fv.CanAddr() {
				return nil, ErrNoPointerToField
			}
			return fv.Addr().Interface(), nil
		}
	}
	return nil, ErrFieldNotFound
}

func field(key string, v reflect.Value) (reflect.Value, bool) {
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
		ft := f.Type
		tag := f.Tag.Get("db")
		if tag == "-" || (f.PkgPath != "" && !f.Anonymous) {
			continue
		}
		if tag == "" {
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				add := buildNames(ft)
				names = append(names, add...)
				continue
			}
		}
		name, opt := parseTag(tag)
		if opt == "ro" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		names = append(names, name)
	}
	sort.Sort(&names)
	return []string(names)
}

func buildIndexes(t reflect.Type, idx []int, m map[string][]int) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		ft := f.Type
		tag := f.Tag.Get("db")
		if tag == "-" || (f.PkgPath != "" && !f.Anonymous) {
			continue
		}

		nidx := make([]int, len(idx), cap(idx))
		copy(nidx, idx)
		nidx = append(nidx, i)

		if tag == "" {
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				buildIndexes(ft, nidx, m)
				continue
			}
		}
		name, _ := parseTag(tag)
		if name == "" {
			name = f.Name
		}
		m[name] = nidx
	}
}

func parseTag(tag string) (string, string) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tag[idx+1:]
	}
	return tag, ""
}
