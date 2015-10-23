package sqlbind

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
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
		Id  int    `db:"-"`
		Foo string `db:"foo"`
		Baz int    `db:"baz"`
		Bar string `db:"bar"`
	}
	ts := testStruct{}
	rows.Next()
	err := Scan(rows, &ts)
	if err != nil {
		t.Errorf("ScanRow returned an error : %s", err)
	} else {
		ref := testStruct{Foo: "foobar", Bar: "barbar", Baz: 42}
		if ts != ref {
			t.Errorf("ScanRow returned %v, %v expected", ts, ref)
		}
	}
}

func TestScanEmbedded(t *testing.T) {
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
	type sub struct {
		Foo string `db:"foo"`
	}
	type testStruct struct {
		Id  int `db:"-"`
		Sub sub
		Baz int    `db:"baz"`
		Bar string `db:"bar"`
	}
	ts := testStruct{}
	rows.Next()
	err := Scan(rows, &ts)
	if err != nil {
		t.Errorf("ScanRow returned an error : %s", err)
	} else {
		ref := testStruct{Sub: sub{Foo: "foobar"}, Bar: "barbar", Baz: 42}
		if ts != ref {
			t.Errorf("ScanRow returned %v, %v expected", ts, ref)
		}
	}
}

func TestScanAnonymous(t *testing.T) {
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
	type sub struct {
		Foo string `db:"foo"`
	}
	type testStruct struct {
		Id int `db:"-"`
		sub
		Baz int    `db:"baz"`
		Bar string `db:"bar"`
	}
	ts := testStruct{}
	rows.Next()
	err := Scan(rows, &ts)
	if err != nil {
		t.Errorf("ScanRow returned an error : %s", err)
	} else {
		ref := testStruct{sub: sub{Foo: "foobar"}, Bar: "barbar", Baz: 42}
		if ts != ref {
			t.Errorf("ScanRow returned %v, %v expected", ts, ref)
		}
	}
}

func TestScanEmbeddedPointer(t *testing.T) {
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
	type sub struct {
		Foo string `db:"foo"`
	}
	type testStruct struct {
		Id  int `db:"-"`
		Sub *sub
		Baz int    `db:"baz"`
		Bar string `db:"bar"`
	}
	ts := testStruct{Sub: &sub{}}
	rows.Next()
	err := Scan(rows, &ts)
	if err != nil {
		t.Errorf("ScanRow returned an error : %s", err)
	} else {
		ref := testStruct{Sub: &sub{Foo: "foobar"}, Bar: "barbar", Baz: 42}
		if !reflect.DeepEqual(ts, ref) {
			t.Errorf("ScanRow returned %v/%v, %v/%v expected", ts, ts.Sub, ref, ref.Sub)
		}
	}
}

func TestScanMissing(t *testing.T) {
	defer testdb.Reset()

	testdb.SetQueryFunc(func(query string) (result driver.Rows, err error) {
		columns := []string{"foo", "bar", "baz", "foobar"}
		rows := [][]driver.Value{
			[]driver.Value{"foobar", "barbar", 42, "foobarbar"},
		}
		return testdb.RowsFromSlice(columns, rows), nil
	})

	db, _ := sql.Open("testdb", "")
	rows, _ := db.Query("SELECT foo FROM bar")

	type testStruct struct {
		Id  int    `db:"-"`
		Foo string `db:"foo"`
		Baz int    `db:"baz"`
		Bar string `db:"bar"`
	}
	ts := testStruct{}
	rows.Next()
	err := Scan(rows, &ts)
	if err != nil {
		t.Errorf("ScanRow returned an error : %s", err)
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
		Id  int    `db:"-"`
		Foo string `db:"foo"`
		Baz int    `db:"baz"`
		Bar string `db:"bar"`
	}
	ts := testStruct{}
	err := ScanRow(rows, &ts)
	if err != nil {
		t.Errorf("ScanRow returned an error : %s", err)
	} else {
		ref := testStruct{Foo: "foobar", Bar: "barbar", Baz: 42}
		if ts != ref {
			t.Errorf("ScanRow returned %v, %v expected", ts, ref)
		}
	}
}
