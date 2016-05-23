package server
import (
	"strings"
	"regexp"
	"os"
	"sync"
)

var globalParams = map[string]string {}

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

func SetGlobalParam(k string, value string) {
	varsLock.Lock()
	globalParams[k] = value
	varsLock.Unlock()
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

