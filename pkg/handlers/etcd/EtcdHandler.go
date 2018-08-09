package etcd

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/helpers"
	"io"
	"strconv"
)

type EtcdHandler struct {
	helpers.HandlerHelper
	datadir    string
	binary     string
	clientport int
	peerport   int
	servercert string
	serverkey  string
	cacert     string
	cmd        *helpers.CmdHandler
	out        helpers.OutputHander
}

func NewEtcdHandler(datadir, binary, servercert, serverkey, cacert string, out helpers.OutputHander, retries int, exit helpers.ExitHandler) *EtcdHandler {
	obj := &EtcdHandler{
		datadir:    datadir,
		binary:     binary,
		clientport: 2379,
		peerport:   2380,
		servercert: servercert,
		serverkey:  serverkey,
		cacert:     cacert,
		cmd:        nil,
		out:        out,
	}
	obj.HandlerHelper = *helpers.NewHandlerHelper(exit, obj.healthCheckFun, "https://localhost:2379/health", obj.stop, obj.Start)
	return obj
}

func (handler *EtcdHandler) Start() error {
	handler.cmd = helpers.NewCmdHandler(handler.binary, []string{
		"--data-dir",
		handler.datadir,
		"--listen-peer-urls",
		"http://localhost:" + strconv.Itoa(handler.peerport),
		"--initial-advertise-peer-urls",
		"http://localhost:" + strconv.Itoa(handler.peerport),
		"--initial-cluster",
		"default=http://localhost:" + strconv.Itoa(handler.peerport),
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
	}, handler.HandlerHelper.HandleExit, handler.out, handler.out)
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
