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
	"github.com/uubk/microkube/pkg/pki"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// KubeletHandler handles a kubelet instance, that is the thing that actually schedules pods on nodes, interacting with
// docker
type KubeletHandler struct {
	handlers.BaseServiceHandler
	cmd *helpers.CmdHandler

	// Path to kubelet binary
	binary string
	// Path to kubernetes server certificate
	kubeServerCert string
	// Path to kubernetes server certificate's key
	kubeServerKey string
	// Path to kubernetes CA
	kubeCACert string

	// Where to bind?
	listenAddress string
	// Root dir of kubelet state
	rootDir string
	// Path to kubeconfig
	kubeconfig string
	// Path to kubelet config (!= kubeconfig, replacement for commandline flags)
	config string
	// Output handler
	out handlers.OutputHandler
}

// NewKubeletHandler creates a KubeletHandler from the arguments provided
func NewKubeletHandler(execEnv handlers.ExecutionEnvironment, creds *pki.MicrokubeCredentials) (*KubeletHandler, error) {
	obj := &KubeletHandler{
		binary:         execEnv.Binary,
		kubeServerCert: creds.KubeServer.CertPath,
		kubeServerKey:  creds.KubeServer.KeyPath,
		kubeCACert:     creds.KubeCA.CertPath,
		cmd:            nil,
		out:            execEnv.OutputHandler,
		rootDir:        execEnv.Workdir,
		kubeconfig:     creds.Kubeconfig,
		listenAddress:  execEnv.ListenAddress.String(),
		config:         path.Join(execEnv.Workdir, "kubelet.cfg"),
	}
	os.Mkdir(path.Join(execEnv.Workdir, "kubelet"), 0770)
	os.Mkdir(path.Join(execEnv.Workdir, "staticpods"), 0770)

	err := CreateKubeletConfig(obj.config, creds, path.Join(execEnv.Workdir, "staticpods"))
	if err != nil {
		return nil, err
	}

	obj.BaseServiceHandler = *handlers.NewHandler(execEnv.ExitHandler, obj.healthCheckFun, "http://localhost:10248/healthz",
		obj.stop, obj.Start, creds.KubeCA, creds.KubeClient)
	return obj, nil
}

// Stop the child process
func (handler *KubeletHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

// Start starts the process, see interface docs
func (handler *KubeletHandler) Start() error {
	handler.cmd = helpers.NewCmdHandler("pkexec", []string{
		handler.binary,
		"kubelet",
		"--config",
		handler.config,
		"--node-ip",
		handler.listenAddress,
		"--kubeconfig",
		handler.kubeconfig,
		"--cni-bin-dir",
		path.Join(handler.rootDir, "kubelet/cni"),
		"--root-dir",
		path.Join(handler.rootDir, "kubelet"),
		"--seccomp-profile-root",
		path.Join(handler.rootDir, "kubelet/seccomp"),
		"--bootstrap-checkpoint-path",
		path.Join(handler.rootDir, "kubelet/checkpoint"),
		"--network-plugin",
		"kubenet",
		"--runtime-cgroups",
		"/systemd/system.slice",
	}, handler.BaseServiceHandler.HandleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

// Handle result of a health probe
func (handler *KubeletHandler) healthCheckFun(responseBin *io.ReadCloser) error {
	str, err := ioutil.ReadAll(*responseBin)
	if err != nil {
		return err
	}
	if strings.Trim(string(str), " \r\n") != "ok" {
		return errors.New("Health != ok: " + string(str))
	}
	return nil
}

// TODO: Test this (somehow...)
