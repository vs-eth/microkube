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

package kube

import (
	"errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/helpers"
	"io"
	"io/ioutil"
	"path"
	"strings"
)

// KubeSchedulerHandler handles invocation of the kubernetes scheduler binary
type KubeSchedulerHandler struct {
	handlers.BaseServiceHandler
	cmd *helpers.CmdHandler

	// Path to kubelet binary
	binary string

	// Path to kubeconfig
	kubeconfig string
	// Path to scheduler config (!= kubeconfig, replacement for commandline flags)
	config string
	// Output handler
	out handlers.OutputHander
}

// NewKubeSchedulerHandler creates a KubeSchedulerHandler from the arguments provided
func NewKubeSchedulerHandler(binary, root, kubeconfig string, out handlers.OutputHander, exit handlers.ExitHandler) (*KubeSchedulerHandler, error) {
	obj := &KubeSchedulerHandler{
		binary:     binary,
		cmd:        nil,
		out:        out,
		kubeconfig: kubeconfig,
		config:     path.Join(root, "kube-scheduler.cfg"),
	}

	err := CreateKubeSchedulerConfig(obj.config, kubeconfig)
	if err != nil {
		return nil, err
	}

	obj.BaseServiceHandler = *handlers.NewHandler(exit, obj.healthCheckFun, "http://localhost:10251/healthz",
		obj.stop, obj.Start, nil, nil)
	return obj, nil
}

// Stop the child process
func (handler *KubeSchedulerHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

// See interface docs
func (handler *KubeSchedulerHandler) Start() error {
	handler.cmd = helpers.NewCmdHandler(handler.binary, []string{
		"kube-scheduler",
		"--config",
		handler.config,
	}, handler.BaseServiceHandler.HandleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

// Handle result of a health probe
func (handler *KubeSchedulerHandler) healthCheckFun(responseBin *io.ReadCloser) error {
	str, err := ioutil.ReadAll(*responseBin)
	if err != nil {
		return err
	}
	if strings.Trim(string(str), " \r\n") != "ok" {
		return errors.New("Health != ok: " + string(str))
	}
	return nil
}
