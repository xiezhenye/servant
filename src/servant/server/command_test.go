package server

import (
	"testing"
	"reflect"
)

func TestGetCmdExecArgs(t *testing.T) {
	name, args := getCmdExecArgs(`test a b ${a} ${b} '' "" 'a b' "c d${c}"`, func(string)string{
		return "X"
	})
	if name != "test" {
		t.Error("name != test")
	}
	if ! reflect.DeepEqual(args, []string{"a", "b", "X", "X", "", "", "a b", "c dX"}) {
		t.Error("args wrong")
	}
}
