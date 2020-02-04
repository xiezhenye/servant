package server

import "testing"

func TestCheckPermission(t *testing.T) {
	if !checkPermission("a", []string{"a", "b", "c"}) {
		t.Fail()
	}
	if !checkPermission("b", []string{"a", "b", "c"}) {
		t.Fail()
	}
	if !checkPermission("c", []string{"a", "b", "c"}) {
		t.Fail()
	}
	if checkPermission("x", []string{"a", "b", "c"}) {
		t.Fail()
	}
}

func TestCheckHosts(t *testing.T) {
	if !checkHosts("10.11.12.13", []string{"10.0.0.0/8"}) {
		t.Fail()
	}

	if !checkHosts("10.11.12.13", []string{"10.11.12.13/32"}) {
		t.Fail()
	}
	/*
		if ! checkHosts("10.11.12.13", []string{"10.11.12.13"}) {
			t.Fail()
		}*/

	if checkHosts("10.11.12.13", []string{"10.11.12.254/32"}) {
		t.Fail()
	}

	if checkHosts("10.11.12.13", []string{"xxxx", "10.11.12.254/32"}) {
		t.Fail()
	}

	if !checkHosts("10.11.12.13", []string{"192.168.0.1/24", "10.11.12.13/32"}) {
		t.Fail()
	}

	if !checkHosts("10.11.12.13", []string{}) {
		t.Fail()
	}
}

func TestParseAuthHeader(t *testing.T) {
	if _, _, _, e := parseAuthHeader(""); e == nil {
		t.Fail()
	}
	if _, _, _, e := parseAuthHeader("a x"); e == nil {
		t.Fail()
	}

	if _, _, _, e := parseAuthHeader("a x x"); e == nil {
		t.Fail()
	}

	if _, _, _, e := parseAuthHeader("a 123 xxx"); e != nil {
		t.Fail()
	}
}
