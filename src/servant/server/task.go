package server
import (
	"servant/conf"
	"time"
	"os/exec"
	"sync"
	"syscall"
	"os/signal"
	"os"
)


var taskProcesses = make(map[int] *exec.Cmd)
var taskProcessesLock sync.Mutex
var _isExiting bool = false
var sigHandlerOnce sync.Once

func registerProcess(cmd *exec.Cmd) {
	taskProcessesLock.Lock()
	taskProcesses[cmd.Process.Pid] = cmd
	taskProcessesLock.Unlock()
}

func unregisterProcess(cmd *exec.Cmd) {
	taskProcessesLock.Lock()
	delete(taskProcesses, cmd.Process.Pid)
	taskProcessesLock.Unlock()
}

func isExiting() bool {
	taskProcessesLock.Lock()
	ret := _isExiting
	taskProcessesLock.Unlock()
	return ret
}

func RunTimer(name string, timerConf *conf.Timer) {
	if timerConf.Tick <= 0 {
		logger.Printf("WARN (_) [timer] %s tick not set", name)
		return
	}
	cmdConf := conf.Command {
		Lang: timerConf.Lang,
		Code: timerConf.Code,
		User: timerConf.User,
		Background: true,
		Timeout: timerConf.Deadline,
	}
	ticker := time.NewTicker(time.Duration(timerConf.Tick) * time.Second)
	logger.Printf("INFO (_) [timer] starting timer %s", name)
	for range(ticker.C) {
		if isExiting() {
			break
		}
		cmd, out, err := cmdFromConf(&cmdConf, requestParams(nil), nil)
		if err != nil {
			logger.Printf("WARN (_) [timer] create %s command failed: %s", name, err.Error())
			break
		}
		if out != nil {
			out.Close()
		}
		logger.Printf("INFO (_) [timer] command: %v", cmd.Args)
		err = cmd.Start()
		if err != nil {
			logger.Printf("WARN (_) [timer] start %s command failed: %s", name, err.Error())
			break
		}
		//registerProcess(cmd)
		ch := make(chan error, 1)
		go func() {
			err = cmd.Wait()
			ch <- err
		}()
		timeout := time.Duration(cmdConf.Timeout)
		select {
		case err = <-ch:
			if err != nil {
				logger.Printf("WARN (_) [timer] %s command execution failed: %s", name, err.Error())
			}
		case <-time.After(timeout * time.Second):
			cmd.Process.Kill()
			logger.Printf("WARN (_) [timer] %s command execution timeout: %d", name, timeout)
		}
		//unregisterProcess(cmd)
	}
	ticker = nil
}

func RunDaemon(name string, daemonConf *conf.Daemon) {
	cmdConf := conf.Command {
		Lang: daemonConf.Lang,
		Code: daemonConf.Code,
		User: daemonConf.User,
		Background: true,
	}
	if daemonConf.Retries < 0 {
		daemonConf.Retries = 0
	}
	logger.Printf("INFO (_) [daemon] starting daemon %s", name)
	cleanupOnExit()
	for i := 0; i < daemonConf.Retries + 1; i++ {
		if isExiting() {
			return
		}
		cmd, out, err := cmdFromConf(&cmdConf, requestParams(nil), nil)
		if out != nil {
			out.Close()
		}
		if err != nil {
			logger.Printf("WARN (_) [daemon] create %s command failed: %s", name, err.Error())
			return
		}
		logger.Printf("INFO (_) [daemon] command: %v", cmd.Args)
		err = cmd.Start()
		if err != nil {
			logger.Printf("WARN (_) [daemon] start %s failed: %s", name, err.Error())
			return
		}
		logger.Printf("INFO (_) [daemon] %s started. pid: %d", name, cmd.Process.Pid)
		t0 := time.Now()
		registerProcess(cmd)
		err = cmd.Wait()
		unregisterProcess(cmd)
		if err == nil {
			logger.Printf("WARN (_) [daemon] %s normal exit", name)
			return
		}
		t1 := time.Now()
		if t1.Sub(t0) >= time.Duration(daemonConf.Live) * time.Second {
			i = 0
		}
	}
	logger.Printf("WARN (_) [daemon] %s give up after %d retries", name, daemonConf.Retries)
}

func cleanupOnExit() {
	sigHandlerOnce.Do(func(){
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			sig := <- sigChan
			logger.Printf("INFO (_) [daemon] got signal %s", sig.String())
			cleanupProcesses()
			logger.Println("INFO (_) [daemon] cleaning up done")
			os.Exit(0)
		}()
	})
}

func cleanupProcesses() {
	logger.Println("INFO (_) [daemon] cleaning up process")
	taskProcessesLock.Lock()
	_isExiting = true
	taskProcessesLock.Unlock()

	for i := 0; i < 10; i++ { // in about 100ms
		taskProcessesLock.Lock()
		for _, cmd := range(taskProcesses) {
			if cmd.Process.Signal(syscall.SIGTERM) != nil {
				delete(taskProcesses, cmd.Process.Pid)
				logger.Printf("INFO (_) [daemon] process %d terminated", cmd.Process.Pid)
			}
		}
		taskProcessesLock.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	for _, cmd := range(taskProcesses) {
		logger.Printf("INFO (_) [daemon] killing process %d ", cmd.Process.Pid)
		cmd.Process.Kill()
	}
	taskProcessesLock.Lock()
	taskProcesses = map[int]*exec.Cmd{}
	taskProcessesLock.Unlock()
}
