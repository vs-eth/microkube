package kube_proxy

import (
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/helpers"
	"io"
	"path"
	"encoding/json"
	"github.com/pkg/errors"
)

type KubeProxyHandler struct {
	handlers.BaseServiceHandler
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

func (handler *KubeProxyHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

func (handler *KubeProxyHandler) Start() error {
	// TODO(uubk)/XXX: kube-proxy unfortunately needs root privileges due to iptables invocations. Find a way around this or
	// use a sensible way to gain root. This only works when sudo can be done passwordless...
	handler.cmd = helpers.NewCmdHandler("sudo", []string{
		handler.binary,
		"kube-proxy",
		"--config",
		handler.config,
	}, handler.BaseServiceHandler.HandleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

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
