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

// Package etcd contains the handler for etcd
package etcd

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/helpers"
	"github.com/uubk/microkube/pkg/pki"
	"io"
	"strconv"
)

// EtcdHandler takes care of running a single etcd listening on (hardcoded) localhost.
type EtcdHandler struct {
	// Base ref
	handlers.BaseServiceHandler
	// command exec helper
	cmd *helpers.CmdHandler

	// Where should etcd's data be stored?
	datadir string
	// etcd binary location
	binary string
	// Client port (currently hardcoded to 2379)
	clientport int
	// Peer port (currently hardcoded to 2380)
	peerport int
	// Path to etcd server certificate
	servercert string
	// Path to etcd server certificate key
	serverkey string
	// Path to etcd ca certificate
	cacert string
	// Output handler
	out handlers.OutputHandler
	// Exit handler
	exit handlers.ExitHandler
}

// NewEtcdHandler creates an EtcdHandler from the arguments provided
func NewEtcdHandler(execEnv handlers.ExecutionEnvironment, creds *pki.MicrokubeCredentials) *EtcdHandler {
	obj := &EtcdHandler{
		datadir:    execEnv.Workdir,
		binary:     execEnv.Binary,
		clientport: execEnv.EtcdClientPort,
		peerport:   execEnv.EtcdPeerPort,
		servercert: creds.EtcdServer.CertPath,
		serverkey:  creds.EtcdServer.KeyPath,
		cacert:     creds.EtcdCA.CertPath,
		cmd:        nil,
		out:        execEnv.OutputHandler,
		exit:       execEnv.ExitHandler,
	}
	obj.BaseServiceHandler = *handlers.NewHandler(execEnv.ExitHandler, obj.healthCheckFun,
		"https://localhost:"+strconv.Itoa(obj.clientport)+"/health", obj.stop, obj.Start, creds.EtcdCA, creds.EtcdClient)
	return obj
}

// Start starts the process, see interface docs
func (handler *EtcdHandler) Start() error {
	handler.cmd = helpers.NewCmdHandler(handler.binary, []string{
		"--data-dir",
		handler.datadir,
		"--listen-peer-urls",
		"https://localhost:" + strconv.Itoa(handler.peerport),
		"--initial-advertise-peer-urls",
		"https://localhost:" + strconv.Itoa(handler.peerport),
		"--initial-cluster",
		"default=https://localhost:" + strconv.Itoa(handler.peerport),
		"--listen-client-urls",
		"https://localhost:" + strconv.Itoa(handler.clientport),
		"--advertise-client-urls",
		"https://localhost:" + strconv.Itoa(handler.clientport),
		"--trusted-ca-file",
		handler.cacert,
		"--cert-file",
		handler.servercert,
		"--key-file",
		handler.serverkey,
		"--peer-trusted-ca-file",
		handler.cacert,
		"--peer-cert-file",
		handler.servercert,
		"--peer-key-file",
		handler.serverkey,
		"--client-cert-auth",
		"--peer-client-cert-auth",
	}, handler.BaseServiceHandler.HandleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

// Stop the child process
func (handler *EtcdHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

// Handle result of a health probe
func (handler *EtcdHandler) healthCheckFun(responseBin *io.ReadCloser) error {
	type EtcdStatus struct {
		Health string `json:"health"`
	}
	response := EtcdStatus{}
	err := json.NewDecoder(*responseBin).Decode(&response)
	if err != nil {
		return errors.Wrap(err, "JSON decode of response failed")
	}
	if response.Health != "true" {
		return errors.Wrap(err, "etcd is unhealthy!")
	}
	return nil
}

// EtcdHandlerConstructor is supposed to be only used for testing
func EtcdHandlerConstructor(execEnv handlers.ExecutionEnvironment, creds *pki.MicrokubeCredentials) ([]handlers.ServiceHandler, error) {
	handler := NewEtcdHandler(execEnv, creds)
	handler.BaseServiceHandler = *handlers.NewHandler(handler.exit, handler.healthCheckFun,
		"https://localhost:"+strconv.Itoa(handler.clientport)+"/health", handler.stop, handler.Start, creds.EtcdCA, creds.EtcdClient)
	return []handlers.ServiceHandler{
		handler,
	}, nil
}
