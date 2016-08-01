package server
import (
	"testing"
)
func TestExpand(t *testing.T) {
	reqParams := requestParams(nil)
	SetGlobalParam("hello", "hello ${w}")
	SetGlobalParam("w", "world")
	SetVarCanExpand("hello", true)
	v, exists := reqParams("hello")
	if ! exists {
		t.Error("should exists!")
		return
	}
	if v != "hello world" {
		t.Error("expand wrong!")
	}
	SetGlobalParam("w", "servant")
	v, exists = reqParams("hello")
	if ! exists {
		t.Error("should exists!")
		return
	}
	if v != "hello servant" {
		t.Error("expand wrong!")
	}
	SetVarCanExpand("a", true)
	SetVarCanExpand("b", true)
	SetGlobalParam("a", "${b}")
	SetGlobalParam("b", "${a}")
	_, exists = reqParams("a")
	if exists {
		t.Error("exists shoud be false")
	}

	SetGlobalParam("hello", "hello ${xxx}")
	_, exists = reqParams("hello")
	if exists {
		t.Error("exists shoud be false")
	}

}
