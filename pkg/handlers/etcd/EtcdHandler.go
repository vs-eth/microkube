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

// Take care of running a single etcd listening on (hardcoded) localhost.
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
	out handlers.OutputHander
}

// NewEtcdHandler creates an EtcdHandler from the arguments provided
func NewEtcdHandler(datadir, binary string, server, client, ca *pki.RSACertificate, out handlers.OutputHander, exit handlers.ExitHandler) *EtcdHandler {
	obj := &EtcdHandler{
		datadir:    datadir,
		binary:     binary,
		clientport: 2379,
		peerport:   2380,
		servercert: server.CertPath,
		serverkey:  server.KeyPath,
		cacert:     ca.CertPath,
		cmd:        nil,
		out:        out,
	}
	obj.BaseServiceHandler = *handlers.NewHandler(exit, obj.healthCheckFun,
		"https://localhost:2379/health", obj.stop, obj.Start, ca, client)
	return obj
}

// See interface docs
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

// This function is supposed to be only used for testing
func EtcdHandlerConstructor(ca, server, client *pki.RSACertificate, binary, workdir string, outputHandler handlers.OutputHander, exitHandler handlers.ExitHandler) ([]handlers.ServiceHandler, error) {
	return []handlers.ServiceHandler{
		NewEtcdHandler(workdir, binary, server, client, ca, outputHandler, exitHandler),
	}, nil
}
