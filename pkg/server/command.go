package server

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/xiezhenye/servant/pkg/conf"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var argRe, _ = regexp.Compile(`("[^"]*"|'[^']*'|[^\s"']+)`)

type CommandServer struct {
	*Session
}

func NewCommandServer(sess *Session) Handler {
	return CommandServer{
		Session: sess,
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

func getCmdBashArgs(code string, query ParamFunc) (string, []string) {
	return "bash", []string{"-c", code}
}

func replaceCmdParams(arg string, query ParamFunc) (string, bool) {
	return VarExpand(arg, query, func(s string) string { return s })
}

func getCmdExecArgs(code string, query ParamFunc) (string, []string, bool) {
	argsMatches := argRe.FindAllStringSubmatch(code, -1)
	args := make([]string, 0, 4)
	var exists bool
	for i := 0; i < len(argsMatches); i++ {
		arg := argsMatches[i][1]
		if arg[0] == '\'' || arg[0] == '"' {
			arg = arg[1 : len(arg)-1]
		}
		arg, exists = replaceCmdParams(arg, query)
		if !exists {
			return "", nil, false
		}
		args = append(args, arg)
	}
	return args[0], args[1:], true
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
		return
	}
	if cmdConf.Lock.Name == "" {
		self.serveCommand(cmdConf)
	} else {
		if cmdConf.Lock.Wait {
			GetLock(cmdConf.Lock.Name).TimeoutWith(time.Duration(cmdConf.Lock.Timeout)*time.Second, func() {
				self.serveCommand(cmdConf)
			})
		} else {
			GetLock(cmdConf.Lock.Name).TryWith(func() {
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
	uid, err := strconv.Atoi(sysUser.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(sysUser.Gid)
	if err != nil {
		return err
	}
	cred := syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}

	cmd.SysProcAttr.Credential = &cred
	return nil
}

func (self CommandServer) serveCommand(cmdConf *conf.Command) {
	outBuf, err := self.execCommand(cmdConf)
	if err != nil {
		if sErr, ok := err.(ServantError); ok {
			self.ErrorEnd(sErr.HttpCode, sErr.Message)
		} else {
			self.ErrorEnd(500, err.Error())
		}
		return
	}
	_, err = self.resp.Write(outBuf) // may log errors
	if err != nil {
		self.BadEnd("io error: %s", err)
	} else {
		self.GoodEnd("execution done")
	}
}

func cmdFromConf(cmdConf *conf.Command, params ParamFunc, input io.ReadCloser) (cmd *exec.Cmd, out io.ReadCloser, err error) {
	var name string
	var args []string
	if !ValidateParams(cmdConf.Validators, params) {
		return nil, nil, NewServantError(http.StatusBadRequest, "validate params failed")
	}
	code := strings.TrimSpace(cmdConf.Code)
	if code == "" {
		return nil, nil, NewServantError(http.StatusInternalServerError, "command code is empty")
	}
	switch cmdConf.Lang {
	case "exec":
		var exists bool
		name, args, exists = getCmdExecArgs(code, params)
		if !exists {
			err = NewServantError(http.StatusBadRequest, "some params missing")
			return
		}
	case "bash", "":
		name, args = getCmdBashArgs(code, params)
	default:
		err = NewServantError(http.StatusInternalServerError, "unknown language")
		return
	}
	cmd = exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.Dir = "/"
	if cmdConf.User != "" {
		err = setCmdUser(cmd, cmdConf.User)
		if err != nil {
			err = NewServantError(http.StatusInternalServerError, "set user failed: %s", err.Error())
			return
		}
	}
	buf := bytes.NewBuffer([]byte(""))
	if input != nil{
		inputData, err:= ioutil.ReadAll(input)
		if err!=nil{
			logger.Printf("read input error %s \n", err.Error())
		}
		buf=bytes.NewBuffer(inputData)
	}

	cmd.Stdin = buf
	cmd.Stderr = nil
	if cmdConf.Background {
		cmd.SysProcAttr.Setsid = true
		cmd.SysProcAttr.Foreground = false
		cmd.Stdout = nil
		// cmd.Stdin = nil
	} else {
		cmd.SysProcAttr.Setpgid = true
		cmd.SysProcAttr.Pgid = 0
		out, err = cmd.StdoutPipe()
		if err != nil {
			err = NewServantError(http.StatusInternalServerError, "pipe stdout failed: %s", err.Error())
			return
		}
	}
	return cmd, out, nil
}

func (self CommandServer) execCommand(cmdConf *conf.Command) (outBuf []byte, err error) {
	var input io.ReadCloser = nil
	if self.req.Method == "POST" {
		input = self.req.Body
	}
	params := requestParams(self.req)
	cmd, out, err := cmdFromConf(cmdConf, params, input)
	if err != nil {
		return
	}
	self.info("command: %v", cmd.Args)
	if out != nil {
		defer out.Close()
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
			if out != nil {
				if outBuf, err = ioutil.ReadAll(out); err != nil {
					ch <- err
					_ = cmd.Wait()
					return
				}
				if err = cmd.Wait(); err != nil {
					ch <- errors.WithMessage(err, string(outBuf))
					return
				}
				ch <- nil
				return
			}
			ch <- cmd.Wait()
		}()
		timeout := time.Duration(cmdConf.Timeout)
		select {
		case err = <-ch:
			if err != nil {
				err = NewServantError(http.StatusBadGateway, "execution error: %s", err)
			}
		case <-time.After(timeout * time.Second):
			_ = cmd.Process.Kill()
			err = NewServantError(http.StatusGatewayTimeout, "command execution timeout: %d", timeout)
		}
	}
	return
}
