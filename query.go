package sqlbind

import (
	"errors"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	MySQL = Style(iota)
	Postgresql
)

var (
	ErrUnsupportedFormat = errors.New("Unsupported data format")
	defaultBinder        = New(MySQL)
)

// The placeholder style to be used, either MySQL (?) or Postgresql ($N)
type Style int

type SQLBinder struct {
	style Style

	sync.Mutex
	cache map[string]*compiled
}

// New creates a SQLBinder object, using the specified placeholder style
func New(style Style) *SQLBinder {
	return &SQLBinder{
		style: style,
		cache: map[string]*compiled{},
	}
}

// SetStyle sets the style of the default binder
func SetStyle(style Style) {
	defaultBinder.style = style
}

type context struct {
	parts []part
	names []string
}

type namedOption func(*context) error

// Named formats a SQL query, parsing named parameters and variables using the default binder.
// It returns the SQL query and the list of parameters to be used for the database/sql call
//
//   sql, sqlargs, err := sqlbin.Named("SELECT * FROM example WHERE foo=:foo", args)
//   rows, err := db.Query(sql, sqlargs...)
//
// args can either be a map[string]interface{} or a struct
func Named(sql string, args interface{}, opts ...namedOption) (string, []interface{}, error) {
	return defaultBinder.Named(sql, args, opts...)
}

// Named formats a SQL query, parsing named parameters and variables using the specified binder
// It returns the SQL query and the list of parameters to be used for the database/sql call
//
//   sql, sqlargs, err := sqlbin.Named("SELECT * FROM example WHERE foo=:foo", args)
//   rows, err := db.Query(sql, sqlargs...)
//
// args can either be a map[string]interface{} or a struct
func (s *SQLBinder) Named(sql string, args interface{}, opts ...namedOption) (string, []interface{}, error) {
	var c *compiled
	var found bool
	s.Lock()
	if c, found = s.cache[sql]; !found {
		c = compile(sql)
		// TODO : test compilation error
		s.cache[sql] = c
	}
	s.Unlock()
	if m, ok := args.(map[string]interface{}); ok {
		return s.namedMap(c, m, opts...)
	}
	return "", nil, ErrUnsupportedFormat
}

// Variables sets variable values. If a variable has no value, it is replaced with an empty string.
//
//   sqlbin.Named("SELECT /* {comment} */ * FROM {table_prefix}example WHERE foo=:foo", args, sqlbind.Variables("comment", "foobar", "table_prefix", "foo_"))
func Variables(vars ...string) namedOption {
	if len(vars) != 2 {
		return errorOption(errors.New("Variables must have a multiple of 2 args"))
	}
	v := map[string]string{}
	for i := 0; i < len(vars); i += 2 {
		v[vars[i]] = vars[i+1]
	}

	return func(e *context) error {
		n := make([]part, 0, len(e.parts))
		for _, p := range e.parts {
			if p.t == typeVariable {
				if val, ok := v[p.data]; ok {
					n = append(n, part{t: typeSQL, data: val})
				}
			} else {
				n = append(n, p)
			}
		}
		e.parts = n
		return nil
	}
}

// Only sets the list of parameters to be used in ::names, ::values and ::name=::value tags.
//
//  sqlbin.Named("UPDATE example SET ::name=::value WHERE foo=:foo", args, sqlbind.Only("bar", "baz"))
//
// would be equivalent to :
//
//  sqlbin.Named("UPDATE example SET bar=:bar, baz=:baz WHERE foo=:foo", args)
func Only(names ...string) namedOption {
	return func(e *context) error {
		e.names = names
		return nil
	}
}

// Exclude removes parameters from ::names, ::values and ::name=::value tags.
//
//  sqlbin.Named("UPDATE example SET ::name=::value WHERE foo=:foo", args, sqlbind.Exclude("foo"))
//
// would be equivalent to :
//
//  sqlbin.Named("UPDATE example SET bar=:bar, baz=:baz WHERE foo=:foo", args)
func Exclude(names ...string) namedOption {
	ex := map[string]struct{}{}
	for _, name := range names {
		ex[name] = struct{}{}
	}
	return func(e *context) error {
		n := make([]string, 0, len(e.names))
		for _, name := range e.names {
			if _, found := ex[name]; found {
				continue
			}
			n = append(n, name)
		}
		e.names = n
		return nil
	}
}

func errorOption(err error) namedOption {
	return func(e *context) error {
		return err
	}
}

// replaceNamesValues replaces ::names, ::values and ::name=::value parts with placeholders
func replaceNamesValues(e *context) error {
	n := make([]part, 0, len(e.parts))
	for _, p := range e.parts {
		switch p.t {
		case typeNames:
			n = append(n, part{t: typeSQL, data: strings.Join(e.names, ", ")})
		case typeValues:
			for i, name := range e.names {
				if i > 0 {
					n = append(n, part{t: typeSQL, data: ", "})
				}
				n = append(n, part{t: typePlaceholder, data: name})
			}
		case typeNameValue:
			for i, name := range e.names {
				if i > 0 {
					n = append(n, part{t: typeSQL, data: ", " + name + "="})
				} else {
					n = append(n, part{t: typeSQL, data: name + "="})
				}
				n = append(n, part{t: typePlaceholder, data: name})
			}
		default:
			n = append(n, p)
		}
	}
	e.parts = n
	return nil
}

func (s *SQLBinder) namedMap(c *compiled, m map[string]interface{}, opts ...namedOption) (string, []interface{}, error) {
	e := &context{
		names: make([]string, 0, len(m)),
		parts: c.parts,
	}

	for name := range m {
		e.names = append(e.names, name)
	}
	tmp := sort.StringSlice(e.names)
	sort.Sort(&tmp)

	for _, opt := range opts {
		if err := opt(e); err != nil {
			return "", nil, err
		}
	}
	replaceNamesValues(e)

	args := []interface{}{}
	sql := ""
	i := 1
	for _, p := range e.parts {
		switch p.t {
		case typeSQL:
			sql += p.data
		case typePlaceholder:
			if val := reflect.ValueOf(m[p.data]); val.Kind() == reflect.Slice {
				for si := 0; si < val.Len(); si++ {
					if si != 0 {
						sql += ", "
					}
					sql += s.placeholder(i)
					i++
					args = append(args, val.Index(si).Interface())
				}
			} else {
				sql += s.placeholder(i)
				i++
				args = append(args, m[p.data])
			}
		default:
			return "", nil, errors.New("Unhandled part type")
		}
	}
	return sql, args, nil
}

func (s *SQLBinder) placeholder(i int) string {
	switch s.style {
	case Postgresql:
		return "$" + strconv.Itoa(i)
	default:
		return "?"
	}
}
