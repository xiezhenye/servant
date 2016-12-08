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
	SetGlobalParam("s_world", "golang")
	SetGlobalParam("hello2", "hello ${s_${w}}")
	SetVarCanExpand("hello2", true)
	v, exists = reqParams("hello2")
    if ! exists {
        t.Error("should exists!")
        return
    }
    if v != "hello golang" {
        t.Errorf("expand wrong! %s", v)
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

func TestVarExpand(t *testing.T) {
	vars := map[string]string {
		"a": "a",
		"b": "b",
		"var": "var",
		"variable": "variable",
	}
	q := func(k string) (string, bool) {
		return vars[k], true
	}
	r := func(s string) string {
		return s
	}
	var ret string
	var ok bool
	ret, ok = VarExpand("111${a}222", q, r)
	if !ok || ret != "111a222" {
		t.Errorf("fail: %s", ret)
	}
	ret, ok = VarExpand("111${a}222${b}333", q, r)
	if !ok || ret != "111a222b333" {
		t.Errorf("fail: %s", ret)
	}
	ret, ok = VarExpand("111${a}222${b}333", q, r)
	if !ok || ret != "111a222b333" {
		t.Errorf("fail: %s", ret)
	}
	ret, ok = VarExpand("${v${a}r}", q, r)
	if !ok || ret != "var" {
		t.Errorf("fail: %s", ret)
	}
	ret, ok = VarExpand("!${${v${a}r}iable}!", q, r)
	if !ok || ret != "!variable!" {
		t.Errorf("fail: %s", ret)
	}
}