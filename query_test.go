package sqlbind

import (
	"reflect"
	"testing"
)

type testCase struct {
	src   string
	opts  []namedOption
	mySQL string
	pgSQL string
	args  []interface{}
}

func doTest(t *testing.T, data map[string]interface{}, table []testCase) {
	my := New(MySQL)
	pg := New(Postgresql)
	for _, it := range table {
		if it.opts == nil {
			it.opts = []namedOption{}
		}
		mySQL, myArgs, myErr := my.Named(it.src, data, it.opts...)
		if myErr != nil {
			t.Errorf("[MySQL] Unable to generate sql for '%s' : %s", it.src, myErr)
		}
		if mySQL != it.mySQL {
			t.Errorf("[MySQL] Expected sql for '%s' was '%s' but got '%s'", it.src, it.mySQL, mySQL)
		}
		if !reflect.DeepEqual(myArgs, it.args) {
			t.Errorf("[MySQL] Expected args for '%s' was '%v' but got '%v'", it.src, it.args, myArgs)
		}
		pgSQL, pgArgs, pgErr := pg.Named(it.src, data, it.opts...)
		if pgErr != nil {
			t.Errorf("[Posgresql] Unable to generate sql for '%s' : %s", it.src, pgErr)
		}
		if pgSQL != it.pgSQL {
			t.Errorf("[Posgresql] Expected sql for '%s' was '%s' but got '%s'", it.src, it.pgSQL, pgSQL)
		}
		if !reflect.DeepEqual(pgArgs, it.args) {
			t.Errorf("[Posgresql] Expected args for '%s' was '%v' but got '%v'", it.src, it.args, pgArgs)
		}
	}
}

func TestNamed(t *testing.T) {
	doTest(t, map[string]interface{}{
		"foo": "foobar",
		"bar": "barbar",
		"int": 42,
		"nil": nil,
	}, []testCase{
		{
			src:   ``,
			mySQL: ``,
			pgSQL: ``,
			args:  []interface{}{},
		},
		{
			src:   `SELECT * FROM foo`,
			mySQL: `SELECT * FROM foo`,
			pgSQL: `SELECT * FROM foo`,
			args:  []interface{}{},
		},
		{
			src:   `SELECT * FROM foo WHERE foo=:foo AND bar=:bar`,
			mySQL: `SELECT * FROM foo WHERE foo=? AND bar=?`,
			pgSQL: `SELECT * FROM foo WHERE foo=$1 AND bar=$2`,
			args:  []interface{}{"foobar", "barbar"},
		},
		{
			src:   `SELECT * FROM foo WHERE foo=:int AND bar=:nil`,
			mySQL: `SELECT * FROM foo WHERE foo=? AND bar=?`,
			pgSQL: `SELECT * FROM foo WHERE foo=$1 AND bar=$2`,
			args:  []interface{}{42, nil},
		},
		{
			src:   `SELECT * FROM foo WHERE foo=:int AND bar=:missing`,
			mySQL: `SELECT * FROM foo WHERE foo=? AND bar=?`,
			pgSQL: `SELECT * FROM foo WHERE foo=$1 AND bar=$2`,
			args:  []interface{}{42, nil},
		},
		{
			src:   `SELECT * FROM foo WHERE foo=:foo AND bar=":bar"`,
			mySQL: `SELECT * FROM foo WHERE foo=? AND bar=":bar"`,
			pgSQL: `SELECT * FROM foo WHERE foo=$1 AND bar=":bar"`,
			args:  []interface{}{"foobar"},
		},
		{
			src:   `SELECT /* {comment} */ * FROM foo`,
			opts:  []namedOption{Variables("comment", "foobarbaz")},
			mySQL: `SELECT /* foobarbaz */ * FROM foo`,
			pgSQL: `SELECT /* foobarbaz */ * FROM foo`,
			args:  []interface{}{},
		},
		{
			src:   `SELECT /* {comment} */ * FROM foo WHERE foo=:foo`,
			opts:  []namedOption{Variables("comment", "foobarbaz")},
			mySQL: `SELECT /* foobarbaz */ * FROM foo WHERE foo=?`,
			pgSQL: `SELECT /* foobarbaz */ * FROM foo WHERE foo=$1`,
			args:  []interface{}{"foobar"},
		},
		{
			src:   `/* {comment} */`,
			opts:  []namedOption{Variables("comment", "foobarbaz")},
			mySQL: `/* foobarbaz */`,
			pgSQL: `/* foobarbaz */`,
			args:  []interface{}{},
		},
		{
			src:   `SELECT * FROM foo where comment="{comment}"`,
			opts:  []namedOption{Variables("comment", "foobarbaz")},
			mySQL: `SELECT * FROM foo where comment="{comment}"`,
			pgSQL: `SELECT * FROM foo where comment="{comment}"`,
			args:  []interface{}{},
		},
		{
			src:   `INSERT INTO example (::names) VALUES(::values)`,
			mySQL: `INSERT INTO example (bar, foo, int, nil) VALUES(?, ?, ?, ?)`,
			pgSQL: `INSERT INTO example (bar, foo, int, nil) VALUES($1, $2, $3, $4)`,
			args:  []interface{}{"barbar", "foobar", 42, nil},
		},
		{
			src:   `INSERT INTO example (::names) VALUES(::values)`,
			opts:  []namedOption{Only("bar")},
			mySQL: `INSERT INTO example (bar) VALUES(?)`,
			pgSQL: `INSERT INTO example (bar) VALUES($1)`,
			args:  []interface{}{"barbar"},
		},
		{
			src:   `INSERT INTO example (::names) VALUES(::values)`,
			opts:  []namedOption{Exclude("bar")},
			mySQL: `INSERT INTO example (foo, int, nil) VALUES(?, ?, ?)`,
			pgSQL: `INSERT INTO example (foo, int, nil) VALUES($1, $2, $3)`,
			args:  []interface{}{"foobar", 42, nil},
		},
		{
			src:   `UPDATE example SET ::name=::value WHERE bar=:bar`,
			mySQL: `UPDATE example SET bar=?, foo=?, int=?, nil=? WHERE bar=?`,
			pgSQL: `UPDATE example SET bar=$1, foo=$2, int=$3, nil=$4 WHERE bar=$5`,
			args:  []interface{}{"barbar", "foobar", 42, nil, "barbar"},
		},
	})
}

func TestNamedIn(t *testing.T) {
	doTest(t, map[string]interface{}{
		"foo": "foobar",
		"bar": []string{"barbar", "barbaz"},
	}, []testCase{
		{
			src:   ``,
			mySQL: ``,
			pgSQL: ``,
			args:  []interface{}{},
		},
		{
			src:   `SELECT * FROM foo WHERE foo=:foo AND bar IN(:bar)`,
			mySQL: `SELECT * FROM foo WHERE foo=? AND bar IN(?, ?)`,
			pgSQL: `SELECT * FROM foo WHERE foo=$1 AND bar IN($2, $3)`,
			args:  []interface{}{"foobar", "barbar", "barbaz"},
		},
	})
}
