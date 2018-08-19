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

// Handle a kubelet instance, that is the thing that actually schedules pods on nodes, interacting with docker
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
	out handlers.OutputHander
}

// NewKubeletHandler creates a KubeletHandler from the arguments provided
func NewKubeletHandler(binary, root, kubeconfig, listenAddress string, server, client, ca *pki.RSACertificate, out handlers.OutputHander, exit handlers.ExitHandler) (*KubeletHandler, error) {
	obj := &KubeletHandler{
		binary:         binary,
		kubeServerCert: server.CertPath,
		kubeServerKey:  server.KeyPath,
		kubeCACert:     ca.CertPath,
		cmd:            nil,
		out:            out,
		rootDir:        root,
		kubeconfig:     kubeconfig,
		listenAddress:  listenAddress,
		config:         path.Join(root, "kubelet.cfg"),
	}
	os.Mkdir(path.Join(root, "kubelet"), 0770)
	os.Mkdir(path.Join(root, "staticpods"), 0770)

	err := CreateKubeletConfig(obj.config, server, ca, path.Join(root, "staticpods"))
	if err != nil {
		return nil, err
	}

	obj.BaseServiceHandler = *handlers.NewHandler(exit, obj.healthCheckFun, "http://localhost:10248/healthz",
		obj.stop, obj.Start, ca, client)
	return obj, nil
}

// Stop the child process
func (handler *KubeletHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

// See interface docs
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
