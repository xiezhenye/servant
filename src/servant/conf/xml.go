package conf
import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"path"
)

type XConfig struct {
	XMLName  xml.Name    `xml:"config"`
	Server   XServer     `xml:"server"`
	Users    []XUser     `xml:"user"`
	Commands []XCommands `xml:"commands"`
	Files    []XFiles    `xml:"files"`
}

type XServer struct {
	Listen string      `xml:"listen"`
}

type XUser struct {
	Name      string           `xml:"id,attr"`
	Hosts     []string         `xml:"host"`
	Key       string           `xml:"key"`
	Files     []XUserFiles     `xml:"files"`
	Commands  []XUserCommands  `xml:"commands"`
}

type XCommands struct {
	Name     string      `xml:"id,attr"`
	Commands []XCommand  `xml:"command"`
}

type XCommand struct {
	Name     string      `xml:"id,attr"`
	Lang     string	     `xml:"lang,attr"`
	Code     string      `xml:",chardata"`
	Timeout  uint32      `xml:"timeout,attr"`
}

type XFiles struct {
	Name   string       `xml:"id,attr"`
	Dirs   []XDir       `xml:"dir"`
}

type XDir struct {
	Name     string     `xml:"id,attr"`
	Root     string     `xml:"root"`
	Allow    []string   `xml:"allow"`
	Pattern  string     `xml:"pattern"`
}

type XUserFiles struct {
	Name   string   `xml:"id,attr"`
}

type XUserCommands struct {
	Name   string   `xml:"id,attr"`
}

func XConfigFromData(data []byte) (*XConfig, error) {
	ret := XConfig{}
	err := xml.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	//trimAll(&ret)
	return &ret, nil
}
/*
func trimAll(conf *XConfig) {
	for i := range(conf.Files) {
		for j := range(conf.Files[i].Dirs) {
			conf.Files[i].Dirs[j].Root = strings.TrimSpace(conf.Files[i].Dirs[j].Root)
			conf.Files[i].Dirs[j].Pattern = strings.TrimSpace(conf.Files[i].Dirs[j].Pattern)
			for k := range(conf.Files[i].Dirs[j].Allow) {
				conf.Files[i].Dirs[j].Allow[k] = strings.ToLower(strings.TrimSpace(conf.Files[i].Dirs[j].Allow[k]))
			}
		}
	}
	for i := range(conf.Commands) {
		for j := range(conf.Commands[i].Commands) {
			conf.Commands[i].Commands[j].Code = strings.TrimSpace(conf.Commands[i].Commands[j].Code)
		}
	}
	for i := range(conf.Users) {
		conf.Users[i].Key = strings.TrimSpace(conf.Users[i].Key)
		for j := range(conf.Users[i].Hosts) {
			conf.Users[i].Hosts[j] = strings.TrimSpace(conf.Users[i].Hosts[j])
		}
	}
}
*/
func XConfigFromReader(reader io.Reader) (*XConfig, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return XConfigFromData(data)
}

func XConfigFromFile(path string) (*XConfig, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return XConfigFromReader(reader)
}

func (conf *XConfig) ToConfig() *Config {
	ret := Config{}
	ret.Server = Server {
		Listen: conf.Server.Listen,
	}
	ret.Files = make(map[string]*Files)
	for i := range(conf.Files) {
		fname := conf.Files[i].Name
		ret.Files[fname] = &Files{}
		ret.Files[fname].Dirs = make(map[string]*Dir)
		for j := range(conf.Files[i].Dirs) {
			dname := conf.Files[i].Dirs[j].Name
			ret.Files[fname].Dirs[dname] = &Dir{}
			ret.Files[fname].Dirs[dname].Root = path.Clean(strings.TrimSpace(conf.Files[i].Dirs[j].Root))
			ret.Files[fname].Dirs[dname].Pattern = strings.TrimSpace(ret.Files[fname].Dirs[dname].Pattern)
			ret.Files[fname].Dirs[dname].Allow = make([]string, 0, 4)
			for k := range(conf.Files[i].Dirs[j].Allow) {
				act := strings.TrimSpace(conf.Files[i].Dirs[j].Allow[k])
				ret.Files[fname].Dirs[dname].Allow = append(ret.Files[fname].Dirs[dname].Allow, act)
			}
		}
	}
	ret.Commands = make(map[string]*Commands)
	for i := range(conf.Commands) {
		csname := conf.Commands[i].Name
		ret.Commands[csname] = &Commands{}
		ret.Commands[csname].Commands = make(map[string]*Command)
		for j := range(conf.Commands[i].Commands) {
			cname := conf.Commands[i].Commands[j].Name
			ret.Commands[csname].Commands[cname] = &Command{}
			ret.Commands[csname].Commands[cname].Code = strings.TrimSpace(conf.Commands[i].Commands[j].Code)
			ret.Commands[csname].Commands[cname].Lang = conf.Commands[i].Commands[j].Lang
			ret.Commands[csname].Commands[cname].Timeout = conf.Commands[i].Commands[j].Timeout
		}
	}
	ret.Users = make(map[string]*User)
	for i := range(conf.Users) {
		uname := conf.Users[i].Name
		ret.Users[uname] = &User{}
		ret.Users[uname].Key = strings.TrimSpace(conf.Users[i].Key)
		ret.Users[uname].Hosts = make([]string, len(conf.Users[i].Hosts))
		for j := range(conf.Users[i].Hosts) {
			ret.Users[uname].Hosts[j] = strings.TrimSpace(conf.Users[i].Hosts[j])
		}
		ret.Users[uname].Commands = make([]string, 0, 2)
		ret.Users[uname].Files = make([]string, 0, 2)
		for j := range(conf.Users[i].Commands) {
			ret.Users[uname].Commands = append(ret.Users[uname].Commands, conf.Users[i].Commands[j].Name)
		}
		for j := range(conf.Users[i].Files) {
			ret.Users[uname].Files = append(ret.Users[uname].Files, conf.Users[i].Commands[j].Name)
		}
	}
	return &ret
}
