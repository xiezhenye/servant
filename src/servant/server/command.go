package server
import (
	"net/http"
	"regexp"
	"os/user"
	"os/exec"
	"servant/conf"
	"io/ioutil"
	"time"
	"syscall"
	"strconv"
)

var argRe, _ = regexp.Compile(`("[^"]*"|'[^']*'|[^\s]+)`)

type CommandServer struct {
	*Session
}

func NewCommandServer(sess *Session) Handler {
	return &CommandServer{
		Session:sess,
	}
}

func (self *CommandServer) findCommandConfigByPath() *conf.Command {
	cmdsConf, ok := self.config.Commands[self.group]
	if !ok {
		return nil
	}
	cmdConf, ok := cmdsConf.Commands[self.item]
	if !ok {
		return nil
	}
	return cmdConf
}

func getCmdBash(code string, query map[string][]string) *exec.Cmd{
	return exec.Command("bash", "-c", code)
}

func replaceCmdParams(arg string, query map[string][]string) string {
	return paramRe.ReplaceAllStringFunc(arg, func(s string) string {
		v, ok := query[s[2:len(s) - 1]]
		if ok {
			return v[0] // only the first arg with the name will be used
		}
		return ""
	})
}

func getCmdExec(code string, query map[string][]string) *exec.Cmd {
	argsMatches := argRe.FindAllStringSubmatch(code, -1)
	args := make([]string, 0, 8)
	for i := 0; i < len(argsMatches); i++ {
		arg := argsMatches[i][1]
		if arg[0] == '\'' || arg[0] == '"' {
			arg = arg[1 : len(arg)-1]
		}
		arg = replaceCmdParams(arg, query)
		args = append(args, arg)
	}
	return exec.Command(args[0], args[1:]...)
}

func (self *CommandServer) serve() {
	urlPath := self.req.URL.Path
	method := self.req.Method

	if method != "GET" && method != "POST" {
		self.ErrorEnd(http.StatusMethodNotAllowed, "not allow method: %s", method)
		return
	}
	cmdConf := self.findCommandConfigByPath()
	if cmdConf == nil {
		self.ErrorEnd(http.StatusNotFound, "command %s not found", urlPath)
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

func (self *CommandServer) execCommand(cmdConf *conf.Command) {
	var cmd *exec.Cmd
	switch cmdConf.Lang {
	case "exec":
		cmd = getCmdExec(cmdConf.Code, self.req.URL.Query())
	case "bash", "":
		cmd = getCmdBash(cmdConf.Code, self.req.URL.Query())
	default:
		self.ErrorEnd(http.StatusInternalServerError, "unknown language")
		return
	}
	if cmdConf.User != "" {
		err := setCmdUser(cmd, cmdConf.User)
		if err != nil {
			self.ErrorEnd(http.StatusInternalServerError, "set user failed: %s", err.Error())
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
		self.ErrorEnd(http.StatusInternalServerError, err.Error())
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
			self.ErrorEnd(http.StatusBadGateway, "execution error: %s", err)
			return
		}
	case <-time.After(timeout * time.Second):
		cmd.Process.Kill()
		self.ErrorEnd(http.StatusGatewayTimeout, "command execution timeout: %d", timeout)
		return
	}
	_, err = self.resp.Write(outBuf) // may log errors
	if err != nil {
		self.BadEnd("io error: %s", err)
	} else {
		self.GoodEnd("execution done")
	}
}
