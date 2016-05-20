package server
import (
	"servant/conf"
	"time"
)

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
	for _ = range(ticker.C) {
		cmd, err := cmdFromConf(&cmdConf, nil, nil)
		if err != nil {
			logger.Printf("WARN (_) [timer] create %s command failed: %s", name, err.Error())
			break
		}
		err = cmd.Start()
		if err != nil {
			logger.Printf("WARN (_) [timer] start %s command failed: %s", name, err.Error())
			break
		}
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
	for i := 0; i < daemonConf.Retries + 1; i++ {
		cmd, err := cmdFromConf(&cmdConf, nil, nil)
		if err != nil {
			logger.Printf("WARN (_) [daemon] create %s command failed: %s", name, err.Error())
			return
		}
		err = cmd.Start()
		if err != nil {
			logger.Printf("WARN (_) [daemon] start %s failed: %s", name, err.Error())
			return
		}
		logger.Printf("INFO (_) [daemon] %s started. pid: %d", name, cmd.Process.Pid)
		t0 := time.Now()
		err = cmd.Wait()
		if err == nil {
			logger.Printf("WARN (_) [daemon] %s normal exit", name)
			break
		}
		t1 := time.Now()
		if t1.Sub(t0) >= time.Duration(daemonConf.Live) * time.Second {
			i = 0
		}
	}
	logger.Printf("WARN (_) [daemon] %s give up after %d retries", name, daemonConf.Retries)
}
