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
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/pki"
	"os"
	"os/exec"
	"testing"
)

// dummyServiceHandler implements the ServiceHandler interface and does nothing
type dummyServiceHandler struct {
	isStarted      bool
	errorCallCount int
}

// Start starts this service.
func (d *dummyServiceHandler) Start() error {
	d.isStarted = true
	d.errorCallCount--
	if d.errorCallCount == 0 {
		return errors.New("Test error")
	}
	return nil
}

// EnableHealthChecks enable health checks, either for one check (forever == false) or until the process is stopped.
// Each health probe will write it's result to the channel provided
func (d *dummyServiceHandler) EnableHealthChecks(messages chan handlers.HealthMessage, forever bool) {
	d.errorCallCount--
	healthy := !(d.errorCallCount == 0)

	msg := handlers.HealthMessage{
		IsHealthy: healthy,
	}
	if !healthy {
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
func testUUTConstructorConstructor(t *testing.T, errorCallCount int) func(execEnv handlers.ExecutionEnvironment, creds *pki.MicrokubeCredentials) ([]handlers.ServiceHandler, error) {
	return func(execEnv handlers.ExecutionEnvironment, creds *pki.MicrokubeCredentials) ([]handlers.ServiceHandler, error) {
		// Certificate generation checked elsewhere

		status, err := os.Stat(execEnv.Workdir)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
		if !status.IsDir() {
			t.Fatalf("Expected '%s' to be a directory!", execEnv.Workdir)
		}

		errorCallCount--
		if errorCallCount == 0 {
			return nil, errors.New("Test error")
		}

		execEnv.OutputHandler([]byte("Foobar"))

		return []handlers.ServiceHandler{
			&dummyServiceHandler{errorCallCount: errorCallCount},
		}, nil
	}
}

// TestStartHandlerForTest uses StartHandlerForTest to start a dummy handler
func TestStartHandlerForTest(t *testing.T) {
	handler := testUUTConstructorConstructor(t, 0)
	handlerList, _, _, err := StartHandlerForTest(123, "testhandler", "/bin/bash", handler, func(success bool,
		exitError *exec.ExitError) {

	}, true, 1, nil, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if len(handlerList) != 1 {
		t.Fatalf("Expected handler list to only contain _one_ element")
	}
}

// TestStartHandlerForTestErrors injects faults into StartHandlerForTest to test different error cases
func TestStartHandlerForTestErrors(t *testing.T) {
	// Inject fault into start
	handler := testUUTConstructorConstructor(t, 2)
	_, _, _, err := StartHandlerForTest(123, "testhandler", "/bin/bash", handler, func(success bool,
		exitError *exec.ExitError) {

	}, false, 1, nil, nil)
	if err == nil {
		t.Fatal("Expected error missing!")
	}
	if err.Error() != "testhandler startup failed: 'Test error'" {
		t.Fatalf("Unexpected error: %s", err)
	}

	// Inject fault into constructor
	handler = testUUTConstructorConstructor(t, 1)
	_, _, _, err = StartHandlerForTest(123, "testhandler", "/bin/bash", handler, func(success bool,
		exitError *exec.ExitError) {

	}, false, 1, nil, nil)
	if err == nil {
		t.Fatal("Expected error missing!")
	}
	if err.Error() != "testhandler handler creation failed: 'Test error'" {
		t.Fatalf("Unexpected error: %s", err)
	}

	// Inject fault into health check
	handler = testUUTConstructorConstructor(t, 3)
	_, _, _, err = StartHandlerForTest(123, "testhandler", "/bin/bash", handler, func(success bool,
		exitError *exec.ExitError) {

	}, false, 1, nil, nil)
	if err == nil {
		t.Fatal("Expected error missing!")
	}
	if err.Error() != "testhandler unhealthy: Invalid state, not started" {
		t.Fatalf("Unexpected error: %s", err)
	}

	// Inject fault into binary check
	handler = testUUTConstructorConstructor(t, 0)
	_, _, _, err = StartHandlerForTest(123, "testhandler", "/bin/bashbashbashbashbashABC", handler, func(success bool,
		exitError *exec.ExitError) {

	}, false, 1, nil, nil)
	if err == nil {
		t.Fatal("Expected error missing!")
	}
	if err.Error() != "error while searching for testhandler binary: 'Couldn't find file'" {
		t.Fatalf("Unexpected error: %s", err)
	}
}
