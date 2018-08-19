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

package helpers

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/pki"
	"io/ioutil"
	"time"
)

// UUTConstrutor is implemented by all types that use this simplified mechanism to be tested and is used to create a
// test object with all related resources
type UUTConstrutor func(ca, server, client *pki.RSACertificate, binary, workdir string, outputHandler handlers.OutputHander, exitHandler handlers.ExitHandler) ([]handlers.ServiceHandler, error)

// StartHandlerForTest starts a given handler for a unit test
func StartHandlerForTest(name, binary string, constructor UUTConstrutor, exitHandler handlers.ExitHandler, print bool, healthCheckTries int) (handlerList []handlers.ServiceHandler, ca *pki.RSACertificate, client *pki.RSACertificate, server *pki.RSACertificate, err error) {
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

	wd, err := FindBinary(binary, "", "")
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "error while searching for "+name+" binary")
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
