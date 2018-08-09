package helpers

import (
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"os/signal"
)

type CmdHandler struct {
	binary string
	args   []string
	cmd    *exec.Cmd
	exit   ExitHandler
	stdout OutputHander
	stderr OutputHander
}

func NewCmdHandler(binary string, args []string, exit ExitHandler, stdout OutputHander, stderr OutputHander) *CmdHandler {
	return &CmdHandler{
		binary: binary,
		args:   args,
		cmd:    nil,
		exit:   exit,
		stdout: stdout,
		stderr: stderr,
	}
}

func (handler *CmdHandler) Stop() {
	if handler.cmd != nil {
		handler.cmd.Process.Kill()
	}
}

func (handler *CmdHandler) Start() error {
	handler.cmd = exec.Command(handler.binary, handler.args...)

	// Handle stdout
	if handler.stdout != nil {
		pipe, err := handler.cmd.StdoutPipe()
		if err != nil {
			return errors.Wrap(err, "stdout pip creation failed")
		}
		go func() {
			buf := make([]byte, 256)
			for {
				n, err := pipe.Read(buf)
				if n > 0 {
					handler.stdout(buf[0:n])
				}
				if err != nil {
					break
				}
			}
		}()
	}

	// Handle stderr
	if handler.stderr != nil {
		pipe, err := handler.cmd.StderrPipe()
		if err != nil {
			return errors.Wrap(err, "stderr pip creation failed")
		}
		go func() {
			buf := make([]byte, 256)
			for {
				n, err := pipe.Read(buf)
				if n > 0 {
					handler.stderr(buf[0:n])
				}
				if err != nil {
					break
				}
			}
		}()
	}

	err := handler.cmd.Start()
	if err != nil {
		return errors.Wrap(err, "process start failed")
	}

	// In case this program is interrupted, stop the child!
	sigchan := make(chan os.Signal, 2)
	statechan := make(chan bool, 2)
	go func() {
		select {
		case _ = <-sigchan:
			handler.cmd.Process.Kill()
			return
		case _ = <-statechan:
			return
		}
	}()
	signal.Notify(sigchan, os.Interrupt, os.Kill)

	go func() {
		result := handler.cmd.Wait()
		statechan <- true
		if handler.exit != nil {
			if result == nil {
				handler.exit(true, nil)
			} else {
				if err, ok := result.(*exec.ExitError); ok {
					handler.exit(false, err)
				} else {
					handler.exit(false, nil)
				}
			}
		}
	}()
	return nil
}
