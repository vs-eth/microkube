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

package handlers

import "os/exec"

// ExitHandler describes a function that is called when a process exits.
type ExitHandler func(success bool, exitError *exec.ExitError)

// OutputHander describes a function that is called whenever a process outputs something
type OutputHander func(output []byte)

// HealthMessage describes health check results from services
type HealthMessage struct {
	// Is this service healthy?
	IsHealthy bool
	// If the service isn't healthy, is there a specific reason as to why?
	Error error
}

// ServiceHandler handle some kind of running service. This interface is implemented by all service handlers below this
// package
type ServiceHandler interface {
	// Start starts this service. If no error is returned, you are responsible for stopping it
	Start() error
	// EnableHealthChecks enable health checks, either for one check (forever == false) or until the process is stopped.
	// Each health probe will write it's result to the channel provided
	EnableHealthChecks(messages chan HealthMessage, forever bool)
	// Stop stops this service and all associated goroutines (e.g. health checks). If it as already stopped,
	// this method does nothing.
	Stop()
}
