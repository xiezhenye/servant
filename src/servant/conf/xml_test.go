package conf

import (
	"testing"
	"fmt"
	"sort"
	"math"
)

func TestConfig(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8" ?>
<config>
	<server><listen>:2465</listen></server>
    <commands id="db1">
        <command id="foo">
            <code>echo hello</code>
        </command>
        <command id="bar" lang="bash">
            <code>echo world</code>
        </command>
        <command id="sleep" timeout="5">
           <code> sleep 1000</code>
        </command>
    </commands>
    <files id="db1">
        <dir id="binlog1">
            <root>
            	/data/mysql0
            </root>
            <allow>get</allow>
            <allow>delete</allow><!-- put, post -->
            <pattern>log-bin</pattern>
        </dir>
    </files>
    <user id="db_ha">
        <key>&_var.foo;</key>
        <host>10.200.180.11 </host>
        <files id="db1" />
        <commands id="db1" />
    </user>
</config>`
	xconf, err := XConfigFromData([]byte(data), map[string]string{
		"_var.foo": "FOO",
	})
	if err != nil {
		t.Errorf("parse error: %s", err)
		return
	}
	conf := xconf.ToConfig()
	if len(conf.Commands) != 1 {
		t.Errorf("parse commands failed")
		return
	}
	if len(conf.Commands) != 1 {

	}
	if _, ok := conf.Commands["db1"]; !ok {
		t.Errorf("commands name wrong")
	}
	if len(conf.Commands["db1"].Commands) != 3 {
		t.Errorf("commands members wrong")
		return
	}
	foo, ok := conf.Commands["db1"].Commands["foo"]
	if !ok {
		t.Errorf("command name wrong")
		return
	}
	if foo.Code != "echo hello" {
		t.Errorf("command code wrong")
	}
	if foo.Timeout != math.MaxUint32 {
		t.Errorf("timeout code wrong")
	}
	bar, ok := conf.Commands["db1"].Commands["bar"]
	if bar.Code != "echo world" {
		t.Errorf("command code wrong")
	}
	if bar.Lang != "bash" {
		t.Errorf("command lang wrong")
	}
	if bar.Timeout != math.MaxUint32 {
		t.Errorf("timeout code wrong")
	}
	sleep, ok := conf.Commands["db1"].Commands["sleep"]

	if sleep.Code != "sleep 1000" {
		t.Errorf("command code wrong")
	}
	if sleep.Lang != "" {
		t.Errorf("command lang wrong")
	}
	if sleep.Timeout != 5 {
		t.Errorf("timeout code wrong")
	}

	if len(conf.Files) != 1 {
		t.Errorf("parse files failed")
	}
	if _, ok := conf.Files["db1"]; !ok {
		t.Errorf("files name wrong")
		return
	}
	if len(conf.Files["db1"].Dirs) != 1 {
		t.Errorf("files members wrong")
	}
	binlog1, ok := conf.Files["db1"].Dirs["binlog1"]
	if !ok {
		t.Errorf("dir name wrong")
		return
	}
	if binlog1.Root != "/data/mysql0" {
		t.Errorf("dir root wrong")
	}
	if binlog1.Patterns[0] != "log-bin" {
		t.Errorf("dir pattern wrong")
	}
	if len(binlog1.Allows) != 2 {
		t.Errorf("dir allows wrong")
	}
	sort.Strings(binlog1.Allows)
	if binlog1.Allows[0] != "DELETE" {
		t.Errorf("allows 0 not DELETE")
	}
	if binlog1.Allows[1] != "GET" {
		t.Errorf("allows 0 not get")
	}
	if conf.Users["db_ha"].Key != "FOO" {
		t.Error("entity parse wrong")
	}
	//fmt.Printf("%v\n", conf)
}

func TestFile(t *testing.T) {
	conf, err := XConfigFromFile("../../../conf/example.xml", make(map[string]string))
	if err != nil {
		t.Errorf("parse error: %s", err)
		return
	}
	fmt.Printf("%v\n", conf.ToConfig())
}


