package sqlbind

import (
	"bytes"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

const (
	MySQL = Style(iota)
	PostgreSQL
)

var (
	ErrUnsupportedFormat = errors.New("Unsupported data format")
	defaultBinder        = New(MySQL)
)

// The placeholder style to be used, either MySQL (?) or PostgreSQL ($N)
type Style int

type SQLBinder struct {
	style Style

	sync.Mutex
	cache map[string]*decoded
}

// New creates a SQLBinder object, using the specified placeholder style (MySQL or PostgreSQL)
func New(style Style) *SQLBinder {
	return &SQLBinder{
		style: style,
		cache: map[string]*decoded{},
	}
}

// SetStyle sets the style (MySQL or PostgreSQL) of the default binder
func SetStyle(style Style) {
	defaultBinder.style = style
}

type context struct {
	parts []part
	names []string
}

type NamedOption func(*context) error

// Named formats a SQL query, parsing named parameters and variables using the default binder.
// It returns the SQL query and the list of parameters to be used for the database/sql call
//
//   sql, args, err := sqlbin.Named("SELECT * FROM example WHERE foo=:foo", arg)
//   rows, err := db.Query(sql, args...)
//
// args can either be a map[string]interface{} or a struct
func Named(sql string, arg interface{}, opts ...NamedOption) (string, []interface{}, error) {
	return defaultBinder.Named(sql, arg, opts...)
}

// Named formats a SQL query, parsing named parameters and variables using the specified binder
// It returns the SQL query and the list of parameters to be used for the database/sql call
//
//   sql, args, err := sqlbin.Named("SELECT * FROM example WHERE foo=:foo", arg)
//   rows, err := db.Query(sql, args...)
//
// args can either be a map[string]interface{} or a struct
func (s *SQLBinder) Named(sql string, arg interface{}, opts ...NamedOption) (string, []interface{}, error) {
	var c *decoded
	var found bool
	s.Lock()
	if c, found = s.cache[sql]; !found {
		c = decode(sql)
		// TODO : test compilation error
		s.cache[sql] = c
	}
	s.Unlock()
	return s.named(c, arg, opts...)
}

// Variables sets variable values. If a variable has no value, it is replaced with an empty string.
//
//   sqlbin.Named("SELECT /* {comment} */ * FROM {table_prefix}example WHERE foo=:foo", args, sqlbind.Variables("comment", "foobar", "table_prefix", "foo_"))
func Variables(vars ...string) NamedOption {
	if len(vars)%2 != 0 {
		return errorOption(errors.New("Variables() must have a multiple of 2 args"))
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
// 	var e struct {
// 		Foo string `db:"foo"`
// 		Bar string `db:"bar"`
// 		Baz string `db:"baz"`
// 	}
//  sqlbin.Named("UPDATE example SET ::name=::value", arg, sqlbind.Only("bar", "baz"))
//
// would be equivalent to :
//
//  sqlbin.Named("UPDATE example SET bar=:bar, baz=:baz", args)
func Only(names ...string) NamedOption {
	return func(e *context) error {
		e.names = names
		return nil
	}
}

// Exclude removes parameters from ::names, ::values and ::name=::value tags.
//
// 	var e struct {
// 		Foo string `db:"foo"`
// 		Bar string `db:"bar"`
// 		Baz string `db:"baz"`
// 	}
//  sqlbin.Named("UPDATE example SET ::name=::value", arg, sqlbind.Exclude("foo"))
//
// would be equivalent to :
//
//  sqlbin.Named("UPDATE example SET bar=:bar, baz=:baz", args)
func Exclude(names ...string) NamedOption {
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

func errorOption(err error) NamedOption {
	return func(e *context) error {
		return err
	}
}

// replaceNamesValues replaces ::names, ::values and ::name=::value parts with placeholders
func replaceNamesValues(e *context) error {
	var n []part
	for i, p := range e.parts {
		if n == nil && (p.t == typeNames || p.t == typeValues || p.t == typeNameValue) {
			n = make([]part, i, len(e.parts)+len(e.names)*2)
			copy(n, e.parts)
		}
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
			if n != nil {
				n = append(n, p)
			}
		}
	}
	if n != nil {
		e.parts = n
	}
	return nil
}

var bufPool sync.Pool

func newBuf() *bytes.Buffer {
	if v := bufPool.Get(); v != nil {
		buf := v.(*bytes.Buffer)
		buf.Reset()
		return buf
	}
	return &bytes.Buffer{}
}

func (s *SQLBinder) named(c *decoded, arg interface{}, opts ...NamedOption) (string, []interface{}, error) {
	e := &context{
		names: names(arg),
		parts: c.parts,
	}

	for _, opt := range opts {
		if err := opt(e); err != nil {
			return "", nil, err
		}
	}
	replaceNamesValues(e)

	args := make([]interface{}, 0, len(e.names))
	sql := newBuf()
	defer bufPool.Put(sql)
	i := 1
	for _, p := range e.parts {
		switch p.t {
		case typeSQL:
			sql.WriteString(p.data)
		case typePlaceholder:
			val := value(arg, p.data)
			if rval := reflect.ValueOf(val); rval.Kind() == reflect.Slice {
				for si := 0; si < rval.Len(); si++ {
					if si != 0 {
						sql.WriteString(", ")
					}
					s.writePlaceholder(sql, i)
					i++
					args = append(args, rval.Index(si).Interface())
				}
			} else {
				s.writePlaceholder(sql, i)
				i++
				args = append(args, val)
			}
		default:
			return "", nil, errors.New("Unhandled part type")
		}
	}
	return sql.String(), args, nil
}

func (s *SQLBinder) writePlaceholder(buf *bytes.Buffer, i int) {
	switch s.style {
	case PostgreSQL:
		buf.WriteByte('$')
		buf.WriteString(strconv.Itoa(i))
	default:
		buf.WriteByte('?')
	}
}
