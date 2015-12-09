package conf

type Config struct {
	Server   Server
	Users    map[string]*User
	Commands map[string]*Commands
	Files    map[string]*Files
}

type Server struct {
	Listen string
}

type User struct {
	Hosts     []string
	Key       string
	Files     []string
	Commands  []string
}

type Commands struct {
	Commands map[string]*Command
}

type Command struct {
	Lang    string
	Code    string
	Timeout uint32
}

type Files struct {
	Dirs   map[string]*Dir
}

type Dir struct {
	Root     string
	Allow    []string
	Pattern  string
}
