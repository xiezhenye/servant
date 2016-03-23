package server
import (
	"net/http"
	"regexp"
	"os/user"
	"os/exec"
	"servant/conf"
	"io/ioutil"
	"time"
	"fmt"
	"syscall"
	"strconv"
)

var paramRe, _ = regexp.Compile(`^\$\w+$`)
var cmdUrlRe, _ = regexp.Compile(`^/commands/(\w+)/(\w+)/?$`)
var argRe, _ = regexp.Compile(`"([^"]*)"|'([^']*)'|([^\s]+)`)

func (self *Session) findCommandConfigByPath(path string) *conf.Command {
	m := cmdUrlRe.FindStringSubmatch(path)
	if len(m) != 3 {
		return nil
	}
	cmdsConf, ok := self.config.Commands[m[1]]
	if !ok {
		return nil
	}
	cmdConf, ok := cmdsConf.Commands[m[2]]
	if !ok {
		return nil
	}
	return cmdConf
}

func getCmdBash(code string, query map[string][]string) *exec.Cmd{
	return exec.Command("bash", "-c", code)
}

func getCmdExec(code string, query map[string][]string) *exec.Cmd {
	argsMatches := argRe.FindAllStringSubmatch(code, -1)
	args := make([]string, 0, 8)
	for i := 0; i < len(argsMatches); i++ {
		arg := argsMatches[i][0]
		if paramRe.MatchString(argsMatches[i][0]) {
			v, ok := query[argsMatches[i][0][1:]]
			if ok {
				arg = v[0] // only the first arg with the name will be used
			} else {
				arg = ""
			}
		}
		args = append(args, arg)
	}
	return exec.Command(args[0], args[1:]...)
}

func (self *Session) serveCommand() {
	defer self.req.Body.Close()
	urlPath := self.req.URL.Path
	method := self.req.Method
	self.info("command", "+ %s %s %s", self.req.RemoteAddr, method, urlPath)
	err := self.auth()
	if err != nil {
		self.warn("command", "- auth failed: %s", err.Error())
		self.resp.WriteHeader(http.StatusForbidden)
		return
	}

	if method != "GET" && method != "POST" {
		self.warn("command", "- not allow method: %s", method)
		self.resp.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	cmdConf := self.findCommandConfigByPath(urlPath)
	if cmdConf == nil {
		self.warn("command", "- command %s not found", urlPath)
		self.resp.WriteHeader(http.StatusNotFound)
		return
	}

	if cmdConf.Lock.Name == "" {
		self.execCommand(cmdConf)
	} else {
		if cmdConf.Lock.Wait {
			GetLock(cmdConf.Lock.Name).TimeoutWith(time.Duration(cmdConf.Lock.Timeout) * time.Second, func() {
				self.execCommand(cmdConf)
			})
		} else {
			GetLock(cmdConf.Lock.Name).TryWith(func(){
				self.execCommand(cmdConf)
			})
		}
	}
}

func setCmdUser(cmd *exec.Cmd, username string) error {
	sysuser, err := user.Lookup(username)
	if err != nil {
		return err
	}
	uid , err := strconv.Atoi(sysuser.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(sysuser.Gid)
	if err != nil {
		return err
	}
	cred := syscall.Credential{ Uid: uint32(uid), Gid: uint32(gid) }
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &cred
	return nil
}

func (self *Session) execCommand(cmdConf *conf.Command) {
	var cmd *exec.Cmd
	switch cmdConf.Lang {
	case "exec":
		cmd = getCmdExec(cmdConf.Code, self.req.URL.Query())
	case "bash", "":
		cmd = getCmdBash(cmdConf.Code, self.req.URL.Query())
	default:
		self.warn("command", "- unknown language")
		self.resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	if cmdConf.User != "" {
		err := setCmdUser(cmd, cmdConf.User)
		if err != nil {
			self.warn("command", "- %s", err.Error())
			self.resp.Header().Set(ServantErrHeader, err.Error())
			self.resp.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	if self.req.Method == "POST" {
		cmd.Stdin = self.req.Body
	}

	cmd.Stderr = nil

	out, err := cmd.StdoutPipe()
	defer out.Close()
	if err != nil {
		self.warn("command", "- %s", err.Error())
		self.resp.Header().Set(ServantErrHeader, err.Error())
		self.resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	var outBuf []byte
	timeout := time.Duration(cmdConf.Timeout)
	ch := make(chan error, 1)
	go func() {
		err = cmd.Start()
		if err != nil {
			ch <- err
			return
		}
		outBuf, err = ioutil.ReadAll(out)
		if err != nil {
			ch <- err
			return
		}
		err = cmd.Wait()
		if err != nil {
			ch <- err
			return
		}
		ch <- nil
	}()
	select {
	case err = <-ch:
		if err != nil {
			self.warn("command", "- execution error: %s", err.Error())
			self.resp.Header().Set(ServantErrHeader, err.Error())
			self.resp.WriteHeader(http.StatusBadGateway)
			return
		}
	case <-time.After(timeout * time.Second):
		cmd.Process.Kill()
		err = fmt.Errorf("command execution timeout: %d", timeout)
		self.warn("command", "- %s", err.Error())
		self.resp.Header().Set(ServantErrHeader, err.Error())
		self.resp.WriteHeader(http.StatusGatewayTimeout)
		return
	}
	_, err = self.resp.Write(outBuf) // may log errors
	if err != nil {
		self.warn("command", "- io error: %s", err.Error())
	} else {
		self.info("command", "- execution done")
	}
}
