package server
import (
	"testing"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"database/sql"
)

func TestReplaceSqlParams(t *testing.T) {
	p1 := func(k string)string {
		if k == "a" {
			return "1"
		}
		return ""
	}
	p2 := func(k string)string {
		if k == "a" {
			return "1"
		}
		if k == "b" {
			return "2"
		}
		return ""
	}

	s, p := replaceSqlParams("select 1", p1)
	if s != "select 1" || len(p) != 0 {
		t.Fail()
	}

	s, p = replaceSqlParams("select ${a}", p1)
	if s != "select ?" || len(p) != 1 || p[0] != "1" {
		t.Fail()
	}

	s, p = replaceSqlParams("select ${a}", p2)
	if s != "select ?" || len(p) != 1 || p[0] != "1" {
		t.Fail()
	}

	s, p = replaceSqlParams("select ${a}, ${b}", p2)
	if s != "select ?, ?" || len(p) != 2 || p[0] != "1" || p[1] != "2" {
		t.Fail()
	}
}

func mockRowsToSqlRows(mockRows sqlmock.Rows) *sql.Rows {
	db, mock, _ := sqlmock.New()
	mock.ExpectQuery("select").WillReturnRows(mockRows)
	rows, _ := db.Query("select")
	return rows
}


func TestRowToResult(t *testing.T) {
	mockRows := sqlmock.NewRows([]string{"a","b","c"})
	mockRows.AddRow(1, 2, 3)
	result, err := rowsToResult(mockRowsToSqlRows(mockRows))
	if err != nil {
		t.Error(err)
	}
	if len(result) != 1 || len(result[0]) != 3 {
		t.Fail()
	}
	if result[0]["a"] != "1" || result[0]["b"] != "2" || result[0]["c"] != "3" {
		t.Fail()
	}

}

