package server

import (
	"testing"
	"bytes"
	"strings"
)

func TestLog(t *testing.T) {
	b := make([]byte, 0)
	bb := bytes.NewBuffer(b)
	logger.SetOutput(bb)
	sess := Session{}
	sess.id = 1000
	sess.log("a", "INFO", "hello")
	if ! strings.HasSuffix(string(bb.Bytes()), "INFO (1000) [a] hello\n") {
		t.Fail()
	}
	bb.Reset()
	sess.resource = "res"
	sess.warn("hello %s", "world")
	if ! strings.HasSuffix(string(bb.Bytes()), "WARN (1000) [res] hello world\n") {
		t.Fail()
	}
}
