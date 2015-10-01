package sqlbind

import "database/sql"

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
		ptr, err := pointerto(arg, name)
		if err != nil {
			return err
		}
		vals[i] = ptr
	}
	return rows.Scan(vals...)
}

func ScanRow(rows *sql.Rows, arg interface{}) error {
	defer rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	if !rows.Next() {
		return sql.ErrNoRows
	}
	names, err := rows.Columns()
	if err != nil {
		return err
	}
	vals := make([]interface{}, len(names))
	for i, name := range names {
		ptr, err := pointerto(arg, name)
		if err != nil {
			return err
		}
		vals[i] = ptr
	}
	return rows.Scan(vals...)
}
