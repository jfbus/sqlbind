package sqlbind

import "database/sql"

// Scan maps the columns of the current row of a sql.Rows result to a struct
//
//  type Example struct {
//		ID   int    `db:"id,omit"`
//		Name string `db:"name"`
//	}
//	rows, err := db.Query("SELECT * FROM example")
//	...
//	defer rows.Close()
//	for rows.Next() {
//	    e := Example{}
//	    err = sqlbind.Scan(rows, &e)
//	}
func Scan(rows *sql.Rows, arg interface{}) error {
	if rows.Err() != nil {
		return rows.Err()
	}
	names, err := rows.Columns()
	if err != nil {
		return err
	}
	vals := make([]interface{}, len(names))
	for i, name := range names {
		ptr, err := pointerto(name, arg)
		if err != nil && err != ErrFieldNotFound {
			return err
		}
		if ptr == nil {
			vals[i] = &sql.RawBytes{}
		} else {
			vals[i] = ptr
		}
	}
	return rows.Scan(vals...)
}

// ScanRow maps the columns of the first row of a sql.Rows result either to a struct, and closes the rows.
//
// sql.QueryRow does not expose column names, therefore ScanRow uses sql.Rows instead of sql.Row.
//
// 	rows, err := db.Query("SELECT * FROM example")
// 	err := sqlbind.ScanRow(rows, &e)
func ScanRow(rows *sql.Rows, arg interface{}) error {
	defer rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	if !rows.Next() {
		return sql.ErrNoRows
	}
	return Scan(rows, arg)
}
