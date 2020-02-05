package server

import (
	"reflect"
	"testing"
)

func TestGetCmdExecArgs(t *testing.T) {
	name, args, exists := getCmdExecArgs(`test a b ${a} ${b} '' "" 'a b' "c d${c}"`, func(string) (string, bool) {
		return "X", true
	})
	if !exists {
		t.Error("exists")
	}
	if name != "test" {
		t.Error("name != test")
	}
	if !reflect.DeepEqual(args, []string{"a", "b", "X", "X", "", "", "a b", "c dX"}) {
		t.Error("args wrong")
	}
}
