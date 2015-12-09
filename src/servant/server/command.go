package server
import (
	"net/http"
	"regexp"
	"os/exec"
	"servant/conf"
	"io/ioutil"
	//"fmt"
	"time"
	"math"
	"fmt"
)

var paramRe, _ = regexp.Compile(`^\$\w+$`)
var cmdUrlRe, _ = regexp.Compile(`^/commands/(\w+)/(\w+)/?$`)
var spRe, _ = regexp.Compile(`\s+`)

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
	args := spRe.Split(code, -1)
	for i := 1; i < len(args); i++ {
		if paramRe.MatchString(args[i]) {
			v, ok := query[args[i][1:]]
			if ok {
				args[i] = v[0]
			} else {
				args[i] = ""
			}
		}
	}
	return exec.Command(args[0], args[1:]...)
}

func (self *Session) serveCommand(resp http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	if req.Method != "GET" && req.Method != "POST" {
		resp.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	cmdConf := self.findCommandConfigByPath(req.URL.Path)
	if cmdConf == nil {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
	var cmd *exec.Cmd
	switch cmdConf.Lang {
	case "exec":
		cmd = getCmdExec(cmdConf.Code, req.URL.Query())
	case "bash", "":
		cmd = getCmdBash(cmdConf.Code, req.URL.Query())
	default:
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	if req.Method == "POST" {
		cmd.Stdin = req.Body
	}

	cmd.Stderr = nil

	out, err := cmd.StdoutPipe()
	defer out.Close()
	if err != nil {
		resp.Header().Set("X-SERVANT-ERR", err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
	var outBuf []byte
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
	timeout := time.Duration(cmdConf.Timeout)
	if timeout <= 0 || timeout > math.MaxUint32 {
		timeout = math.MaxUint32
	}
	select {
	case err = <-ch:
		if err != nil {
			resp.Header().Set("X-SERVANT-ERR", err.Error())
			resp.WriteHeader(http.StatusBadGateway)
			return
		}
	case <-time.After(timeout * time.Second):
		cmd.Process.Kill()
		err = fmt.Errorf("command execution timeout")
		resp.Header().Set("X-SERVANT-ERR", err.Error())
		resp.WriteHeader(http.StatusGatewayTimeout)
		return
	}
	_, _ = resp.Write(outBuf) // may log errors
}
