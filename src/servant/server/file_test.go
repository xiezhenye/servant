package server

import (
	"testing"
	"servant/conf"
)

func TestCheck(t *testing.T) {
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
