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

// Package kube contains handlers for all kubernetes related services
package kube

import (
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/handlers/etcd"
	"github.com/uubk/microkube/pkg/helpers"
	"github.com/uubk/microkube/pkg/pki"
	"io"
	"io/ioutil"
	"strings"
)

// ControllerManagerHandler handles invocation of the kubernetes apiserver
type KubeAPIServerHandler struct {
	// Base ref
	handlers.BaseServiceHandler
	// command exec helper
	cmd *helpers.CmdHandler

	// Kube-apiserver binary location
	binary string
	// Path to kube-apiserver certificate
	kubeServerCert string
	// Path to kube-apiserver certificate key
	kubeServerKey string
	// Path to a client certificate signed by the same CA as the server certificate
	kubeClientCert string
	// Path to the key matching the client certificate
	kubeClientKey string
	// Path to CA used to sign the above certificates
	kubeCACert string
	// Path to etcd ca
	etcdCACert string
	// Path to a client certificate allowed to access etcd
	etcdClientCert string
	// Path to the key matching the above certificate
	etcdClientKey string
	// Service account signing cert public key
	svcCert string
	// Service account signing cert private key
	svcKey string
	// Output handler
	out handlers.OutputHander
	// Listen address
	listenAddress string
	// Service network in CIDR notation
	serviceNet string
}

// NewKubeAPIServerHandler creates a KubeAPIServerHandler from the arguments provided
func NewKubeAPIServerHandler(binary string, kubeServer, kubeClient, kubeCA, kubeSvc,
	etcdClient, etcdCA *pki.RSACertificate, out handlers.OutputHander, exit handlers.ExitHandler, listenAddress string, serviceNet string) *KubeAPIServerHandler {
	obj := &KubeAPIServerHandler{
		binary:         binary,
		kubeServerCert: kubeServer.CertPath,
		kubeServerKey:  kubeServer.KeyPath,
		kubeClientCert: kubeClient.CertPath,
		kubeClientKey:  kubeClient.KeyPath,
		kubeCACert:     kubeCA.CertPath,
		etcdClientCert: etcdClient.CertPath,
		etcdClientKey:  etcdClient.KeyPath,
		etcdCACert:     etcdCA.CertPath,
		cmd:            nil,
		out:            out,
		listenAddress:  listenAddress,
		serviceNet:     serviceNet,
		svcCert:        kubeSvc.CertPath,
		svcKey:         kubeSvc.KeyPath,
	}
	obj.BaseServiceHandler = *handlers.NewHandler(exit, obj.healthCheckFun, "https://"+listenAddress+":7443/healthz",
		obj.stop, obj.Start, kubeCA, kubeClient)
	return obj
}

// Stop the child process
func (handler *KubeAPIServerHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

// Start starts the process, see interface docs
func (handler *KubeAPIServerHandler) Start() error {
	handler.cmd = helpers.NewCmdHandler(handler.binary, []string{
		"kube-apiserver",
		"--bind-address",
		handler.listenAddress,
		"--secure-port",
		"7443",
		"--kubernetes-service-node-port",
		"7444",
		"--service-node-port-range",
		"7000-9000",
		"--service-cluster-ip-range",
		handler.serviceNet,
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
		"--service-account-key-file",
		handler.svcCert,
		"--service-account-key-file",
		handler.svcKey,
	}, handler.BaseServiceHandler.HandleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

// Handle result of a health probe
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

// kubeApiServerConstructor is supposed to be only used for testing
func kubeApiServerConstructor(ca, server, client *pki.RSACertificate, binary, workdir string, outputHandler handlers.OutputHander, exitHandler handlers.ExitHandler) ([]handlers.ServiceHandler, error) {
	handlerList, etcdCA, etcdClient, _, err := helpers.StartHandlerForTest("etcd", etcd.EtcdHandlerConstructor, exitHandler, false, 1)
	if err != nil {
		return handlerList, errors.Wrap(err, "etcd startup prereq failed")
	}
	handlerList = append(handlerList, NewKubeAPIServerHandler(binary, server, client, ca, ca, etcdClient, etcdCA, outputHandler, exitHandler, "0.0.0.0", "127.10.10.0/24"))

	return handlerList, nil
}
