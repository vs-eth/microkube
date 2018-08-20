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
	"testing"
	"github.com/uubk/microkube/pkg/pki"
	"github.com/uubk/microkube/pkg/handlers"
	"os"
	"os/exec"
	"github.com/pkg/errors"
)

// dummyServiceHandler implements the ServiceHandler interface and does nothing
type dummyServiceHandler struct {
	isStarted bool
}

// Start starts this service.
func (d *dummyServiceHandler) Start() error {
	d.isStarted = true
	return nil
}

// EnableHealthChecks enable health checks, either for one check (forever == false) or until the process is stopped.
// Each health probe will write it's result to the channel provided
func (d *dummyServiceHandler) EnableHealthChecks(messages chan handlers.HealthMessage, forever bool) {
	msg := handlers.HealthMessage{
		IsHealthy: d.isStarted,
	}
	if !d.isStarted {
		msg.Error = errors.New("Invalid state, not started")
	}
	messages <- msg

	if forever {
		panic("Not implemented")
	}
}

// Stop stops this service and all associated goroutines (e.g. health checks). If it as already stopped,
// this method does nothing.
func (d *dummyServiceHandler) Stop() {
	d.isStarted = false
}

// testUUTConstructorConstructor returns an UUTConstructor for some test 't'
func testUUTConstructorConstructor(t *testing.T) (func(ca, server, client *pki.RSACertificate, binary, workdir string, outputHandler handlers.OutputHander, exitHandler handlers.ExitHandler) ([]handlers.ServiceHandler, error)) {
	return func(ca, server, client *pki.RSACertificate, binary, workdir string, outputHandler handlers.OutputHander, exitHandler handlers.ExitHandler) ([]handlers.ServiceHandler, error) {
		// Check if all files exist
		files := []string {
			ca.CertPath,
			ca.KeyPath,
			server.CertPath,
			server.KeyPath,
			client.CertPath,
			client.KeyPath,
			binary,
		}
		for _, file := range files {
			status, err := os.Stat(file)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			if !status.Mode().IsRegular() {
				t.Fatalf("Expected '%s' to be a regular file!", file)
			}
			if status.Size() < 512 {
				t.Fatalf("Expected '%s' to be at least 512 bytes!", file)
			}
		}

		status, err := os.Stat(workdir)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
		if !status.IsDir() {
			t.Fatalf("Expected '%s' to be a directory!", workdir)
		}

		return []handlers.ServiceHandler {
			&dummyServiceHandler{},
		}, nil
	}
}

// TestStartHandlerForTest uses StartHandlerForTest to start a dummy handler
func TestStartHandlerForTest(t *testing.T) {
	handler := testUUTConstructorConstructor(t)
	handlerList, _, _, _, err := StartHandlerForTest("testhandler", "/bin/bash", handler, func(success bool,
		exitError *exec.ExitError) {

	}, false, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if len(handlerList) != 1 {
		t.Fatalf("Expected handler list to only contain _one_ element")
	}
}