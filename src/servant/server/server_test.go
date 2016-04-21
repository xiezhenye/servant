package server

import (
	"testing"
)

func TestParseUriPath(t *testing.T) {
	r, g, i, l := parseUriPath("/aaa/bbb/ccc/ddd")
	if r != "aaa" || g != "bbb" || i != "ccc" || l != "/ddd" {
		t.Fail()
	}
	r, g, i, l = parseUriPath("/a_a_a/b-b-b/_ccc_")
	if r != "a_a_a" || g != "b-b-b" || i != "_ccc_" || l != "" {
		t.Fail()
	}
	r, g, i, l = parseUriPath("/a_a_a/b-b-b/-/ddd")
	if r != "" || g != "" || i != "" || l != "" {
		t.Fail()
	}
	r, g, i, l = parseUriPath("/aaa/bbb")
	if r != "" || g != "" || i != "" || l != "" {
		t.Fail()
	}
 }