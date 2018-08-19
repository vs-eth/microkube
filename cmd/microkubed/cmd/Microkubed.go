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

// Package cmd contains the implementation of the microkubed command
package cmd

import (
	"flag"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/uubk/microkube/internal/cmd"
	log2 "github.com/uubk/microkube/internal/log"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/handlers/etcd"
	"github.com/uubk/microkube/pkg/handlers/kube"
	"github.com/uubk/microkube/pkg/helpers"
	"github.com/uubk/microkube/pkg/pki"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"time"
)

// serviceConstructor describes a function that can create a service, given the I/O handlers
type serviceConstructor func(handlers.OutputHander, handlers.ExitHandler) (handlers.ServiceHandler, error)

// serviceEntry describes all information about a running service needed to check it's health and stop it
type serviceEntry struct {
	exitChan   chan bool
	healthChan chan handlers.HealthMessage
	handler    handlers.ServiceHandler
	name       string
}

// Microkubed handles an invocation of the 'microkubed' command line tool
type Microkubed struct {
	// Are we in 'graceful termination mode', that is, everything has started successfully and in order to stop we
	// should first stop all pods?
	gracefulTerminationMode bool
	// List of service handlers. This is used to handle shutdown on fatal errors
	serviceHandlers []handlers.ServiceHandler
	// Base directory of microkubed, which is where all state will be stored
	baseDir string
	// Extra binary dir which is added to the binary search path
	extraBinDir string

	// CIDR of pod IPs (commandline argument)
	podRangeNet *net.IPNet
	// CIDR of service IPs (commandline argument)
	serviceRangeNet *net.IPNet
	// First IP in service network, reserved for Kubernetes API Server service
	serviceRangeIP net.IP
	// CIDR of pod and service network combined
	clusterIPRange net.IPNet
	// Some host-local address that the services will bind to
	bindAddr net.IP

	// CA certificate for etcd
	etcdCA *pki.RSACertificate
	// Client certificate for etcd
	etcdClient *pki.RSACertificate
	// Server certificate for etcd
	etcdServer *pki.RSACertificate
	// CA certificate for kubernetes
	kubeCA *pki.RSACertificate
	// Client certificate for kubernetes
	kubeClient *pki.RSACertificate
	// Server certificate for kubernetes
	kubeServer *pki.RSACertificate
	// CA certificate for kubernetes in-cluster CA
	kubeClusterCA *pki.RSACertificate
	// Signing certificate for kubernetes service account tokens
	kubeSvcSignCert *pki.RSACertificate

	// Path to etcd server binary
	etcdBin string
	// Path to hyperkube binary
	hyperkubeBin string

	// A list of running services
	serviceList []serviceEntry
}

// Register and evaluate command-line arguments
func (m *Microkubed) handleArgs() {
	verbose := flag.Bool("verbose", true, "Enable verbose output")
	root := flag.String("root", "~/.mukube", "Microkube root directory")
	extraBinDir := flag.String("extra-bin-dir", "", "Additional directory to search for executables")
	podRange := flag.String("pod-range", "10.233.42.1/24", "Pod IP range to use")
	serviceRange := flag.String("service-range", "10.233.43.1/24", "Service IP range to use")
	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}
	var err error
	m.baseDir, err = homedir.Expand(*root)
	if err != nil {
		log.WithError(err).WithField("root", *root).Fatal("Couldn't expand root directory")
	}
	m.extraBinDir, err = homedir.Expand(*extraBinDir)
	if err != nil {
		log.WithError(err).WithField("extraBinDir", *extraBinDir).Fatal("Couldn't expand extraBin directory")
	}

	m.calculateIPRanges(*podRange, *serviceRange)
}

// Calculate all IP ranges from the command line strings
func (m *Microkubed) calculateIPRanges(podRange, serviceRange string) {
	var podRangeIP, serviceRangeIP net.IP
	var err error

	// Parse commandline arguments
	podRangeIP, m.podRangeNet, err = net.ParseCIDR(podRange)
	if err != nil {
		log.WithFields(log.Fields{
			"range": podRange,
		}).WithError(err).Fatal("Couldn't parse pod CIDR range")
	}
	m.serviceRangeIP, m.serviceRangeNet, err = net.ParseCIDR(serviceRange)
	if err != nil {
		log.WithFields(log.Fields{
			"range": podRange,
		}).WithError(err).Fatal("Couldn't parse service CIDR range")
	}

	// Find address to bind to
	m.bindAddr = cmd.FindBindAddress()

	// To combine pod and service range to form the cluster range, find first diverging bit
	baseOffset := 0
	serviceBelowPod := false
	for idx, octet := range m.serviceRangeNet.IP {
		if m.podRangeNet.IP[idx] != octet {
			// This octet diverges -> find bit
			baseOffset = idx * 8
			for mask := byte(0x80); mask > 0; mask /= 2 {
				baseOffset++
				if (m.podRangeNet.IP[idx] & mask) != (octet & mask) {
					// Found it
					serviceBelowPod = octet < m.podRangeNet.IP[idx]
					break
				}
			}
			baseOffset--
		}
	}
	m.clusterIPRange = net.IPNet{
		IP: podRangeIP,
	}
	if serviceBelowPod {
		m.clusterIPRange.IP = serviceRangeIP
	}
	m.clusterIPRange.Mask = net.CIDRMask(baseOffset, 32)
	log.WithFields(log.Fields{
		"podRange":     m.podRangeNet.String(),
		"serviceRange": m.serviceRangeNet.String(),
		"clusterRange": m.clusterIPRange.String(),
		"hostIP":       m.bindAddr,
	}).Info("IP ranges calculated")
}

// Create directories and copy CNI plugins if appropriate
func (m *Microkubed) createDirectories() {
	cmd.EnsureDir(m.baseDir, "", 0770)
	cmd.EnsureDir(m.baseDir, "kube", 0770)
	cmd.EnsureDir(m.baseDir, "etcdtls", 0770)
	cmd.EnsureDir(m.baseDir, "kubesched", 0770)
	cmd.EnsureDir(m.baseDir, "kubetls", 0770)
	cmd.EnsureDir(m.baseDir, "kubectls", 0770)
	cmd.EnsureDir(m.baseDir, "kubestls", 0770)
	cmd.EnsureDir(m.baseDir, "etcddata", 0770)

	// Special case: in case the extra binaries directory contains CNI plugins, copy them to the right location
	cmd.EnsureDir(m.baseDir, path.Join("kube", "kubelet"), 0755)
	cmd.EnsureDir(m.baseDir, path.Join("kube", "kubelet", "cni"), 0755)
	cniPlugins := []string{
		"bridge",
		"host-local",
		"loopback",
	}
	for _, plugin := range cniPlugins {
		_, err := os.Stat(path.Join(m.extraBinDir, plugin))
		if err == nil {
			_, err := os.Stat(path.Join(m.baseDir, "kube", "kubelet", "cni", plugin))
			if err != nil {
				err = os.Link(path.Join(m.extraBinDir, plugin), path.Join(m.baseDir, "kube", "kubelet", "cni", plugin))
				if err != nil {
					log.WithFields(log.Fields{
						"src":       path.Join(m.extraBinDir, plugin),
						"dest":      path.Join(m.baseDir, "kube", "kubelet", "cni", plugin),
						"app":       "microkube",
						"component": "prep",
					}).WithError(err).Fatal("Couldn't link CNI plugin")
				}
			}
		}
	}
}

// Create certificates if they don't already exist
func (m *Microkubed) createCertificates() {
	var err error
	m.etcdCA, m.etcdServer, m.etcdClient, err = cmd.EnsureFullPKI(path.Join(m.baseDir, "etcdtls"), "Microkube ETCD",
		false, true, []string{m.bindAddr.String()})
	if err != nil {
		log.WithError(err).Fatal("Couldn't create etcd PKI")
	}
	m.kubeCA, m.kubeServer, m.kubeClient, err = cmd.EnsureFullPKI(path.Join(m.baseDir, "kubetls"), "Microkube Kubernetes",
		true, false, []string{m.bindAddr.String(), m.serviceRangeIP.String()})
	if err != nil {
		log.WithError(err).Fatal("Couldn't create kubernetes PKI")
	}
	m.kubeClusterCA, err = cmd.EnsureCA(path.Join(m.baseDir, "kubectls"), "Microkube Cluster CA")
	if err != nil {
		log.WithError(err).Fatal("Couldn't create kubernetes cluster CA")
	}
	m.kubeSvcSignCert, err = cmd.EnsureSigningCert(path.Join(m.baseDir, "kubestls"), "Microkube Cluster SVC Signcert")
	if err != nil {
		log.WithError(err).Fatal("Couldn't create kubernetes service secret signing certificate")
	}
}

// Find binaries
func (m *Microkubed) findBinaries() {
	var err error
	m.etcdBin, err = helpers.FindBinary("etcd", m.baseDir, m.extraBinDir)
	if err != nil {
		log.WithError(err).Fatal("Couldn't find etcd binary")
	}
	m.hyperkubeBin, err = helpers.FindBinary("hyperkube", m.baseDir, m.extraBinDir)
	if err != nil {
		log.WithError(err).Fatal("Couldn't find hyperkube binary")
	}
}

// Start etcd
func (m *Microkubed) startEtcd() {
	etcdHandler, etcdChan, etcdHealthChan := m.startService("etcd", func(etcdOutputHandler handlers.OutputHander,
		etcdExitHandler handlers.ExitHandler) (handlers.ServiceHandler, error) {
		return etcd.NewEtcdHandler(path.Join(m.baseDir, "etcddata"), m.etcdBin, m.etcdServer, m.etcdClient, m.etcdCA,
			etcdOutputHandler, etcdExitHandler), nil
	}, log2.NewETCDLogParser())
	m.serviceHandlers = append(m.serviceHandlers, etcdHandler)
	log.Debug("ETCD ready")

	m.serviceList = append(m.serviceList, serviceEntry{
		handler:    etcdHandler,
		exitChan:   etcdChan,
		healthChan: etcdHealthChan,
		name:       "etcd",
	})
}

// Start Kube APIServer
func (m *Microkubed) startKubeAPIServer() {
	log.Debug("Starting kube api server...")
	kubeAPIHandler, kubeAPIChan, kubeAPIHealthChan := m.startService("kube-apiserver",
		func(kubeAPIOutputHandler handlers.OutputHander,
			kubeAPIExitHandler handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return kube.NewKubeAPIServerHandler(m.hyperkubeBin, m.kubeServer, m.kubeClient, m.kubeCA, m.kubeSvcSignCert, m.etcdClient,
				m.etcdCA, kubeAPIOutputHandler, kubeAPIExitHandler, m.bindAddr.String(), m.serviceRangeNet.String()), nil
		}, log2.NewKubeLogParser("kube-api"))
	m.serviceHandlers = append(m.serviceHandlers, kubeAPIHandler)
	log.Debug("Kube api server ready")

	// Generate kubeconfig for kubelet and kubectl
	log.Debug("Generating kubeconfig...")
	kubeconfig := path.Join(m.baseDir, "kube/", "kubeconfig")
	_, err := os.Stat(kubeconfig)
	if err != nil {
		log.Debug("Creating kubeconfig")
		err = kube.CreateClientKubeconfig(m.kubeCA, m.kubeClient, kubeconfig, m.bindAddr.String())
		if err != nil {
			log.WithError(err).Fatal("Couldn't create kubeconfig!")
			return
		}
	}

	m.serviceList = append(m.serviceList, serviceEntry{
		handler:    kubeAPIHandler,
		exitChan:   kubeAPIChan,
		healthChan: kubeAPIHealthChan,
		name:       "kube-api",
	})
}

// Start controller-manager
func (m *Microkubed) startKubeControllerManager() {
	log.Debug("Starting controller-manager...")
	kubeCtrlMgrHandler, kubeCtrlMgrChan, kubeCtrlMgrHealthChan := m.startService("kube-controller-manager",
		func(kubeCtrlMgrOutputHandler handlers.OutputHander,
			kubeCtrlMgrExitHandler handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return kube.NewControllerManagerHandler(m.hyperkubeBin, path.Join(m.baseDir, "kube/", "kubeconfig"),
				m.bindAddr.String(), m.kubeServer, m.kubeClient, m.kubeCA, m.kubeClusterCA, m.kubeSvcSignCert, m.podRangeNet.String(),
				kubeCtrlMgrOutputHandler, kubeCtrlMgrExitHandler), nil
		}, log2.NewKubeLogParser("kube-controller-manager"))
	m.serviceHandlers = append(m.serviceHandlers, kubeCtrlMgrHandler)
	log.Debug("Kube controller-manager ready")

	m.serviceList = append(m.serviceList, serviceEntry{
		handler:    kubeCtrlMgrHandler,
		exitChan:   kubeCtrlMgrChan,
		healthChan: kubeCtrlMgrHealthChan,
		name:       "kube-controller-manager",
	})
}

// Start scheduler
func (m *Microkubed) startKubeScheduler() {
	log.Debug("Starting kube-scheduler...")
	kubeSchedHandler, kubeSchedChan, kubeSchedHealthChan := m.startService("kube-scheduler",
		func(kubeSchedOutputHandler handlers.OutputHander,
			kubeSchedExitHandler handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return kube.NewKubeSchedulerHandler(m.hyperkubeBin, path.Join(m.baseDir, "kubesched"),
				path.Join(m.baseDir, "kube/", "kubeconfig"), kubeSchedOutputHandler, kubeSchedExitHandler)
		}, log2.NewKubeLogParser("kube-scheduler"))
	m.serviceHandlers = append(m.serviceHandlers, kubeSchedHandler)
	log.Debug("Kube-scheduler ready")

	m.serviceList = append(m.serviceList, serviceEntry{
		handler:    kubeSchedHandler,
		exitChan:   kubeSchedChan,
		healthChan: kubeSchedHealthChan,
		name:       "kube-scheduler",
	})
}

// Start kubelet
func (m *Microkubed) startKubelet() {
	log.Debug("Starting kubelet...")
	kubeletHandler, kubeletChan, kubeletHealthChan := m.startService("kubelet",
		func(kubeletOutputHandler handlers.OutputHander,
			kubeletExitHandler handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return kube.NewKubeletHandler(m.hyperkubeBin, path.Join(m.baseDir, "kube"), path.Join(m.baseDir, "kube/", "kubeconfig"),
				m.bindAddr.String(), m.kubeServer, m.kubeClient, m.kubeCA, kubeletOutputHandler, kubeletExitHandler)
		}, log2.NewKubeLogParser("kubelet"))
	m.serviceHandlers = append(m.serviceHandlers, kubeletHandler)
	log.Debug("Kubelet ready")

	m.serviceList = append(m.serviceList, serviceEntry{
		handler:    kubeletHandler,
		exitChan:   kubeletChan,
		healthChan: kubeletHealthChan,
		name:       "kubelet",
	})
}

// Start kube-proxy
func (m *Microkubed) startKubeProxy() {
	log.Debug("Starting kube-proxy...")
	kubeProxyHandler, kubeProxyChan, kubeProxyHealthChan := m.startService("kube-proxy",
		func(output handlers.OutputHander, exit handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return kube.NewKubeProxyHandler(m.hyperkubeBin, path.Join(m.baseDir, "kube"),
				path.Join(m.baseDir, "kube/", "kubeconfig"), m.clusterIPRange.String(), output, exit)
		}, log2.NewKubeLogParser("kube-proxy"))
	defer kubeProxyHandler.Stop()
	m.serviceHandlers = append(m.serviceHandlers, kubeProxyHandler)
	log.Debug("kube-proxy ready")

	m.serviceList = append(m.serviceList, serviceEntry{
		handler:    kubeProxyHandler,
		exitChan:   kubeProxyChan,
		healthChan: kubeProxyHealthChan,
		name:       "kube-proxy",
	})
}

func (m *Microkubed) checkService(handler serviceEntry) {
	unhealthyCount := 0
	for {
		select {
		case <-handler.exitChan:
			log.Fatal("Service " + handler.name + " exitted, aborting!")
		case msg := <-handler.healthChan:
			if !msg.IsHealthy {
				log.WithFields(log.Fields{
					"app":   handler.name,
					"count": unhealthyCount,
				}).Warn("unhealthy!")
				unhealthyCount++
				if unhealthyCount == 10 {
					log.WithFields(log.Fields{
						"app":   handler.name,
						"count": unhealthyCount,
					}).Fatal("Too many failed health checks, aborting!")
				}
			} else {
				log.WithField("app", handler.name).Debug("healthy")
				unhealthyCount = 0
			}
		}
	}
}

// Start periodic health checks
func (m *Microkubed) enableHealthChecks() {
	for _, handler := range m.serviceList {
		log.WithField("app", handler.name).Debug("Enabling health check...")
		handler.handler.EnableHealthChecks(handler.healthChan, true)
		go m.checkService(handler)
	}
}

// Wait until node is ready
func (m *Microkubed) waitUntilNodeReady() chan bool {
	kCl, err := cmd.NewKubeClient(path.Join(m.baseDir, "kube/", "kubeconfig"))
	if err != nil {
		log.WithError(err).Fatalf("Couldn't init kube client")
	}
	log.Info("Waiting for node...")
	kCl.WaitForNode()
	// Since we got to this point: Handle quitting gracefully (that is stop all pods!)
	sigChan := make(chan os.Signal, 1)
	exitChan := make(chan bool, 1)
	go func() {
		<-sigChan
		log.Info("Shutting down...")
		kCl.DrainNode()
		exitChan <- true
	}()
	// Unregister "terminate immediately" serviceHandlers set during startup
	signal.Reset(os.Interrupt, os.Kill)
	// Register ordinary exit handler
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	log.Info("Exit handler enabled...")
	m.gracefulTerminationMode = true

	return exitChan
}

// Run the actual command invocation. This function will not return until the program should exit
func (m *Microkubed) Run() {
	m.handleArgs()

	m.gracefulTerminationMode = false
	log.RegisterExitHandler(func() {
		// Fatal() will not run the normal exit serviceHandlers, therefore, we need to run them manually. However, after
		// startup, this shouldn't be used anymore
		if !m.gracefulTerminationMode {
			for _, h := range m.serviceHandlers {
				h.Stop()
			}
		}
	})

	m.createDirectories()
	m.createCertificates()
	m.findBinaries()

	m.startEtcd()
	m.startKubeAPIServer()
	m.startKubeControllerManager()
	m.startKubeScheduler()
	m.startKubelet()
	m.startKubeProxy()

	exitChan := m.waitUntilNodeReady()

	m.enableHealthChecks()

	// Wait until exit
	<-exitChan
	log.WithField("app", "microkube").Info("Exit signal received, stopping now.")
	for _, h := range m.serviceHandlers {
		h.Stop()
	}
	return
}

// Starts a service. This function takes care of setting up the infrastructure required by a service constructor
func (m *Microkubed) startService(name string, constructor serviceConstructor,
	logParser log2.Parser) (handlers.ServiceHandler, chan bool, chan handlers.HealthMessage) {

	log.Debug("Starting " + name + "...")
	outputHandler := func(output []byte) {
		err := logParser.HandleData(output)
		if err != nil {
			log.WithError(err).Warn("Couldn't parse log line!")
		}
	}
	stateChan := make(chan bool, 2)
	healthChan := make(chan handlers.HealthMessage, 2)
	exitHandler := func(success bool, exitError *exec.ExitError) {
		log.WithFields(log.Fields{
			"success": success,
			"app":     name,
		}).WithError(exitError).Error(name + " stopped!")
		if !m.gracefulTerminationMode {
			log.WithFields(log.Fields{
				"success": success,
				"app":     name,
			}).WithError(exitError).Fatal("App exitted during startup phase, bailing out _now_")
		}
		stateChan <- success
	}

	serviceHandler, err := constructor(outputHandler, exitHandler)
	if err != nil {
		log.WithError(err).Fatal("Couldn't create " + name + " handler")
	}
	err = serviceHandler.Start()
	if err != nil {
		log.WithError(err).Fatal("Couldn't start " + name)
	}

	msg := handlers.HealthMessage{
		IsHealthy: false,
	}
	for retries := 0; retries < 8 && !msg.IsHealthy; retries++ {
		time.Sleep(1 * time.Second)
		serviceHandler.EnableHealthChecks(healthChan, false)
		msg = <-healthChan
		log.WithFields(log.Fields{
			"app":    name,
			"health": msg.IsHealthy,
		}).Debug("Healthcheck")
	}
	if !msg.IsHealthy {
		log.WithError(msg.Error).Fatal(name + " didn't become healthy in time!")
	}

	return serviceHandler, stateChan, healthChan
}
