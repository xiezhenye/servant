package conf
import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"path"
	"math"
)

type XConfig struct {
	XMLName  xml.Name    `xml:"config"`
	Server   XServer     `xml:"server"`
	Auth     XAuth       `xml:"auth"`
	Users    []XUser     `xml:"user"`
	Commands []XCommands `xml:"commands"`
	Files    []XFiles    `xml:"files"`
}

type XServer struct {
	Listen string      `xml:"listen"`
}

type XAuth struct {
	Enabled       bool     `xml:"enabled,attr"`
	MaxTimeDelta  uint32   `xml:"maxTimeDelta"`
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
	Name         string  `xml:"id,attr"`
	Lang         string	 `xml:"lang,attr"`
	Code         string  `xml:"code"`
	Timeout      uint32  `xml:"timeout,attr"`
	User		 string  `xml:"runas,attr`
	Lock         XLock   `xml:"lock"`
}

type XLock struct {
	Name     string  `xml:"id,attr"`
	Timeout  uint    `xml:"timeout,attr"`
	Wait     bool    `xml:"wait,attr"`
}

type XFiles struct {
	Name   string       `xml:"id,attr"`
	Dirs   []XDir       `xml:"dir"`
}

type XDir struct {
	Name      string    `xml:"id,attr"`
	Root      string    `xml:"root"`
	Allows    []string  `xml:"allow"`
	Patterns  []string  `xml:"pattern"`
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
	return &ret, nil
}

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
	ret.Auth = Auth {
		Enabled:      conf.Auth.Enabled,
		MaxTimeDelta: conf.Auth.MaxTimeDelta,
	}
	ret.Files = make(map[string]*Files)
	for _, file := range(conf.Files) {
		fname := file.Name
		ret.Files[fname] = &Files{}
		ret.Files[fname].Dirs = make(map[string]*Dir)
		for _, dir := range(file.Dirs) {
			dname := dir.Name
			ret.Files[fname].Dirs[dname] = &Dir{}
			ret.Files[fname].Dirs[dname].Root = path.Clean(strings.TrimSpace(dir.Root))
			ret.Files[fname].Dirs[dname].Allows = make([]string, 0, 4)
			ret.Files[fname].Dirs[dname].Patterns = make([]string, 0, 4)
			for _, method := range(dir.Allows) {
				ret.Files[fname].Dirs[dname].Allows = append(ret.Files[fname].Dirs[dname].Allows,
					strings.ToUpper(strings.TrimSpace(method)))
			}
			for _, pattern := range(dir.Patterns) {
				ret.Files[fname].Dirs[dname].Patterns = append(ret.Files[fname].Dirs[dname].Patterns,
					strings.TrimSpace(pattern))
			}
		}
	}
	ret.Commands = make(map[string]*Commands)
	for _, commands := range(conf.Commands) {
		csname := commands.Name
		ret.Commands[csname] = &Commands{}
		ret.Commands[csname].Commands = make(map[string]*Command)
		for _, command := range(commands.Commands) {
			cname := command.Name
			ret.Commands[csname].Commands[cname] = &Command{}
			ret.Commands[csname].Commands[cname].Code = strings.TrimSpace(command.Code)
			ret.Commands[csname].Commands[cname].Lang = command.Lang
			if command.Timeout == 0 {
				command.Timeout = math.MaxUint32
			}
			ret.Commands[csname].Commands[cname].Timeout = command.Timeout
			ret.Commands[csname].Commands[cname].Lock.Name = strings.TrimSpace(command.Lock.Name)
			if command.Lock.Timeout == 0 {
				command.Lock.Timeout = math.MaxUint32
			}
			ret.Commands[csname].Commands[cname].Lock.Timeout = command.Lock.Timeout
			ret.Commands[csname].Commands[cname].Lock.Wait = command.Lock.Wait
		}
	}
	ret.Users = make(map[string]*User)
	for _, user := range(conf.Users) {
		uname := user.Name
		ret.Users[uname] = &User{}
		ret.Users[uname].Key = strings.TrimSpace(user.Key)
		ret.Users[uname].Hosts = make([]string, len(user.Hosts))
		for j := range(user.Hosts) {
			ret.Users[uname].Hosts[j] = strings.TrimSpace(user.Hosts[j])
		}
		ret.Users[uname].Commands = make([]string, 0, 2)
		ret.Users[uname].Files = make([]string, 0, 2)
		for _, command := range(user.Commands) {
			ret.Users[uname].Commands = append(ret.Users[uname].Commands, command.Name)
		}
		for _, file := range(user.Files) {
			ret.Users[uname].Files = append(ret.Users[uname].Files, file.Name)
		}
	}
	return &ret
}
