package handlers

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

type StopHandler func()
type StartHandler func() error
type HealthCheckValidatorFunction func(result *io.ReadCloser) error

// BaseServiceHandler serves as a base type for all handlers in github.com/uubk/microkube/pkg/handlers,
// bundling common functions
type BaseServiceHandler struct {
	// Is the health check goroutine running (that is, do we need to stop it)?
	healthCheckRunning bool
	// Channel to communicate with the health check routine. Write anything to this chanel to exit it
	healthCheck chan bool
	// Number of service restart retires left
	retriesLeft int
	// Exit handler, that is a function to be called after the final (retriesLeft == 0) exit
	exit ExitHandler
	// Health check 'validator' function. This should be implemented inside the other handlers. It gets called with the
	// HTTP result of a health check and needs to parse the actual status.
	healthCheckValidator HealthCheckValidatorFunction
	// URL to send health checks to
	healthCheckEndpoint string
	// Handler to be called if the user invokes Stop(). It is expected that other handlers use this pointer to provide
	// a function that stops the actual process
	stopHandler StopHandler
	// Handler to be called if this class wants to start the service. It is expected that other handlers point this to
	// their Start() method
	startHandler StartHandler
	// CA and client certificate for health checks. Can be nil to disable TLS
	ca, client *pki.RSACertificate
}

// Create a new helper handler. For detailed field descriptions, refer to the struct docs.
func NewHandler(exit ExitHandler, healthCheckValidator HealthCheckValidatorFunction, healthCheckEndpoint string,
	stopHandler StopHandler, startHandler StartHandler, ca, client *pki.RSACertificate) *BaseServiceHandler {
	return &BaseServiceHandler{
		healthCheckRunning:   false,
		healthCheck:          make(chan bool, 2),
		retriesLeft:          1,
		exit:                 exit,
		healthCheckValidator: healthCheckValidator,
		stopHandler:          stopHandler,
		startHandler:         startHandler,
		healthCheckEndpoint:  healthCheckEndpoint,
		ca:                   ca,
		client:               client,
	}
}

// Health check implementation. This function performs a single request against the configured health check endpoint,
// passing the results to the healthCheckValidator
func (handler *BaseServiceHandler) healthCheckFun() error {
	var httpClient *http.Client
	if handler.ca != nil {
		caCert, err := ioutil.ReadFile(handler.ca.CertPath)
		if err != nil {
			return errors.Wrap(err, "CA load from file failed")
		}
		clientCert, err := tls.LoadX509KeyPair(handler.client.CertPath, handler.client.KeyPath)
		if err != nil {
			return errors.Wrap(err, "client cert load from file failed")
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return errors.Wrap(err, "CA append to pool failed")
		}

		httpClient = &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
				TLSClientConfig: &tls.Config{
					Certificates: []tls.Certificate{clientCert},
					RootCAs:      caPool,
				},
			},
		}
	} else {
		httpClient = &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
				TLSClientConfig: &tls.Config{},
			},
		}
	}
	responseHTTP, err := httpClient.Get(handler.healthCheckEndpoint)
	// Backoff is doubled starting at .1 seconds until the limit of 7 seconds is exceeded
	waitTime := 100 * time.Millisecond
	for err != nil {
		if uerr, ok := err.(*url.Error); ok {
			if operr, ok := uerr.Err.(*net.OpError); ok {
				// Most services need a moment to open the port...
				if operr.Op == "dial" {
					if waitTime > time.Second*7 {
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

// Stop the service. See interface ServiceHandler.
func (handler *BaseServiceHandler) Stop() {
	handler.stopHandler()
	if handler.healthCheckRunning {
		// Notify goroutine of exit
		handler.healthCheck <- true
	}
}

// Enable health checks, see interface ServiceHandler.
func (handler *BaseServiceHandler) EnableHealthChecks(messages chan HealthMessage, forever bool) {
	if !handler.healthCheckRunning {
		handler.healthCheckRunning = true
		go func() {
			for {
				val := handler.healthCheckFun()
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

// Handle a process exit. Other handlers are expected to call this method on process exit
func (handler *BaseServiceHandler) HandleExit(success bool, exitError *exec.ExitError) {
	handler.retriesLeft--
	if handler.retriesLeft > 0 {
		handler.startHandler()
	} else {
		handler.exit(success, exitError)
	}
}
