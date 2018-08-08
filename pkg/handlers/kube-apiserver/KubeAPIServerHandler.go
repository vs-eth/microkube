package kube_apiserver

import (
	"github.com/uubk/microkube/pkg/handlers"
	"os/exec"
	"github.com/uubk/microkube/pkg/pki"
	"io/ioutil"
	"github.com/pkg/errors"
	"crypto/tls"
	"crypto/x509"
	"net/http"
		"time"
	"net/url"
	"net"
	"encoding/json"
)

type KubeAPIServerHandler struct {
	binary         string
	kubeServerCert string
	kubeServerKey  string
	kubeClientCert string
	kubeClientKey  string
	kubeCACert     string
	etcdCACert     string
	etcdClientCert string
	etcdClientKey  string
	retriesLeft    int
	cmd            *handlers.CmdHandler
	out            handlers.OutputHander
	exit           handlers.ExitHandler
	healthCheckRunning bool
	healthCheck chan bool
}

func NewHandler(binary, kubeServerCert, kubeServerKey, kubeClientCert, kubeClientKey, kubeCACert, etcdClientCert,
	etcdClientKey, etcdCACert string, out handlers.OutputHander, exit handlers.ExitHandler) *KubeAPIServerHandler {
	return &KubeAPIServerHandler{
		binary:         binary,
		kubeServerCert: kubeServerCert,
		kubeServerKey:  kubeServerKey,
		kubeClientCert: kubeClientCert,
		kubeClientKey:  kubeClientKey,
		kubeCACert:     kubeCACert,
		etcdClientCert: etcdClientCert,
		etcdClientKey:  etcdClientKey,
		etcdCACert:     etcdCACert,
		retriesLeft:    1,
		cmd:            nil,
		out:            out,
		exit:           exit,
		healthCheckRunning: false,
		healthCheck: make(chan bool, 2),
	}
}

func (handler *KubeAPIServerHandler) handleExit(success bool, exitError *exec.ExitError) {
	handler.retriesLeft--
	if handler.retriesLeft > 0 {
		handler.Start()
	} else {
		handler.exit(success, exitError)
	}
}

func (handler *KubeAPIServerHandler) Start() error {
	handler.cmd = handlers.NewHandler(handler.binary, []string{
		"--bind-address",
		"0.0.0.0",
		"--insecure-port",
		"0",
		"--secure-port",
		"7443",
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
	}, handler.handleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

func (handler *KubeAPIServerHandler) Stop() {

}

func (handler *KubeAPIServerHandler) EnableHealthChecks(ca, client *pki.RSACertificate, messages chan handlers.HealthMessage, forever bool) {
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

func (handler *KubeAPIServerHandler) healthCheckFun(ca, client *pki.RSACertificate) error {
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
	responseHTTP, err := httpClient.Get("https://localhost:7443/healthz")
	waitTime := 100 * time.Millisecond
	for err != nil {
		if uerr, ok := err.(*url.Error); ok {
			if operr, ok := uerr.Err.(*net.OpError); ok {
				if operr.Op == "dial" {
					if waitTime > time.Second*4 {
						return errors.New("Timeout waiting for kube-apiserver to come up")
					}
					time.Sleep(waitTime)
					responseHTTP, err = httpClient.Get("https://localhost:7443/healtz")
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