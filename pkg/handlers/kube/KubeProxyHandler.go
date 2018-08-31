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
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vs-eth/microkube/pkg/handlers"
	"github.com/vs-eth/microkube/pkg/helpers"
	"github.com/vs-eth/microkube/pkg/pki"
	"io"
	"io/ioutil"
	"path"
	"strconv"
)

// KubeProxyHandler handles invocation of the kubernetes proxy
type KubeProxyHandler struct {
	// Base ref
	handlers.BaseServiceHandler
	// command exec helper
	cmd *helpers.CmdHandler

	// Path to kube proxy binary
	binary string
	// Path to some sudo-like binary
	sudoBin string
	// Path to kubeconfig
	kubeconfig string
	// Path to proxy config (!= kubeconfig, replacement for commandline flags)
	config string
	// Cluster cidr
	clusterCIDR string
	// Output handler
	out handlers.OutputHandler
}

// NewKubeProxyHandler creates a KubeProxyHandler from the arguments provided
func NewKubeProxyHandler(execEnv handlers.ExecutionEnvironment, creds *pki.MicrokubeCredentials, cidr string) (*KubeProxyHandler, error) {
	obj := &KubeProxyHandler{
		binary:     execEnv.Binary,
		cmd:        nil,
		out:        execEnv.OutputHandler,
		kubeconfig: creds.Kubeconfig,
		config:     path.Join(execEnv.Workdir, "kube-proxy.cfg"),
		sudoBin:    execEnv.SudoMethod,
	}

	err := CreateKubeProxyConfig(obj.config, cidr, creds.Kubeconfig, execEnv)
	if err != nil {
		return nil, err
	}

	obj.BaseServiceHandler = *handlers.NewHandler(execEnv.ExitHandler, obj.healthCheckFun, "http://localhost:"+strconv.Itoa(execEnv.KubeProxyHealthPort)+"/healthz",
		obj.stop, obj.Start, nil, nil)
	return obj, nil
}

// Stop the child process
func (handler *KubeProxyHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

// Start starts the process, see interface docs
func (handler *KubeProxyHandler) Start() error {
	handler.cmd = helpers.NewCmdHandler(handler.sudoBin, []string{
		handler.binary,
		"kube-proxy",
		"--config",
		handler.config,
	}, handler.BaseServiceHandler.HandleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

// Handle result of a health probe
func (handler *KubeProxyHandler) healthCheckFun(responseBin *io.ReadCloser) error {
	type KubeProxyStatus struct {
		LastUpdated string `json:"lastUpdated"`
		CurrentTime string `json:"currentTime"`
	}
	response := KubeProxyStatus{}
	err := json.NewDecoder(*responseBin).Decode(&response)
	if err != nil {
		return errors.Wrap(err, "JSON decode of response failed")
	}
	if response.LastUpdated == "" || response.CurrentTime == "" {
		return errors.Wrap(err, "kube-proxy is unhealthy!")
	}
	return nil
}

// kubeProxyConstructor is supposed to be only used for testing
func kubeProxyConstructor(execEnv handlers.ExecutionEnvironment,
	creds *pki.MicrokubeCredentials) ([]handlers.ServiceHandler, error) {

	// Start apiserver (and etcd)
	handlerList, _, _, err := helpers.StartHandlerForTest(-1, "kube-apiserver", "hyperkube",
		kubeApiServerConstructor, execEnv.ExitHandler, false, 30, creds, &execEnv)
	if err != nil {
		return handlerList, fmt.Errorf("kube-apiserver startup prereq failed %s", err)
	}

	// Generate kubeconfig
	tmpdir, err := ioutil.TempDir("", "microkube-unittests-kubeconfig")
	if err != nil {
		return handlerList, fmt.Errorf("tempdir creation failed: %s", err)
	}
	kubeconfig := path.Join(tmpdir, "kubeconfig")
	err = CreateClientKubeconfig(execEnv, creds, kubeconfig, "127.0.0.1")
	if err != nil {
		return handlerList, fmt.Errorf("kubeconfig creation failed: %s", err)
	}

	handler, err := NewKubeProxyHandler(execEnv, creds, "1.0.0.0/1")
	if err != nil {
		return handlerList, fmt.Errorf("kubeProxy handler creation failed: %s", err)
	}
	handlerList = append(handlerList, handler)

	return handlerList, nil
}
