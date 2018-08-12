package kube_scheduler

import (
	"errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/helpers"
	"io"
	"io/ioutil"
	"path"
	"strings"
)

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

func (handler *KubeSchedulerHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

func (handler *KubeSchedulerHandler) Start() error {
	handler.cmd = helpers.NewCmdHandler(handler.binary, []string{
		"--config",
		handler.config,
	}, handler.BaseServiceHandler.HandleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

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
