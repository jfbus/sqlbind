package sqlbind

import (
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
)

type testCase struct {
	src   string
	opts  []NamedOption
	mySQL string
	pgSQL string
	args  []interface{}
}

func doTest(t *testing.T, data interface{}, table []testCase, comment string) {
	my := New(MySQL)
	pg := New(Postgresql)
	for _, it := range table {
		if it.opts == nil {
			it.opts = []NamedOption{}
		}
		mySQL, myArgs, myErr := my.Named(it.src, data, it.opts...)
		if myErr != nil {
			t.Errorf("[%s][MySQL] Unable to generate sql for '%s' : %s", comment, it.src, myErr)
		}
		if mySQL != it.mySQL {
			t.Errorf("[%s][MySQL] Expected sql for '%s' was '%s' but got '%s'", comment, it.src, it.mySQL, mySQL)
		}
		if !reflect.DeepEqual(myArgs, it.args) {
			t.Errorf("[%s][MySQL] Expected args for '%s' were '%v' but got '%v'", comment, it.src, it.args, myArgs)
		}
		pgSQL, pgArgs, pgErr := pg.Named(it.src, data, it.opts...)
		if pgErr != nil {
			t.Errorf("[%s][Posgresql] Unable to generate sql for '%s' : %s", comment, it.src, pgErr)
		}
		if pgSQL != it.pgSQL {
			t.Errorf("[%s][Posgresql] Expected sql for '%s' was '%s' but got '%s'", comment, it.src, it.pgSQL, pgSQL)
		}
		if !reflect.DeepEqual(pgArgs, it.args) {
			t.Errorf("[%s][Posgresql] Expected args for '%s' were '%v' but got '%v'", comment, it.src, it.args, pgArgs)
		}
	}
}

func TestNamed(t *testing.T) {
	tc := []testCase{
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
			opts:  []NamedOption{Variables("comment", "foobarbaz")},
			mySQL: `SELECT /* foobarbaz */ * FROM foo`,
			pgSQL: `SELECT /* foobarbaz */ * FROM foo`,
			args:  []interface{}{},
		},
		{
			src:   `SELECT /* {comment} */ * FROM foo WHERE foo=:foo`,
			opts:  []NamedOption{Variables("comment", "foobarbaz")},
			mySQL: `SELECT /* foobarbaz */ * FROM foo WHERE foo=?`,
			pgSQL: `SELECT /* foobarbaz */ * FROM foo WHERE foo=$1`,
			args:  []interface{}{"foobar"},
		},
		{
			src:   `{comment}`,
			opts:  []NamedOption{Variables("comment", "foobarbaz")},
			mySQL: `foobarbaz`,
			pgSQL: `foobarbaz`,
			args:  []interface{}{},
		},
		{
			src:   `SELECT * FROM foo where comment="{comment}"`,
			opts:  []NamedOption{Variables("comment", "foobarbaz")},
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
			opts:  []NamedOption{Only("bar")},
			mySQL: `INSERT INTO example (bar) VALUES(?)`,
			pgSQL: `INSERT INTO example (bar) VALUES($1)`,
			args:  []interface{}{"barbar"},
		},
		{
			src:   `INSERT INTO example (::names) VALUES(::values)`,
			opts:  []NamedOption{Exclude("bar")},
			mySQL: `INSERT INTO example (foo, int, nil) VALUES(?, ?, ?)`,
			pgSQL: `INSERT INTO example (foo, int, nil) VALUES($1, $2, $3)`,
			args:  []interface{}{"foobar", 42, nil},
		},
		{
			src:   `SELECT ::bar FROM foo`,
			mySQL: `SELECT :bar FROM foo`,
			pgSQL: `SELECT :bar FROM foo`,
			args:  []interface{}{},
		},
		{
			src:   `UPDATE example SET ::name=::value WHERE bar=:bar`,
			mySQL: `UPDATE example SET bar=?, foo=?, int=?, nil=? WHERE bar=?`,
			pgSQL: `UPDATE example SET bar=$1, foo=$2, int=$3, nil=$4 WHERE bar=$5`,
			args:  []interface{}{"barbar", "foobar", 42, nil, "barbar"},
		},
	}

	doTest(t, map[string]interface{}{
		"foo": "foobar",
		"bar": "barbar",
		"int": 42,
		"nil": nil,
	}, tc, "map")

	type testStruct struct {
		Id  int         `db:"-"`
		Foo string      `db:"foo"`
		Bar string      `db:"bar"`
		Int int         `db:"int"`
		Nil interface{} `db:"nil"`
	}
	doTest(t, testStruct{
		Foo: "foobar",
		Bar: "barbar",
		Int: 42,
		Nil: nil,
	}, tc, "struct")
	doTest(t, &testStruct{
		Foo: "foobar",
		Bar: "barbar",
		Int: 42,
		Nil: nil,
	}, tc, "structptr")

	Register(testStruct{})
	doTest(t, testStruct{
		Foo: "foobar",
		Bar: "barbar",
		Int: 42,
		Nil: nil,
	}, tc, "registered")

	type e struct {
		Id  int    `db:"-"`
		Foo string `db:"foo"`
	}
	type f struct {
		Int int `db:"int"`
	}
	type testStructEmbed struct {
		E   e
		Bar string `db:"bar"`
		F   f
		Nil interface{} `db:"nil"`
	}
	doTest(t, testStructEmbed{
		E: e{
			Foo: "foobar",
		},
		Bar: "barbar",
		F: f{
			Int: 42,
		},
		Nil: nil,
	}, tc, "embed")
}

func TestNamedIn(t *testing.T) {
	tc := []testCase{
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
	}
	doTest(t, map[string]interface{}{
		"foo": "foobar",
		"bar": []string{"barbar", "barbaz"},
	}, tc, "map/in")
	type testStructIn struct {
		Foo string   `db:"foo"`
		Bar []string `db:"bar"`
	}
	doTest(t, testStructIn{
		Foo: "foobar",
		Bar: []string{"barbar", "barbaz"},
	}, tc, "struct/in")
}

func TestOmit(t *testing.T) {
	tc := []testCase{
		{
			src:   `INSERT INTO example (::names) VALUES(::values)`,
			mySQL: `INSERT INTO example (foo) VALUES(?)`,
			pgSQL: `INSERT INTO example (foo) VALUES($1)`,
			args:  []interface{}{"foobar"},
		},
		{
			src:   `UPDATE example SET ::name=::value WHERE bar=:bar`,
			mySQL: `UPDATE example SET foo=? WHERE bar=?`,
			pgSQL: `UPDATE example SET foo=$1 WHERE bar=$2`,
			args:  []interface{}{"foobar", "barbar"},
		},
	}
	type testStructOmit struct {
		Foo string `db:"foo"`
		Bar string `db:"bar,omit"`
	}
	doTest(t, testStructOmit{
		Foo: "foobar",
		Bar: "barbar",
	}, tc, "struct/omit")
}

func TestNoTag(t *testing.T) {
	tc := []testCase{
		{
			src:   `INSERT INTO example (::names) VALUES(::values)`,
			mySQL: `INSERT INTO example (Bar, Foo) VALUES(?, ?)`,
			pgSQL: `INSERT INTO example (Bar, Foo) VALUES($1, $2)`,
			args:  []interface{}{"barbar", "foobar"},
		},
		{
			src:   `UPDATE example SET ::name=::value`,
			mySQL: `UPDATE example SET Bar=?, Foo=?`,
			pgSQL: `UPDATE example SET Bar=$1, Foo=$2`,
			args:  []interface{}{"barbar", "foobar"},
		},
		{
			src:   `SELECT * FROM foo WHERE foo=:Foo AND bar=:Bar`,
			mySQL: `SELECT * FROM foo WHERE foo=? AND bar=?`,
			pgSQL: `SELECT * FROM foo WHERE foo=$1 AND bar=$2`,
			args:  []interface{}{"foobar", "barbar"},
		},
	}
	type testStructOmit struct {
		Foo string
		Bar string
	}
	doTest(t, testStructOmit{
		Foo: "foobar",
		Bar: "barbar",
	}, tc, "struct/notag")
}

type Missing bool

func (m Missing) WillUpdate() bool {
	return !bool(m)
}

func TestMissing(t *testing.T) {
	type testStructMissing struct {
		Foo Missing
		Bar *string
		Baz string
	}
	barbar := "barbar"
	tt := testStructMissing{
		Foo: false,
		Bar: &barbar,
		Baz: "bazbar",
	}
	doTest(t, tt, []testCase{
		{
			src:   `INSERT INTO example (::names) VALUES(::values)`,
			mySQL: `INSERT INTO example (Bar, Baz, Foo) VALUES(?, ?, ?)`,
			pgSQL: `INSERT INTO example (Bar, Baz, Foo) VALUES($1, $2, $3)`,
			args:  []interface{}{tt.Bar, "bazbar", Missing(false)},
		},
		{
			src:   `UPDATE example SET ::name=::value`,
			mySQL: `UPDATE example SET Bar=?, Baz=?, Foo=?`,
			pgSQL: `UPDATE example SET Bar=$1, Baz=$2, Foo=$3`,
			args:  []interface{}{tt.Bar, "bazbar", Missing(false)},
		},
		{
			src:   `SELECT * FROM foo WHERE foo=:Foo AND bar=:Bar`,
			mySQL: `SELECT * FROM foo WHERE foo=? AND bar=?`,
			pgSQL: `SELECT * FROM foo WHERE foo=$1 AND bar=$2`,
			args:  []interface{}{Missing(false), tt.Bar},
		},
	}, "struct/missing/none")
	tt.Foo = true
	doTest(t, tt, []testCase{
		{
			src:   `INSERT INTO example (::names) VALUES(::values)`,
			mySQL: `INSERT INTO example (Bar, Baz) VALUES(?, ?)`,
			pgSQL: `INSERT INTO example (Bar, Baz) VALUES($1, $2)`,
			args:  []interface{}{tt.Bar, "bazbar"},
		},
		{
			src:   `UPDATE example SET ::name=::value`,
			mySQL: `UPDATE example SET Bar=?, Baz=?`,
			pgSQL: `UPDATE example SET Bar=$1, Baz=$2`,
			args:  []interface{}{tt.Bar, "bazbar"},
		},
		{
			src:   `SELECT * FROM foo WHERE foo=:Foo AND bar=:Bar`,
			mySQL: `SELECT * FROM foo WHERE foo=? AND bar=?`,
			pgSQL: `SELECT * FROM foo WHERE foo=$1 AND bar=$2`,
			args:  []interface{}{Missing(true), tt.Bar},
		},
	}, "struct/missing/struct")
	tt.Foo = false
	tt.Bar = nil
	doTest(t, tt, []testCase{
		{
			src:   `INSERT INTO example (::names) VALUES(::values)`,
			mySQL: `INSERT INTO example (Baz, Foo) VALUES(?, ?)`,
			pgSQL: `INSERT INTO example (Baz, Foo) VALUES($1, $2)`,
			args:  []interface{}{"bazbar", Missing(false)},
		},
		{
			src:   `UPDATE example SET ::name=::value`,
			mySQL: `UPDATE example SET Baz=?, Foo=?`,
			pgSQL: `UPDATE example SET Baz=$1, Foo=$2`,
			args:  []interface{}{"bazbar", Missing(false)},
		},
		{
			src:   `SELECT * FROM foo WHERE foo=:Foo AND bar=:Bar`,
			mySQL: `SELECT * FROM foo WHERE foo=? AND bar=?`,
			pgSQL: `SELECT * FROM foo WHERE foo=$1 AND bar=$2`,
			args:  []interface{}{Missing(false), tt.Bar},
		},
	}, "struct/missing/ptr")
}

func TestErrors(t *testing.T) {
	_, _, err := Named("{var}", map[string]interface{}{}, Variables("var"))
	if err == nil {
		t.Error("Variables() called with 1 parameter should return an error, but got none")
	}
	_, _, err = Named("foo", nil)
	if err != nil {
		t.Error("Calling Named with a nil arg should not generate an error, but got %s", err)
	}
}

func BenchmarkSQLBindNamedNoRegister(b *testing.B) {
	type testStruct struct {
		Foo string `db:"foo"`
		Bar string `db:"bar"`
		Baz int    `db:"baz"`
	}

	for i := 0; i < b.N; i++ {
		Named("SELECT * FROM foo WHERE foo=:foo AND bar=:bar AND baz=:baz", testStruct{Foo: "foo", Bar: "bar", Baz: 42})
	}
}

func BenchmarkSQLBindNamed(b *testing.B) {
	type testStruct struct {
		Foo string `db:"foo"`
		Bar string `db:"bar"`
		Baz int    `db:"baz"`
	}
	Register(testStruct{})
	for i := 0; i < b.N; i++ {
		Named("SELECT * FROM foo WHERE foo=:foo AND bar=:bar AND baz=:baz", testStruct{Foo: "foo", Bar: "bar", Baz: 42})
	}
}

func BenchmarkSqlxNamed(b *testing.B) {
	type testStruct struct {
		Foo string `db:"foo"`
		Bar string `db:"bar"`
		Baz int    `db:"baz"`
	}

	for i := 0; i < b.N; i++ {
		sqlx.Named("SELECT * FROM foo WHERE foo=:foo AND bar=:bar AND baz=:baz", testStruct{Foo: "foo", Bar: "bar", Baz: 42})
	}
}
