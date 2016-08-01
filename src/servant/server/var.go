package server
import (
	"strings"
	"regexp"
	"os"
	"sync"
	"servant/conf"
	"net/http"
	"io/ioutil"
)

var globalParams = map[string]string {}

type VarServer struct {
	*Session
}
const MaxVarValueSize = 1024

func NewVarServer(sess *Session) Handler {
	return &VarServer{
		Session:sess,
	}
}

func (self *VarServer) serve() {
	switch self.req.Method {
	case "GET":
		v, ok := GetUserVar(self.group, self.item)
		if !ok {
			self.ErrorEnd(http.StatusNotFound, "var %s.%s not found", self.group, self.item)
			return
		}
		self.resp.Write([]byte(v))
	case "PUT", "POST":
		if self.req.ContentLength > MaxVarValueSize {
			self.ErrorEnd(http.StatusBadRequest, "var value too large")
			return
		}
		if self.config.Vars[self.group] == nil || self.config.Vars[self.group].Vars[self.item] == nil {
			self.ErrorEnd(http.StatusNotFound, "var %s.%s not found", self.group, self.item)
			return
		}
		varConf := self.config.Vars[self.group].Vars[self.item]
		if varConf.Readonly {
			self.ErrorEnd(http.StatusForbidden, "var %s.%s is readonly", self.group, self.item)
			return
		}
		value, err := ioutil.ReadAll(self.req.Body)
		if err != nil {
			self.ErrorEnd(http.StatusInternalServerError, "read var value failed: %s", err.Error())
			return
		}
		matches := false
		for _, pattern := range varConf.Patterns {
			if matches, _ = regexp.Match(pattern, value); matches {
				matches = true
				break
			}
		}
		if !matches {
			self.ErrorEnd(http.StatusForbidden, "value not match patterns")
			return
		}
		SetUserVar(self.group, self.item, string(value))
	default:
		self.ErrorEnd(http.StatusMethodNotAllowed, "method %s not allowed", self.req.Method)
	}
}

func SetArgVars(params []string) {
	setVars(params, "_arg.")
}

func setVars(params []string, prefix string) {
	for _, s := range(params) {
		kv := strings.SplitN(s, "=", 2)
		if match, _ := regexp.MatchString(`^[a-zA-Z]\w*$`, kv[0]); match {
			globalParams[prefix + kv[0]] = kv[1]
		}
	}
}

func SetEnvVars() {
	setVars(os.Environ(), "_env.")
}

var varsLock = sync.Mutex{}

func SetUserVar(group, name string, value string) {
	k := group + "." + name
	SetGlobalParam(k, value)
}

func UserVarExists(group, name string) bool {
	k := group + "." + name
	return GlobalParamExists(k)
}

func SetGlobalParam(k string, value string) {
	varsLock.Lock()
	globalParams[k] = value
	varsLock.Unlock()
}

func GlobalParamExists(k string) bool {
	varsLock.Lock()
	_, ok := globalParams[k]
	varsLock.Unlock()
	return ok
}

func GetUserVar(group, name string) (string, bool) {
	k := group + "." + name
	return GetGlobalParam(k)
}

func GetGlobalParam(k string) (string, bool) {
	varsLock.Lock()
	ret, ok := globalParams[k]
	varsLock.Unlock()
	return ret, ok
}

var paramsCanExpand = map[string]bool{}
const MaxVarExpandDepth = 10
func GetVarCanExpand(k string) bool {
	varsLock.Lock()
	ret, exists := paramsCanExpand[k]
	varsLock.Unlock()
	return ret && exists
}

func SetVarCanExpand(k string, b bool) {
	varsLock.Lock()
	if b {
		paramsCanExpand[k] = true
	} else {
		delete(paramsCanExpand, k)
	}
	varsLock.Unlock()
}


func CloneGlobalParams() map[string]string {
	ret := make(map[string]string)
	varsLock.Lock()
	for k, v := range(globalParams) {
		ret[k] = v
	}
	varsLock.Unlock()
	return ret
}

func ValidateParams(vs conf.Validators, params ParamFunc) bool {
	if vs == nil {
		return true
	}
	for k, vd := range vs {
		v, ok := params(k)
		if !ok {
			return false
		}
		ret, err := regexp.MatchString(vd.Pattern, v)
		if err != nil || !ret{
			return false
		}
	}
	return true
}
