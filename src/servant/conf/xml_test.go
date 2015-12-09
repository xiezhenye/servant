package conf

import (
	"testing"
	"fmt"
	"sort"
)

func TestConfig(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8" ?>
<config>
	<server><listen>:2465</listen></server>
    <commands id="db1">
        <command id="foo">
            echo hello
        </command>
        <command id="bar" lang="bash">
            echo world
        </command>
        <command id="sleep" timeout="5">
            sleep 1000
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
        <key>s4xuF@P0qj28fwL</key>
        <host>10.200.180.11 </host>
        <files id="db1" />
        <commands id="db1" />
    </user>
</config>`
	conf, err := XConfigFromData([]byte(data))
	if err != nil {
		t.Errorf("parse error: %s", err)
		return
	}

	if len(conf.Commands) != 1 {
		t.Errorf("parse commands failed")
		return
	}
	if conf.Commands[0].Name != "db1" {
		t.Errorf("commands name wrong")
	}
	if len(conf.Commands[0].Commands) != 3 {
		t.Errorf("commands members wrong")
		return
	}
	if conf.Commands[0].Commands[0].Name != "foo" {
		t.Errorf("command name wrong")
	}
	if conf.Commands[0].Commands[0].Code != "echo hello" {
		t.Errorf("command code wrong")
	}
	if conf.Commands[0].Commands[0].Timeout != 0 {
		t.Errorf("timeout code wrong")
	}
	if conf.Commands[0].Commands[1].Name != "bar" {
		t.Errorf("command name wrong")
	}
	if conf.Commands[0].Commands[1].Code != "echo world" {
		t.Errorf("command code wrong")
	}
	if conf.Commands[0].Commands[1].Lang != "bash" {
		t.Errorf("command lang wrong")
	}
	if conf.Commands[0].Commands[1].Timeout != 0 {
		t.Errorf("timeout code wrong")
	}
	if conf.Commands[0].Commands[2].Name != "sleep" {
		t.Errorf("command name wrong")
	}
	if conf.Commands[0].Commands[2].Code != "sleep 1000" {
		t.Errorf("command code wrong")
	}
	if conf.Commands[0].Commands[2].Lang != "" {
		t.Errorf("command lang wrong")
	}
	if conf.Commands[0].Commands[2].Timeout != 5 {
		t.Errorf("timeout code wrong")
	}

	if len(conf.Files) != 1 {
		t.Errorf("parse files failed")
	}
	if conf.Files[0].Name != "db1" {
		t.Errorf("files name wrong")
	}
	if len(conf.Files[0].Dirs) != 1 {
		t.Errorf("files members wrong")
	}
	if conf.Files[0].Dirs[0].Name != "binlog1" {
		t.Errorf("dir name wrong")
	}
	if conf.Files[0].Dirs[0].Root != "/data/mysql0" {
		t.Errorf("dir root wrong")
	}
	if conf.Files[0].Dirs[0].Pattern != "log-bin" {
		t.Errorf("dir pattern wrong")
	}
	if len(conf.Files[0].Dirs[0].Allow) != 2 {
		t.Errorf("dir allows wrong")
	}
	sort.Strings(conf.Files[0].Dirs[0].Allow)
	if conf.Files[0].Dirs[0].Allow[0] != "delete" {
		t.Errorf("allows 0 not delete")
	}
	if conf.Files[0].Dirs[0].Allow[1] != "get" {
		t.Errorf("allows 0 not get")
	}
	//fmt.Printf("%v\n", conf)
}

func TestFile(t *testing.T) {
	conf, err := XConfigFromFile("../../../conf/example.xml")
	if err != nil {
		t.Errorf("parse error: %s", err)
	}
	fmt.Printf("%v\n", conf.ToConfig())
}


