package kube_apiserver

import (
		"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/helpers"
	"io"
	"io/ioutil"
	"strings"
)

type KubeAPIServerHandler struct {
	helpers.HandlerHelper
	binary         string
	kubeServerCert string
	kubeServerKey  string
	kubeClientCert string
	kubeClientKey  string
	kubeCACert     string
	etcdCACert     string
	etcdClientCert string
	etcdClientKey  string
	cmd            *helpers.CmdHandler
	out            helpers.OutputHander
}

func NewKubeAPIServerHandler(binary, kubeServerCert, kubeServerKey, kubeClientCert, kubeClientKey, kubeCACert,
	etcdClientCert, etcdClientKey, etcdCACert string, out helpers.OutputHander, exit helpers.ExitHandler) *KubeAPIServerHandler {
	obj := &KubeAPIServerHandler{
		binary:         binary,
		kubeServerCert: kubeServerCert,
		kubeServerKey:  kubeServerKey,
		kubeClientCert: kubeClientCert,
		kubeClientKey:  kubeClientKey,
		kubeCACert:     kubeCACert,
		etcdClientCert: etcdClientCert,
		etcdClientKey:  etcdClientKey,
		etcdCACert:     etcdCACert,
		cmd:            nil,
		out:            out,
	}
	obj.HandlerHelper = *helpers.NewHandlerHelper(exit, obj.healthCheckFun, "https://localhost:7443/healthz",
		obj.stop, obj.Start)
	return obj
}

func (handler *KubeAPIServerHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

func (handler *KubeAPIServerHandler) Start() error {
	handler.cmd = helpers.NewCmdHandler(handler.binary, []string{
		"--bind-address",
		"0.0.0.0",
		"--insecure-port",
		"0",
		"--secure-port",
		"7443",
		"--kubernetes-service-node-port",
		"7443",
		"--service-node-port-range",
		"7000-9000",
		"--allow-privileged",
		"--anonymous-auth",
		"false",
		"--authorization-mode",
		"RBAC",
		"--client-ca-file",
		handler.kubeCACert,
		"--etcd-cafile",
		handler.etcdCACert,
		"--etcd-certfile",
		handler.etcdClientCert,
		"--etcd-keyfile",
		handler.etcdClientKey,
		"--etcd-servers",
		"https://127.0.0.1:2379",
		"--kubelet-certificate-authority",
		handler.kubeCACert,
		"--kubelet-client-certificate",
		handler.kubeClientCert,
		"--kubelet-client-key",
		handler.kubeClientKey,
		"--tls-cert-file",
		handler.kubeServerCert,
		"--tls-private-key-file",
		handler.kubeServerKey,
	}, handler.HandlerHelper.HandleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

func (handler *KubeAPIServerHandler) healthCheckFun(responseBin *io.ReadCloser) error {
	str, err := ioutil.ReadAll(*responseBin)
	if err != nil {
		return err
	}
	if strings.Trim(string(str), " \r\n") != "ok" {
		return errors.New("Health != ok: " + string(str))
	}
	return nil
}
