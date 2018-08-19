/*
 * Copyright 2018 The microkube authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package helpers contains utility functions needed to implement handlers
package helpers

import (
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"path"
)

// CmdHandler is used to abstract the low-level handling of exec.Command, providing callbacks for events
type CmdHandler struct {
	binary string
	args   []string
	cmd    *exec.Cmd
	exit   handlers.ExitHandler
	stdout handlers.OutputHander
	stderr handlers.OutputHander
}

// NewCmdHandler creates a CmdHandler for the arguments provided
func NewCmdHandler(binary string, args []string, exit handlers.ExitHandler, stdout handlers.OutputHander, stderr handlers.OutputHander) *CmdHandler {
	return &CmdHandler{
		binary: binary,
		args:   args,
		cmd:    nil,
		exit:   exit,
		stdout: stdout,
		stderr: stderr,
	}
}

// Stop stops a running process if there is one
func (handler *CmdHandler) Stop() {
	if handler.cmd != nil {
		handler.cmd.Process.Kill()
	}
}

// Start starts a new process and sets up all related handlers
func (handler *CmdHandler) Start() error {
	handler.cmd = exec.Command(handler.binary, handler.args...)
	// Detach from process group
	handler.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

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
		select { // Exit this because either...
		// ... we got a signal, therefore terminating the process
		case <-sigchan:
			handler.cmd.Process.Kill()
			return
		// ... we got an exit notification, terminating the routine
		case <-statechan:
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

// FindBinary tries to find binary 'name'. The following locations are checked in this order:
//  - cwd/../../../third_party/name
//  - cwd/../../third_party/name
//  - cwd/../third_party/name
//  - cwd/third_party/name
//  - 'appdir'/third_party/name
//  - 'extraDir'/name
func FindBinary(name string, appDir, extraDir string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "couldn't read cwd")
	}

	candidates := []string {
		path.Join(path.Dir(path.Dir(path.Dir(cwd))), "third_party"),
		path.Join(path.Dir(path.Dir(cwd)), "third_party"),
		path.Join(path.Dir(cwd), "third_party"),
		path.Join(cwd, "third_party"),
		path.Join(appDir, "third_party"),
		extraDir,
	}
	for _, candidate := range candidates {
		test := path.Join(candidate, name)
		_, err = os.Stat(test)
		if err == nil {
			return test, nil
		}
	}

	return "", errors.New("Couldn't find file")
}