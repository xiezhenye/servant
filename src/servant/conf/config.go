package conf

type Config struct {
	Server     Server
	Users      map[string]*User
	Commands   map[string]*Commands
	Files      map[string]*Files
	Databases  map[string]*Database
	Vars       map[string]*Vars
	Timers     map[string]*Timer
	Daemons    map[string]*Daemon

	Auth       Auth
	Log        string
}


type Server struct {
	Listen string
}

type Auth struct {
	Enabled       bool
	MaxTimeDelta  uint32
}

type User struct {
	Hosts     []string
	Key       string
	Allows    map[string] []string
}

type Commands struct {
	Commands map[string]*Command
}

type Command struct {
	Lang         string
	Code         string
	Timeout      uint32
	User		 string
	Background   bool
	Validators   Validators
	Lock         Lock
}

type Database struct {
	Queries  map[string]*Query
	Driver   string
	Dsn      string
}

type Query struct {
	Sqls    []string
	Validators   Validators
}

type Lock struct {
	Name     string
	Timeout  uint
	Wait     bool
}

type Files struct {
	Dirs   map[string]*Dir
}

type Dir struct {
	Root       string
	Allows     []string
	Patterns   []string
	Validators Validators
}

type Vars struct {
	Vars map[string]*Var
}

type Var struct {
	Value    string
	Readonly bool
	Patterns []string
}

type Timer struct {
	Lang      string
	Code      string
	User      string
	Tick      int
	Deadline  uint32
}

type Daemon struct {
	Lang      string
	Code      string
	User      string
	Retries   int
	Live      int
}

type Validator struct {
	Name     string
	//class  string
	Pattern  string
}

type Validators map[string]Validator
