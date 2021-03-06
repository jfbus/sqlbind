# sqlbind [![Build Status](https://travis-ci.org/jfbus/sqlbind.svg)](https://travis-ci.org/jfbus/sqlbind) [![Coverage Status](https://coveralls.io/repos/jfbus/sqlbind/badge.svg?branch=master&service=github)](https://coveralls.io/github/jfbus/sqlbind?branch=master) [![](https://godoc.org/github.com/jfbus/sqlbind?status.svg)](http://godoc.org/github.com/jfbus/sqlbind)

sqlbind is a set of `database/sql` helpers to remove a lot of boilerplate code while always using standard `database/sql` calls.

It adds :

* Named parameters,
* Binding named parameters to structs,
* Binding structs to `sql.Row`/`sql.Rows` results,
* Variables in SQL queries.

sqlbind generates as little SQL code as possible, letting you fine tune your SQL requests.

## Named parameters

Basic usage, using maps :
```
sql, args, err := sqlbind.Named("SELECT * FROM example WHERE name=:name", map[string]interface{}{"name":"foo"})
rows, err := db.Query(sql, args...)
```
Automatic `IN` clause expansion :
```
sqlbind.Named("SELECT * FROM example WHERE name IN(:name)", map[string]interface{}{"name":[]string{"foo", "bar"}})
```
Variable args :
```
sqlbind.Named("INSERT INTO example (::names) VALUES(::values)", map[string]interface{}{"name":"foo"}'})
sqlbind.Named("UPDATE example SET ::name=::value", map[string]interface{}{"name":"foo"}'})
```
Structs, using tags to define DB field names :
```
type Example struct {
	Name string `db:"name"`
}
e := Example{Name: "foo"}
sqlbind.Named("SELECT * FROM example WHERE name=:name", e)
```
Add args to a struct (e.g. from query string parameters) :
```
sqlbind.Named("SELECT * FROM example WHERE name=:name AND domain=:domain", e, sqlbind.Args("domain", "example.com"))
```

Named placeholders are automatically translated to the right driver-dependant placeholder : `?` for MySQL (default style)
```
sqlbind.SetStyle(sqlbind.MySQL)
```
or `$N` for PostgreSQL
```
sqlbind.SetStyle(sqlbind.PostgreSQL)
```

Colons inside quotes are ignored and do not need to be escaped (`":value"` will neither be rewritten neither considered a named parameter), but otherwise need to be doubled (`::value` will be rewritten to `:value` but not be considered a named parameter).

## Controlling ::names and ::name=::value

Not all fields need to be expanded by `::names` and `::name=::value`.

This can be achieved using an optional parameter to `sqlbin.Named` :
```
sqlbind.Named("INSERT INTO example (::names) VALUES(::values)", map[string]interface{}{"id": 42, "name":"foo"}'}, sqlbind.Only("name"))
sqlbind.Named("INSERT INTO example (::names) VALUES(::values)", map[string]interface{}{"id": 42, "name":"foo"}'}, sqlbind.Exclude("id"))
```
or using struct tags (better performance) :
```
type Example struct {
	ID   int    `db:"id,ro"` // will not be expanded by ::names and ::name=::value
}
```

## Variables

Additional variables can be added to SQL queries :
```
sqlbind.Named("SELECT /* {comment} */ * FROM {table_prefix}example WHERE name=:name", e, sqlbind.Variables("comment", "foo", "table_prefix", "bar_"))
```

Braces inside quotes are ignored : `"{value}"` will not be modified.

## JSON and missing fields

In a REST API, `PATCH` update calls may update only certain fields. When using structs with plain types, it is impossible to differentiate between empty fields `{"name":""}`, null fields : `{"name": null}` and missing fields : `{}`.

### pointers

```
type Example struct {
	Name *string `db:"name"`
}
```

Using pointers, one can differentiate between empty fields and null/missing fields, but not between null and missing fields. In this case, nil values are usually considered missing.

sqlbind will never expand nil pointer values in `::names` and `::name=::value`.

### jsontypes

[jsontypes](https://github.com/jfbus/jsontypes) defines types that will be able to manage various cases : null values, missing JSON fields, zero/empty values, read-only values. All types are automatically converted from/to their underlying type when reading/writing to the database.

```
type Example struct {
	Name jsontypes.NullString `db:"name"`
}
```

* `jsontypes.NullString` will either be expanded to `""` (`""` in JSON), `NULL` (`null` in JSON) or not expanded (absent from JSON).
* `jsontypes.String` will either be expanded to `""` (`""` or `null` in JSON) or not expanded (absent from JSON)
* `jsontypes.ROString` will never be expanded

See [jsontypes](https://github.com/jfbus/jsontypes) for all types.

More generally, all structs that implement `Missing() bool` will be managed by sqlbind.

## Result struct binding

```
type Example struct {
	ID   int    `db:"id,ro"`
	Name string `db:"name"`
}
rows, err := db.Query("SELECT * FROM example")
...
defer rows.Close()
for rows.Next() {
    e := Example{}
    err = sqlbind.Scan(rows, &e)
}
```
QueryRow does not expose column names, Query+ScanRow can be used instead.
```
rows, err := db.Query("SELECT * FROM example")
err := sqlbind.ScanRow(rows, &e) // closes
```

Slices of structs are not mapped, only structs.

## Performance

sqlbind uses reflection to parse structs. In order to achieve the best performance, it is recommended to register your structs before binding :
```
sqlbind.Register(Example{}, Foo{})
```

Benchmark against [sqlx](https://github.com/jmoiron/sqlx):

```
BenchmarkSQLBindNamedRegister-4  	 1000000	      2069 ns/op	     288 B/op	       5 allocs/op
BenchmarkSqlxNamed-4             	  500000	      2382 ns/op	     624 B/op	      13 allocs/op
```

## Instances

You can build a SQLBinder instance :
```
s := sqlbind.New(sqlbind.MySQL)
s.Register(Example{}, Foo{})
s.Named("SELECT * FROM example WHERE name=:name", e)
```

## License

License: MIT - see LICENSE
