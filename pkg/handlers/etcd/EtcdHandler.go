package etcd

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/pki"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"time"
)

type EtcdHandler struct {
	datadir            string
	binary             string
	clientport         int
	peerport           int
	servercert         string
	serverkey          string
	cacert             string
	cmd                *handlers.CmdHandler
	out                handlers.OutputHander
	retriesLeft        int
	exit               handlers.ExitHandler
	healthCheck        chan bool
	healthCheckRunning bool
}

func NewHandler(datadir, binary, servercert, serverkey, cacert string, out handlers.OutputHander, retries int, exit handlers.ExitHandler) *EtcdHandler {
	return &EtcdHandler{
		datadir:            datadir,
		binary:             binary,
		clientport:         2379,
		peerport:           2380,
		servercert:         servercert,
		serverkey:          serverkey,
		cacert:             cacert,
		cmd:                nil,
		out:                out,
		retriesLeft:        retries,
		exit:               exit,
		healthCheck:        make(chan bool, 1),
		healthCheckRunning: false,
	}
}

func (handler *EtcdHandler) handleExit(success bool, exitError *exec.ExitError) {
	handler.retriesLeft--
	if handler.retriesLeft > 0 {
		handler.Start()
	} else {
		handler.exit(success, exitError)
	}
}

func (handler *EtcdHandler) Start() error {
	handler.cmd = handlers.NewHandler(handler.binary, []string{
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
	}, handler.handleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

func (handler *EtcdHandler) Stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
	if handler.healthCheckRunning {
		// Notify goroutine of exit
		handler.healthCheck <- true
	}
}

func (handler *EtcdHandler) EnableHealthChecks(ca, client *pki.RSACertificate, messages chan handlers.HealthMessage, forever bool) {
	if !handler.healthCheckRunning {
		handler.healthCheckRunning = true
		go func() {
			for {
				val := handler.healthCheckFun(ca, client)
				messages <- handlers.HealthMessage{
					IsHealthy: val == nil,
					Error:     val,
				}
				if !forever {
					handler.healthCheckRunning = false
					break
				}
				select {
				case <-handler.healthCheck:
					return
				case <-time.After(10 * time.Second):
					continue
				}
			}
		}()
	}
}

func (handler *EtcdHandler) healthCheckFun(ca, client *pki.RSACertificate) error {
	caCert, err := ioutil.ReadFile(ca.CertPath)
	if err != nil {
		return errors.Wrap(err, "CA load from file failed")
	}
	clientCert, err := tls.LoadX509KeyPair(client.CertPath, client.KeyPath)
	if err != nil {
		return errors.Wrap(err, "client cert load from file failed")
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return errors.Wrap(err, "CA append to pool failed")
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{clientCert},
				RootCAs:      caPool,
			},
		},
	}
	responseHTTP, err := httpClient.Get("https://localhost:" + strconv.Itoa(handler.clientport) + "/health")
	waitTime := 100 * time.Millisecond
	for err != nil {
		if uerr, ok := err.(*url.Error); ok {
			if operr, ok := uerr.Err.(*net.OpError); ok {
				if operr.Op == "dial" {
					if waitTime > time.Second*4 {
						return errors.New("Timeout waiting for etcd to come up")
					}
					time.Sleep(waitTime)
					responseHTTP, err = httpClient.Get("https://localhost:" + strconv.Itoa(handler.clientport) + "/health")
					waitTime = 2 * waitTime
					continue
				}
			}
		}
		return errors.Wrap(err, "Health check failed")
	}
	responseBin := responseHTTP.Body
	defer responseBin.Close()

	type EtcdStatus struct {
		Health string `json:"health"`
	}
	response := EtcdStatus{}
	err = json.NewDecoder(responseBin).Decode(&response)
	if err != nil {
		return errors.Wrap(err, "JSON decode of response failed")
	}
	if response.Health != "true" {
		return errors.Wrap(err, "etcd is unhealthy!")
	}
	return nil
}
