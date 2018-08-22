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

import (
	"net"
	"os/exec"
)

// ExitHandler describes a function that is called when a process exits.
type ExitHandler func(success bool, exitError *exec.ExitError)

// OutputHandler describes a function that is called whenever a process outputs something
type OutputHandler func(output []byte)

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

// ExecutionEnvironment describes the environment to execute something in
type ExecutionEnvironment struct {
	// Binary contains the full path to the program to run
	Binary string
	// SudoMethod contains the binary to execute when running programs as root (sudo, pkexec, ...)
	SudoMethod string
	// Workdir contains a path where an application may store it's data
	Workdir string
	// ListenAddress is the address to bind exposed services to
	ListenAddress net.IP
	// ServiceAddress is the first address in the k8s service network
	ServiceAddress net.IP
	// OutputHandler to pass command output to
	OutputHandler OutputHandler
	// ExitHandler to notify on command exit
	ExitHandler ExitHandler

	EtcdClientPort            int
	EtcdPeerPort              int
	KubeApiPort               int
	KubeNodeApiPort           int
	KubeControllerManagerPort int
	KubeletHealthPort         int
	KubeProxyHealthPort       int
	KubeProxyMetricsPort      int
	KubeSchedulerHealthPort   int
	KubeSchedulerMetricsPort  int
}

func (e *ExecutionEnvironment) InitPorts(base int) {
	e.EtcdClientPort = base
	e.EtcdPeerPort = base + 1
	e.KubeApiPort = base + 2
	e.KubeNodeApiPort = base + 3
	e.KubeControllerManagerPort = base + 4
	e.KubeletHealthPort = base + 5
	e.KubeProxyHealthPort = base + 6
	e.KubeProxyMetricsPort = base + 7
	e.KubeSchedulerHealthPort = base + 8
	e.KubeSchedulerMetricsPort = base + 9
}

func (e *ExecutionEnvironment) CopyPorts(o *ExecutionEnvironment) {
	e.EtcdClientPort = o.EtcdClientPort
	e.EtcdPeerPort = o.EtcdPeerPort
	e.KubeApiPort = o.KubeApiPort
	e.KubeNodeApiPort = o.KubeNodeApiPort
	e.KubeControllerManagerPort = o.KubeControllerManagerPort
	e.KubeletHealthPort = o.KubeletHealthPort
	e.KubeProxyHealthPort = o.KubeProxyHealthPort
	e.KubeProxyMetricsPort = o.KubeProxyMetricsPort
	e.KubeSchedulerHealthPort = o.KubeSchedulerHealthPort
	e.KubeSchedulerMetricsPort = o.KubeSchedulerMetricsPort
}
