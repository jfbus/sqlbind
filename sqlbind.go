// sqlbind is a set of database/sql helpers to remove a lot of boilerplate code while always using standard database/sql calls.
//
// Named parameters
//
// Basic usage, using maps :
//   sql, args, err := sqlbind.Named("SELECT * FROM example WHERE name=:name", map[string]interface{}{"name":"foo"})
//   rows, err := db.Query(sql, args...)
//
// Automatic IN clause expansion :
//   sqlbind.Named("SELECT * FROM example WHERE name IN(:name)", map[string]interface{}{"name":[]string{"foo", "bar"}})
//
// Variable args :
//   sqlbind.Named("INSERT INTO example (::names) VALUES(::values)", map[string]interface{}{"name":"foo"}'})
//   sqlbind.Named("UPDATE example SET ::name=::value", map[string]interface{}{"name":"foo"}'})
//
// Structs, using tags to define DB field names :
//   type Example struct {
//   	Name string `db:"name"`
//   }
//   e := Example{Name: "foo"}
//   sqlbind.Named("SELECT * FROM example WHERE name=:name", e)
//
// Add args to a struct (e.g. from query string parameters) :
//   sqlbind.Named("SELECT * FROM example WHERE name=:name AND domain=:domain", e, sqlbind.Args("domain", "example.com"))
//
// Named placeholders are automatically translated to the right driver-dependant placeholder : ? for MySQL (default style)
//   sqlbind.SetStyle(sqlbind.MySQL)
// or $N for PostgreSQL
//   sqlbind.SetStyle(sqlbind.PostgreSQL)
//
// Colons inside quotes are ignored and do not need to be escaped (":foo" will neither be rewritten neither considered a named parameter), but otherwise need to be doubled (::foo will be rewritten to :foo but not be considered a named parameter).
//
// Not all fields need to be expanded by ::names and ::name=::value. This can be achieved using an optional parameter to sqlbin.Named :
//   sqlbind.Named("INSERT INTO example (::names) VALUES(::values)", map[string]interface{}{"id": 42, "name":"foo"}'}, sqlbind.Only("name"))
//   sqlbind.Named("INSERT INTO example (::names) VALUES(::values)", map[string]interface{}{"id": 42, "name":"foo"}'}, sqlbind.Exclude("id"))
// or using struct tags (better performance) :
//   type Example struct {
//   	ID int `db:"id,omit"` // will not be expanded by ::names and ::name=::value
//   }
//
// Variables
//
// Additional variables can be added to SQL queries :
//   sqlbind.Named("SELECT /* {comment} */ * FROM {table_prefix}example WHERE name=:name", e, sqlbind.Variables("comment", "foo", "table_prefix", "bar_"))
//
// Braces inside quotes are ignored : "{value}" will not be modified.
//
// JSON and missing fields
//
// In a REST API, PATCH update calls may update only certain fields. When using structs with plain types, it is impossible to differentiate between empty fields {"name":""}, null fields : {"name": null} and missing fields : {}.
//
// Using pointers, one can differentiate between empty fields and null/missing fields, but not between null and missing fields. In this case, nil values are usually considered missing.
//   type Example struct {
//   	Name *string `db:"name"`
//   }
//
// sqlbind will never expand nil pointer values in ::names and ::name=::value.
//
// https://github.com/jfbus/jsontypes defines types that will be able to manage various cases : null values, missing JSON fields, zero/empty values, read-only values. All types are automatically converted from/to their underlying type when reading/writing to the database.
//   type Example struct {
//   	Name jsontypes.NullString `db:"name"`
//   }
//
// * jsontypes.NullString will either be expanded to "" ("" in JSON), NULL (null in JSON) or not expanded (absent from JSON).
//
// * jsontypes.String will either be expanded to "" ("" or null in JSON) or not expanded (absent from JSON)
//
// * jsontypes.ROString will never be expanded
//
// More generally, all structs that implement `Missing() bool` will be managed by sqlbind.
//
// Result struct binding
//
//   type Example struct {
//   	ID   int    `db:"id,omit"`
//   	Name string `db:"name"`
//   }
//   rows, err := db.Query("SELECT * FROM example")
//   ...
//   defer rows.Close()
//   for rows.Next() {
//       e := Example{}
//       err = sqlbind.Scan(rows, &e)
//   }
//
// QueryRow does not expose column names, Query+ScanRow can be used instead.
//   rows, err := db.Query("SELECT * FROM example")
//   err := sqlbind.ScanRow(rows, &e) // closes
//
// Slices of structs are not mapped, only structs.
//
// Instances
//
// You can build a SQLBinder instance :
//   s := sqlbind.New(sqlbind.MySQL)
//   s.Register(Example{}, Foo{})
//   s.Named("SELECT * FROM example WHERE name=:name", e)
package sqlbind
