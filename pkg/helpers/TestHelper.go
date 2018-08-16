package helpers

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/pki"
	"io/ioutil"
	"time"
)

type UUTConstrutor func(ca, server, client *pki.RSACertificate, binary, workdir string, outputHandler handlers.OutputHander, exitHandler handlers.ExitHandler) ([]handlers.ServiceHandler, error)

func StartHandlerForTest(name string, constructor UUTConstrutor, exitHandler handlers.ExitHandler, print bool, healthCheckTries int) (handlerList []handlers.ServiceHandler, ca *pki.RSACertificate, client *pki.RSACertificate, server *pki.RSACertificate, err error) {
	tmpdir, err := ioutil.TempDir("", "microkube-unittests-"+name)
	if err != nil {
		errors.Wrap(err, "tempdir creation failed")
	}
	ca, server, client, err = CertHelper(tmpdir, name+"-unittest")
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "error in PKI handler")
	}

	outputHandler := func(output []byte) {
		if print {
			fmt.Println(name+"   |", string(output))
		}
	}

	wd, err := FindBinary(name, "", "")
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "error while searching for etcd binary")
	}

	// UUT
	handlerList, err = constructor(ca, server, client, wd, tmpdir, outputHandler, exitHandler)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, name+" handler creation failed")
	}
	handler := handlerList[len(handlerList)-1]
	err = handler.Start()
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, name+" startup failed")
	}

	healthMessage := make(chan handlers.HealthMessage, 1)
	msg := handlers.HealthMessage{
		IsHealthy: false,
	}
	for i := 0; i < healthCheckTries && (!msg.IsHealthy); i++ {
		handler.EnableHealthChecks(healthMessage, false)
		msg = <-healthMessage
		time.Sleep(1 * time.Second)
	}
	if !msg.IsHealthy {
		return nil, nil, nil, nil, errors.Wrap(msg.Error, name+" unhealthy: ")
	}

	return handlerList, ca, client, server, nil
}
