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

package cmd

import (
	"flag"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/uubk/microkube/pkg/handlers"
	"net"
	"os"
)

// ArgHandler provides applications with a unified set of command line parameters
type ArgHandler struct {
	/* Extracted data */
	// Directory to create all state directories in
	BaseDir string
	// Directory to additionally include in the binary search path
	ExtraBinDir string
	// Network range to use for pods
	PodRangeNet *net.IPNet
	// Network range to use for services
	ServiceRangeNet *net.IPNet
	// Network range that contains both pod and service range
	ClusterIPRange *net.IPNet

	/* Arguments */
	verbose      *bool
	root         *string
	extraBinDir  *string
	podRange     *string
	serviceRange *string
	sudoMethod   *string
}

// HandleArgs registers, parses and evaluates command line arguments
func (a *ArgHandler) HandleArgs() *handlers.ExecutionEnvironment {
	a.setupArgs()
	flag.Parse()
	return a.evalArgs()
}

// setupArg registers command line arguments
func (a *ArgHandler) setupArgs() {
	a.verbose = flag.Bool("verbose", true, "Enable verbose output")
	a.root = flag.String("root", "~/.mukube", "Microkube root directory")
	a.extraBinDir = flag.String("extra-bin-dir", "", "Additional directory to search for executables")
	a.podRange = flag.String("pod-range", "10.233.42.1/24", "Pod IP range to use")
	a.serviceRange = flag.String("service-range", "10.233.43.1/24", "Service IP range to use")
	a.sudoMethod = flag.String("sudo", "/usr/bin/pkexec", "Sudo tool to use")
}

// evalArgs parses the command line arguments
func (a *ArgHandler) evalArgs() *handlers.ExecutionEnvironment {
	if *a.verbose {
		log.SetLevel(log.DebugLevel)
	}
	var err error
	a.BaseDir, err = homedir.Expand(*a.root)
	if err != nil {
		log.WithError(err).WithField("root", *a.root).Fatal("Couldn't expand root directory")
	}
	a.ExtraBinDir, err = homedir.Expand(*a.extraBinDir)
	if err != nil {
		log.WithError(err).WithField("extraBinDir", *a.extraBinDir).Fatal("Couldn't expand extraBin directory")
	}

	var serviceRangeIP net.IP
	var bindAddr net.IP
	a.PodRangeNet, a.ServiceRangeNet, a.ClusterIPRange, bindAddr, serviceRangeIP, err = CalculateIPRanges(*a.podRange, *a.serviceRange)
	if err != nil {
		log.Fatal("IP calculation returned error, aborting now!")
	}
	dnsIP := net.IPv4(0, 0, 0, 0)
	copy(dnsIP, serviceRangeIP)
	dnsIP[15]++

	file, err := os.Stat(*a.sudoMethod)
	if err != nil || !file.Mode().IsRegular() {
		log.WithError(err).WithField("sudo", *a.sudoMethod).Fatal("Sudo method is not a regular file!")
	}

	baseExecEnv := handlers.ExecutionEnvironment{}
	baseExecEnv.ListenAddress = bindAddr
	baseExecEnv.ServiceAddress = serviceRangeIP
	baseExecEnv.DNSAddress = dnsIP
	baseExecEnv.SudoMethod = *a.sudoMethod
	baseExecEnv.InitPorts(7000)
	return &baseExecEnv
}
