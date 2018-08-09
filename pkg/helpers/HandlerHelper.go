package helpers

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/pki"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"time"
)

type ExitHandler func(success bool, exitError *exec.ExitError)
type OutputHander func(output []byte)
type StopHandler func()
type StartHandler func() error
type HealthCheckValidatorFunction func(result *io.ReadCloser) error

type HealthMessage struct {
	IsHealthy bool
	Error     error
}

type HandlerHelper struct {
	healthCheckRunning   bool
	healthCheck          chan bool
	retriesLeft          int
	exit                 ExitHandler
	healthCheckValidator HealthCheckValidatorFunction
	healthCheckEndpoint  string
	stopHandler          StopHandler
	startHandler         StartHandler
}

func NewHandlerHelper(exit ExitHandler, healthCheckValidator HealthCheckValidatorFunction, healthCheckEndpoint string,
	stopHandler StopHandler, startHandler StartHandler) *HandlerHelper {
	return &HandlerHelper{
		healthCheckRunning:   false,
		healthCheck:          make(chan bool, 2),
		retriesLeft:          1,
		exit:                 exit,
		healthCheckValidator: healthCheckValidator,
		stopHandler:          stopHandler,
		startHandler:         startHandler,
		healthCheckEndpoint:  healthCheckEndpoint,
	}
}

func (handler *HandlerHelper) healthCheckFun(ca, client *pki.RSACertificate) error {
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
	responseHTTP, err := httpClient.Get(handler.healthCheckEndpoint)
	waitTime := 100 * time.Millisecond
	for err != nil {
		if uerr, ok := err.(*url.Error); ok {
			if operr, ok := uerr.Err.(*net.OpError); ok {
				if operr.Op == "dial" {
					if waitTime > time.Second*4 {
						return errors.New("Timeout waiting for service to come up")
					}
					time.Sleep(waitTime)
					responseHTTP, err = httpClient.Get(handler.healthCheckEndpoint)
					waitTime = 2 * waitTime
					continue
				}
			}
		}
		return errors.Wrap(err, "Health check failed")
	}
	responseBin := responseHTTP.Body
	defer responseBin.Close()

	return handler.healthCheckValidator(&responseBin)
}

func (handler *HandlerHelper) Stop() {
	handler.stopHandler()
	if handler.healthCheckRunning {
		// Notify goroutine of exit
		handler.healthCheck <- true
	}
}

func (handler *HandlerHelper) EnableHealthChecks(ca, client *pki.RSACertificate, messages chan HealthMessage,
	forever bool) {
	if !handler.healthCheckRunning {
		handler.healthCheckRunning = true
		go func() {
			for {
				val := handler.healthCheckFun(ca, client)
				messages <- HealthMessage{
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

func (handler *HandlerHelper) HandleExit(success bool, exitError *exec.ExitError) {
	handler.retriesLeft--
	if handler.retriesLeft > 0 {
		handler.startHandler()
	} else {
		handler.exit(success, exitError)
	}
}
