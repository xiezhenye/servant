package server
import (
	"net/http"
	"regexp"
	"os/user"
	"os/exec"
	"servant/conf"
	//"io/ioutil"
	"time"
	"syscall"
	"strconv"
	"io"
)

var argRe, _ = regexp.Compile(`("[^"]*"|'[^']*'|[^\s]+)`)

type CommandServer struct {
	*Session
}

func NewCommandServer(sess *Session) Handler {
	return CommandServer{
		Session:sess,
	}
}

func (self CommandServer) findCommandConfig() *conf.Command {
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

func getCmdBash(code string, query func(string)string) *exec.Cmd{
	return exec.Command("bash", "-c", code)
}

func replaceCmdParams(arg string, query func(string)string) string {
	return paramRe.ReplaceAllStringFunc(arg, func(s string) string {
		if query == nil {
			return ""
		}
		return query(s[2:len(s) - 1])
	})
}

func getCmdExec(code string, query func(string)string) *exec.Cmd {
	argsMatches := argRe.FindAllStringSubmatch(code, -1)
	args := make([]string, 0, 4)
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

func (self CommandServer) serve() {
	urlPath := self.req.URL.Path
	method := self.req.Method

	if method != "GET" && method != "POST" {
		self.ErrorEnd(http.StatusMethodNotAllowed, "not allow method: %s", method)
		return
	}
	cmdConf := self.findCommandConfig()
	if cmdConf == nil {
		self.ErrorEnd(http.StatusNotFound, "command %s not found", urlPath)
		self.resp.WriteHeader(http.StatusNotFound)
		return
	}
	if cmdConf.Lock.Name == "" {
		self.serveCommand(cmdConf)
	} else {
		if cmdConf.Lock.Wait {
			GetLock(cmdConf.Lock.Name).TimeoutWith(time.Duration(cmdConf.Lock.Timeout) * time.Second, func() {
				self.serveCommand(cmdConf)
			})
		} else {
			GetLock(cmdConf.Lock.Name).TryWith(func(){
				self.serveCommand(cmdConf)
			})
		}
	}
}

func setCmdUser(cmd *exec.Cmd, username string) error {
	sysUser, err := user.Lookup(username)
	if err != nil {
		return err
	}
	uid , err := strconv.Atoi(sysUser.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(sysUser.Gid)
	if err != nil {
		return err
	}
	cred := syscall.Credential{ Uid: uint32(uid), Gid: uint32(gid) }

	cmd.SysProcAttr.Credential = &cred
	return nil
}

func (self CommandServer) serveCommand(cmdConf *conf.Command) {
	outBuf, err := self.execCommand(cmdConf)
	if err != nil {
		self.ErrorEnd(err.(ServantError).HttpCode, err.(ServantError).Message)
		return
	}
	_, err = self.resp.Write(outBuf) // may log errors
	if err != nil {
		self.BadEnd("io error: %s", err)
	} else {
		self.GoodEnd("execution done")
	}
}


func cmdFromConf(cmdConf *conf.Command, params func(string)string, input io.ReadCloser) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	switch cmdConf.Lang {
	case "exec":
		cmd = getCmdExec(cmdConf.Code, params)
	case "bash", "":
		cmd = getCmdBash(cmdConf.Code, params)
	default:
		return nil, NewServantError(http.StatusInternalServerError, "unknown language")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.Dir = "/"
	if cmdConf.User != "" {
		err := setCmdUser(cmd, cmdConf.User)
		if err != nil {
			return nil, NewServantError(http.StatusInternalServerError, "set user failed: %s", err.Error())
		}
	}
	cmd.Stdin = input
	cmd.Stderr = nil
	if cmdConf.Background {
		//cmd.SysProcAttr.Setpgid = true
		//cmd.SysProcAttr.Pgid = 0
		cmd.SysProcAttr.Setsid = true
		cmd.SysProcAttr.Foreground = false
		cmd.Stdout = nil
		cmd.Stdin = nil
	}
	return cmd, nil
}

func (self CommandServer) execCommand(cmdConf *conf.Command) (outBuf []byte, err error) {
	var input io.ReadCloser = nil
	if self.req.Method == "POST" {
		input = self.req.Body
	}
	cmd, err := cmdFromConf(cmdConf, requestParams(self.req), input)
	if err != nil {
		return
	}
	err = cmd.Start()
	if err != nil {
		err = NewServantError(http.StatusBadGateway, "execution error: %s", err)
		return
	}
	self.info("process started. pid: %d", cmd.Process.Pid)
	if cmdConf.Background {
		go func() {
			err = cmd.Wait()
			if err != nil {
				self.warn("background process %d ended with error: %s", cmd.Process.Pid, err.Error())
			} else {
				self.info("background process %d ended", cmd.Process.Pid)
			}
		}()
	} else {
		ch := make(chan error, 1)
		go func() {
			if cmd.Stdout != nil {
				outBuf, err = cmd.Output()
				if err != nil {
					ch <- err
					return
				}
			}
			err = cmd.Wait()
			ch <- err
		}()
		timeout := time.Duration(cmdConf.Timeout)
		select {
		case err = <-ch:
			if err != nil {
				err = NewServantError(http.StatusBadGateway, "execution error: %s", err)
			}
		case <-time.After(timeout * time.Second):
			cmd.Process.Kill()
			err = NewServantError(http.StatusGatewayTimeout, "command execution timeout: %d", timeout)
		}
	}
	return
}
