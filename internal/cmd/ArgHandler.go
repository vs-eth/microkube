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

// argHandlerGlobalState contains the values of all arguments, because flag.CommandLine is a) global and b) cannot be
// resetted. Running two unit tests which initialize two instances of ArgHandler would therefore crash. To work around
// this issue, we create each flag precisely once and point them to an instance of this struct so that we can reuse
// flags across instances of ArgHandler
type argHandlerGlobalState struct {
	verbose        bool
	root           string
	extraBinDir    string
	podRange       string
	serviceRange   string
	sudoMethod     string
	enableDns      bool
	enableKubeDash bool
}

// gs contains the instance of argHandlerGlobalState
var gs = argHandlerGlobalState{}

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
	// Whether to deploy the kubernetes dashboard cluster addon
	EnableKubeDash bool
	// Whether to deploy the CoreDNS cluster addon
	EnableDns bool

	// Whether we should set up all arguments (main binary) or only shared arguments (cluster parameters)
	isMainBinary bool
}

// NewArgHandler returns a new instance of ArgHandler, registering arguments if necessary
func NewArgHandler(isMainBinary bool) *ArgHandler {
	obj := ArgHandler{
		isMainBinary: isMainBinary,
	}
	obj.setupArgs()
	return &obj
}

// HandleArgs registers, parses and evaluates command line arguments
func (a *ArgHandler) HandleArgs() *handlers.ExecutionEnvironment {
	flag.Parse()
	return a.evalArgs()
}

// setupBoolArg creates a boolean argument if necessary. Subsequent calls will be ignored.
func (a *ArgHandler) setupBoolArg(name, description string, global *bool, defaultVal bool) {
	lk := flag.Lookup(name)
	if lk == nil {
		flag.BoolVar(global, name, defaultVal, description)
	}
}

// setupStringArg creates a string argument if necessary. Subsequent calls will be ignored.
func (a *ArgHandler) setupStringArg(name, description string, global *string, defaultVal string) {
	lk := flag.Lookup(name)
	if lk == nil {
		flag.StringVar(global, name, defaultVal, description)
	}
}

// setupArg registers command line arguments
func (a *ArgHandler) setupArgs() {
	a.setupBoolArg("verbose", "Enable verbose output", &gs.verbose, false)
	a.setupStringArg("pod-range", "Pod IP range to use", &gs.podRange, "10.233.42.1/24")
	a.setupStringArg("service-range", "Service IP range to use", &gs.serviceRange, "10.233.43.1/24")

	if a.isMainBinary {
		a.setupStringArg("root", "Microkube root directory", &gs.root, "~/.mukube")
		a.setupStringArg("extra-bin-dir", "Additional directory to search for executables", &gs.extraBinDir, "")
		a.setupStringArg("sudo", "Sudo tool to use", &gs.sudoMethod, "/usr/bin/pkexec")
		a.setupBoolArg("kube-dash", "Enable the kubernetes dashboard deployment", &gs.enableKubeDash, true)
		a.setupBoolArg("dns", "Enable the DNS deployment", &gs.enableDns, true)
	}
}

// evalArgs parses the command line arguments
func (a *ArgHandler) evalArgs() *handlers.ExecutionEnvironment {
	if gs.verbose {
		log.SetLevel(log.DebugLevel)
	}
	var err error
	a.BaseDir, err = homedir.Expand(gs.root)
	if err != nil {
		log.WithError(err).WithField("root", gs.root).Fatal("Couldn't expand root directory")
	}
	a.ExtraBinDir, err = homedir.Expand(gs.extraBinDir)
	if err != nil {
		log.WithError(err).WithField("extraBinDir", gs.extraBinDir).Fatal("Couldn't expand extraBin directory")
	}

	var serviceRangeIP net.IP
	var bindAddr net.IP
	a.PodRangeNet, a.ServiceRangeNet, a.ClusterIPRange, bindAddr, serviceRangeIP, err = CalculateIPRanges(gs.podRange, gs.serviceRange)
	if err != nil {
		log.Fatal("IP calculation returned error, aborting now!")
	}
	dnsIP := net.IPv4(0, 0, 0, 0)
	copy(dnsIP, serviceRangeIP)
	dnsIP[15]++

	file, err := os.Stat(gs.sudoMethod)
	if err != nil || !file.Mode().IsRegular() {
		log.WithError(err).WithField("sudo", gs.sudoMethod).Fatal("Sudo method is not a regular file!")
	}

	a.EnableKubeDash = gs.enableKubeDash
	a.EnableDns = gs.enableDns

	baseExecEnv := handlers.ExecutionEnvironment{}
	baseExecEnv.ListenAddress = bindAddr
	baseExecEnv.ServiceAddress = serviceRangeIP
	baseExecEnv.DNSAddress = dnsIP
	baseExecEnv.SudoMethod = gs.sudoMethod
	baseExecEnv.InitPorts(7000)
	return &baseExecEnv
}
