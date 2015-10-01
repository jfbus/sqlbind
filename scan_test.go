package sqlbind

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/erikstmartin/go-testdb"
)

func TestScan(t *testing.T) {
	defer testdb.Reset()

	testdb.SetQueryFunc(func(query string) (result driver.Rows, err error) {
		columns := []string{"foo", "bar", "baz"}
		rows := [][]driver.Value{
			[]driver.Value{"foobar", "barbar", 42},
		}
		return testdb.RowsFromSlice(columns, rows), nil
	})

	db, _ := sql.Open("testdb", "")
	rows, _ := db.Query("SELECT foo FROM bar")

	type testStruct struct {
		Id  int    `sqlbind:"-"`
		Foo string `sqlbind:"foo"`
		Baz int    `sqlbind:"baz"`
		Bar string `sqlbind:"bar"`
	}
	ts := testStruct{}
	rows.Next()
	err := Scan(rows, &ts)
	if err != nil {
		t.Errorf("ScanRow returned an error", err)
	} else {
		ref := testStruct{Foo: "foobar", Bar: "barbar", Baz: 42}
		if ts != ref {
			t.Errorf("ScanRow returned %v, %v expected", ts, ref)
		}
	}
}

func TestScanRow(t *testing.T) {
	defer testdb.Reset()

	testdb.SetQueryFunc(func(query string) (result driver.Rows, err error) {
		columns := []string{"foo", "bar", "baz"}
		rows := [][]driver.Value{
			[]driver.Value{"foobar", "barbar", 42},
		}
		return testdb.RowsFromSlice(columns, rows), nil
	})

	db, _ := sql.Open("testdb", "")
	rows, _ := db.Query("SELECT foo FROM bar")

	type testStruct struct {
		Id  int    `sqlbind:"-"`
		Foo string `sqlbind:"foo"`
		Baz int    `sqlbind:"baz"`
		Bar string `sqlbind:"bar"`
	}
	ts := testStruct{}
	err := ScanRow(rows, &ts)
	if err != nil {
		t.Errorf("ScanRow returned an error", err)
	} else {
		ref := testStruct{Foo: "foobar", Bar: "barbar", Baz: 42}
		if ts != ref {
			t.Errorf("ScanRow returned %v, %v expected", ts, ref)
		}
	}
}
