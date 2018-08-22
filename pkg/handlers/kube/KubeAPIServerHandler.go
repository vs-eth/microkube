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

// KubeAPIServerHandler handles invocation of the kubernetes apiserver
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
	out handlers.OutputHandler
	// Listen address
	listenAddress string
	// Service network in CIDR notation
	serviceNet string
}

// NewKubeAPIServerHandler creates a KubeAPIServerHandler from the arguments provided
func NewKubeAPIServerHandler(execEnv handlers.ExecutionEnvironment, creds *pki.MicrokubeCredentials, serviceNet string) *KubeAPIServerHandler {
	obj := &KubeAPIServerHandler{
		binary:         execEnv.Binary,
		kubeServerCert: creds.KubeServer.CertPath,
		kubeServerKey:  creds.KubeServer.KeyPath,
		kubeClientCert: creds.KubeClient.CertPath,
		kubeClientKey:  creds.KubeClient.KeyPath,
		kubeCACert:     creds.KubeCA.CertPath,
		etcdClientCert: creds.EtcdClient.CertPath,
		etcdClientKey:  creds.EtcdClient.KeyPath,
		etcdCACert:     creds.EtcdCA.CertPath,
		cmd:            nil,
		out:            execEnv.OutputHandler,
		listenAddress:  execEnv.ListenAddress.String(),
		serviceNet:     serviceNet,
		svcCert:        creds.KubeSvcSignCert.CertPath,
		svcKey:         creds.KubeSvcSignCert.KeyPath,
	}
	obj.BaseServiceHandler = *handlers.NewHandler(execEnv.ExitHandler, obj.healthCheckFun,
		"https://"+obj.listenAddress+":7443/healthz", obj.stop, obj.Start, creds.KubeCA, creds.KubeClient)
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
func kubeApiServerConstructor(execEnv handlers.ExecutionEnvironment, creds *pki.MicrokubeCredentials) ([]handlers.ServiceHandler, error) {
	handlerList, oCreds, err := helpers.StartHandlerForTest("etcd", "etcd", etcd.EtcdHandlerConstructor(2379), execEnv.ExitHandler, false, 1, creds)
	if err != nil {
		return handlerList, errors.Wrap(err, "etcd startup prereq failed")
	}
	if oCreds != creds {
		return handlerList, errors.Wrap(err, "two sets of credentials")
	}
	handlerList = append(handlerList, NewKubeAPIServerHandler(execEnv, creds, "127.0.0.0/16"))

	return handlerList, nil
}
