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
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/helpers"
	"io"
	"path"
)

// KubeProxyHandler handles invocation of the kubernetes proxy
type KubeProxyHandler struct {
	// Base ref
	handlers.BaseServiceHandler
	// command exec helper
	cmd *helpers.CmdHandler

	// Path to kube proxy binary
	binary string
	// Path to kubeconfig
	kubeconfig string
	// Path to proxy config (!= kubeconfig, replacement for commandline flags)
	config string
	// Cluster cidr
	clusterCIDR string
	// Output handler
	out handlers.OutputHander
}

// NewKubeProxyHandler creates a KubeProxyHandler from the arguments provided
func NewKubeProxyHandler(binary, root, kubeconfig, cidr string, out handlers.OutputHander, exit handlers.ExitHandler) (*KubeProxyHandler, error) {
	obj := &KubeProxyHandler{
		binary:     binary,
		cmd:        nil,
		out:        out,
		kubeconfig: kubeconfig,
		config:     path.Join(root, "kube-proxy.cfg"),
	}

	err := CreateKubeProxyConfig(obj.config, cidr, kubeconfig)
	if err != nil {
		return nil, err
	}

	obj.BaseServiceHandler = *handlers.NewHandler(exit, obj.healthCheckFun, "http://localhost:10256/healthz",
		obj.stop, obj.Start, nil, nil)
	return obj, nil
}

// Stop the child process
func (handler *KubeProxyHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

// See interface docs
func (handler *KubeProxyHandler) Start() error {
	handler.cmd = helpers.NewCmdHandler("pkexec", []string{
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
