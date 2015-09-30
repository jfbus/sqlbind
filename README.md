# sqlbind

sqlbind is a set of `database/sql` helpers to remove most of the boilerplate code while using standard `database/sql` calls.

## Named parameters

Building queries using named variables :
```
sql, args, err := sqlbind.Named("SELECT * FROM example WHERE name=:name", map[string]interface{}{"name":"foo"})
```
Automatic in clause expansion :
```
sql, args, err := sqlbind.Named("SELECT * FROM example WHERE name IN(:name)", map[string]interface{}{"name":[]string{"foo", "bar"}})
```
Variable args :
```
sql, args, err := sqlbind.Named("INSERT INTO example (::names) VALUES(::values)", map[string]interface{}{"name":"foo"}'})
sql, args, err := sqlbind.Named("UPDATE example SET ::name=::value WHERE name=:name", map[string]interface{}{"name":"foo"}'})
```
Structs, using tags to define DB field names :
```
type example struct {
	Name string `sqlbind:"name"`
}
e := example{Name: "foo"}
sql, args, err := sqlbind.Named("SELECT * FROM example WHERE name=:name", e)
```

Named placeholders are automatically translated to the right driver-dependant placeholder : `?` for MySQL or `$N` for Postgresql.
```
sqlbind.SetPlaceholderType(sqlbind.MySQL)
```
or
```
sqlbind.SetPlaceholderType(sqlbind.Postgresql)
```

## Controling ::names and ::name=::value

Not all fields need to be expanded by `::names` and `::name=::value`.

This can be achieved using an optional parameter to `sqlbin.Named` :
```
sql, args, err := sqlbind.Named("INSERT INTO example (::names) VALUES(::values)", map[string]interface{}{"id": 42, "name":"foo"}'}, []string{"name"})
```
or using struct tags :
```
type example struct {
	ID   int    `sqlbind:"id,nonames,noname"` // will not be expanded by ::names and ::name=::value
}
```

## JSON and missing fields

In a REST API, update (`PATCH`) calls may update only certain fields. When using structs with plain types, it is impossible to differentiate between empty fields `{"name":""}`, null fields : `{"name": null}` and missing fields : `{}`.

### pointers

```
type example struct {
	Name *string `sqlbind:"name"`
}
```

Using pointers can differentiate between empty fields and null/missing fields, but not between null and missing fields. In this case, nil values are usually considered "missing values".

sqlbind will never expand nil pointer values in by `::names` and `::name=::value`.

### jsontypes

jsontypes defines types that will be able to manage various cases : null values, missing JSON fields, zero/empty values, read-only values.

```
type example struct {
	Name jsontypes.NullString `sqlbind:"name"`
}
```

* `jsontypes.NullString` will either be expanded to `""` (`""` in JSON), `NULL` (`null` in JSON) or not expanded (absent from JSON).
* `jsontypes.String` will either be expanded to `""` (`""` or `null` in JSON) or not expanded (absent from JSON)
* `jsontypes.ROString` will never be expanded

See jsontypes for all types.

## Result struct binding

```
type example struct {
	ID   int    `sqlbind:"id,nonames,noname"` // will not be expanded by ::names and ::name=::value
	Name string `sqlbind:"name"`
}
rows, err := db.Query("SELECT * FROM example")
...
defer rows.Close()
for rows.Next() {
    e := example{}
    err = sqlbind.ScanRows(rows, &e)
}
```
```
row := db.QueryRow("SELECT * FROM example")
err := sqlbind.ScanRow(row, &e)
```

Slices of structs are not mapped, only structs.

## Performance

sqlbind uses reflection to parse structs. In order to achieve the best performance, it is recommended to register your structs before binding :
```
sqlbind.Register(Example{}, Foo{})
```

## Instances

You can build a SQLBinder instance :
```
s := sqlbind.New()
s.SetPlaceholderType(sqlbind.MySQL)
s.Register(Example{}, Foo{})
s.Named("SELECT * FROM example WHERE name=:name", e)
```