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
	handlers.BaseServiceHandler
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

func (handler *EtcdHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

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
