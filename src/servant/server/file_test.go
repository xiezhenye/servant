package server

import (
	"testing"
	"servant/conf"
	"reflect"
)

func TestCheckDirAllow(t *testing.T) {
	dirConf := conf.Dir {
		Allows: []string {
			"POST",
			"GET",
		},
	}
	if err := checkDirAllow(&dirConf, "", "POST"); err != nil {
		t.Errorf("POST should be allowed")
	}
	if checkDirAllow(&dirConf, "aaa", "GET") != nil {
		t.Errorf("GET should be allowed")
	}

	if checkDirAllow(&dirConf, "aaa", "PUT") == nil {
		t.Errorf("PUT should be denied")
	}
	if checkDirAllow(&dirConf, "aaa", "DELETE") == nil {
		t.Errorf("DELETE should be denied")
	}
	if checkDirAllow(&dirConf, "", "HEAD") == nil {
		t.Errorf("HEAD should be denied")
	}
	dirConf = conf.Dir {
		Allows: []string {
			"GET",
		},
		Patterns: []string {
			`aaa`,
			`/abc\.xxx$`,
		},
	}
	if checkDirAllow(&dirConf, "aaa", "GET") != nil {
		t.Errorf("aaa should be allowed")
	}
	if checkDirAllow(&dirConf, "xaaax", "GET") != nil {
		t.Errorf("xaaax should be allowed")
	}
	if checkDirAllow(&dirConf, "/abc.xxx", "GET") != nil {
		t.Errorf("/abc.xxx should be allowed")
	}
	if checkDirAllow(&dirConf, "aaa/abc.xxx", "GET") != nil {
		t.Errorf("aaa/abc.xxx should be allowed")
	}

	if checkDirAllow(&dirConf, "/abc.xxx1", "GET") == nil {
		t.Errorf("/abc.xxx1 should be denied")
	}
	if checkDirAllow(&dirConf, "xxx", "GET") == nil {
		t.Errorf("xxx should be denied")
	}

	if checkDirAllow(&dirConf, "aaa", "POST") == nil {
		t.Errorf("POST should be denied")
	}
}

func TestParseRange(t *testing.T) {
	rgs, err := parseRange("", 1)
	if err != nil || rgs != nil {
		t.Fail()
	}
	rgs, err = parseRange("111", 1)
	if err == nil || rgs != nil {
		t.Fail()
	}
	rgs, err = parseRange("bytes=0-10", 20)
	if len(rgs)!=1 || rgs[0].start != 0 || rgs[0].length != 11 {
		t.Fail()
	}

	rgs, err = parseRange("bytes=-10", 20)
	if len(rgs)!=1 || rgs[0].start != 10 || rgs[0].length != 10 {
		t.Fail()
	}

	rgs, err = parseRange("bytes=1-", 20)
	if len(rgs)!=1 || rgs[0].start != 1 || rgs[0].length != 19 {
		t.Fail()
	}
}

func TestContentRange(t *testing.T) {
	r := httpRange{ 2, 3 }.contentRange(5)
	if r != "bytes 2-4/5" {
		t.Fail()
	}
}

func TestFuncByMethod(t *testing.T) {
	s := FileServer{}
	if reflect.ValueOf(s.funcByMethod("XXX")).Pointer() != reflect.ValueOf(s.serveUnknown).Pointer() {
		t.Fail()
	}
	if reflect.ValueOf(s.funcByMethod("GET")).Pointer() != reflect.ValueOf(s.serveGet).Pointer() {
		t.Fail()
	}
}
