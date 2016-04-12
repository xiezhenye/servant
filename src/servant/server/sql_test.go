package server
import "testing"

func TestReplaceSqlParams(t *testing.T) {
	p1 := map[string][]string{ "a":[]string{"1"} }
	p2 := map[string][]string{ "a":[]string{"1"},"b":[]string{"2"} }

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
