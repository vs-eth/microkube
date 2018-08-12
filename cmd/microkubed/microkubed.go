package main

import (
	"context"
	"flag"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/uubk/microkube/internal/cmd"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/handlers/controller-manager"
	"github.com/uubk/microkube/pkg/handlers/etcd"
	"github.com/uubk/microkube/pkg/handlers/kube-apiserver"
	"github.com/uubk/microkube/pkg/handlers/kube-scheduler"
	"github.com/uubk/microkube/pkg/handlers/kubelet"
	"github.com/uubk/microkube/pkg/helpers"
	"net"
	"os"
	"os/exec"
	"path"
	"time"
	etcd2 "github.com/uubk/microkube/internal/log/etcd"
	"github.com/uubk/microkube/internal/log/kube"
)

func getDockerIPRanges() (myIP, podRangeStr, serviceRangeStr string) {
	// Figure out if Docker is running and if it's network is something that we can use
	docker, err := client.NewEnvClient()
	if err != nil {
		log.WithError(err).Fatal("Couldn't connect to the docker daemon")
		os.Exit(-1)
	}
	dockerNetworks, err := docker.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		log.WithError(err).Fatal("Couldn't list docker networks")
		os.Exit(-1)
	}
	dockerNetworkWanted := "bridge"
	dockerNetworkIP := ""
	dockerNetworkCIDR := ""
	for _, net := range dockerNetworks {
		if net.Name == dockerNetworkWanted {
			if net.IPAM.Config == nil || len(net.IPAM.Config) != 1 {
				log.Fatal("Docker network '" + dockerNetworkWanted + "' has wrong IP configuration")
				os.Exit(-1)
			}
			dockerNetworkIP = net.IPAM.Config[0].Gateway
			dockerNetworkCIDR = net.IPAM.Config[0].Subnet
		}
	}
	if dockerNetworkIP == "" || dockerNetworkCIDR == "" {
		log.Fatal("Docker network '" + dockerNetworkWanted + "' not found!")
		os.Exit(-1)
	}
	// Now that we have the network, let's try to split it into a pod and a service range
	_, fullNet, err := net.ParseCIDR(dockerNetworkCIDR)
	if err != nil {
		log.WithError(err).Fatal("Couldn't parse docker network CIDR")
		os.Exit(-1)
	}
	ones, bits := fullNet.Mask.Size()
	if ones != 16 || bits != 32 {
		log.WithFields(log.Fields{
			"ones": ones,
			"bits": bits,
		}).Fatal("Unexpected netmask on docker CIDR")
		os.Exit(-1)
	}
	podRange := net.IPNet{
		IP:   make(net.IP, 4),
		Mask: make(net.IPMask, 4),
	}
	copy(podRange.IP, fullNet.IP)
	copy(podRange.Mask, fullNet.Mask)
	podRange.Mask[2] = 255
	serviceRange := net.IPNet{
		IP:   make(net.IP, 4),
		Mask: make(net.IPMask, 4),
	}
	copy(serviceRange.IP, fullNet.IP)
	copy(serviceRange.Mask, fullNet.Mask)
	serviceRange.Mask[2] = 255
	serviceRange.IP[2]++
	log.WithFields(log.Fields{
		"pods": podRange.String(),
		"svcs": serviceRange.String(),
	}).Info("Network ranges calculated")

	return dockerNetworkIP, podRange.String(), serviceRange.String()
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	verbose := flag.Bool("verbose", true, "Enabel verbose output")
	root := flag.String("root", "~/.mukube", "Microkube root directory")
	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	dir, err := homedir.Expand(*root)
	if err != nil {
		log.WithError(err).WithField("root", *root).Fatal("Couldn't expand root directory")
		os.Exit(-1)
	}
	dockerNetworkIP, podRange, serviceRange := getDockerIPRanges()

	// Handle certs
	cmd.EnsureDir(dir, "", 0770)
	cmd.EnsureDir(dir, "kube", 0770)
	cmd.EnsureDir(dir, "etcdtls", 0770)
	cmd.EnsureDir(dir, "kubesched", 0770)
	cmd.EnsureDir(dir, "kubetls", 0770)
	cmd.EnsureDir(dir, "kubectls", 0770)
	cmd.EnsureDir(dir, "kubestls", 0770)
	cmd.EnsureDir(dir, "etcddata", 0770)
	etcdCA, etcdServer, etcdClient, err := cmd.EnsureFullPKI(path.Join(dir, "etcdtls"), "Microkube ETCD", false, true, []string{dockerNetworkIP})
	if err != nil {
		log.WithError(err).Fatal("Couldn't create etcd PKI")
		os.Exit(-1)
	}
	kubeCA, kubeServer, kubeClient, err := cmd.EnsureFullPKI(path.Join(dir, "kubetls"), "Microkube Kubernetes", true, false, []string{dockerNetworkIP})
	if err != nil {
		log.WithError(err).Fatal("Couldn't create kubernetes PKI")
		os.Exit(-1)
	}
	kubeClusterCA, err := cmd.EnsureCA(path.Join(dir, "kubectls"), "Microkube Cluster CA")
	if err != nil {
		log.WithError(err).Fatal("Couldn't create kubernetes cluster CA")
		os.Exit(-1)
	}
	kubeSvcSignCert, err := cmd.EnsureSigningCert(path.Join(dir, "kubestls"), "Microkube Cluster SVC Signcert")
	if err != nil {
		log.WithError(err).Fatal("Couldn't create kubernetes service secret signing certificate")
		os.Exit(-1)
	}

	// Find binaries
	etcdBin, err := helpers.FindBinary("etcd", dir)
	if err != nil {
		log.WithError(err).Fatal("Couldn't find etcd binary")
		os.Exit(-1)
	}
	kubeApiBin, err := helpers.FindBinary("kube-apiserver", dir)
	if err != nil {
		log.WithError(err).Fatal("Couldn't find kube apiserver binary")
		os.Exit(-1)
	}
	kubeletBin, err := helpers.FindBinary("kubelet", dir)
	if err != nil {
		log.WithError(err).Fatal("Couldn't find kubelet binary")
		os.Exit(-1)
	}
	ctrlMgrBin, err := helpers.FindBinary("kube-controller-manager", dir)
	if err != nil {
		log.WithError(err).Fatal("Couldn't find kube-controller-manager binary")
		os.Exit(-1)
	}
	kubeSchedBin, err := helpers.FindBinary("kube-scheduler", dir)
	if err != nil {
		log.WithError(err).Fatal("Couldn't find kube-scheduler binary")
		os.Exit(-1)
	}

	// Start etcd
	log.Debug("Starting etcd...")
	etcdLogParser := etcd2.NewETCDLogParser()
	etcdOutputHandler := func(output []byte) {
		err := etcdLogParser.HandleData(output)
		if err != nil {
			log.WithError(err).Warn("Couldn't parse log line!")
		}
	}
	etcdChan := make(chan bool, 2)
	etcdHealthChan := make(chan handlers.HealthMessage, 2)
	etcdExitHandler := func(success bool, exitError *exec.ExitError) {
		log.WithFields(log.Fields{
			"success": success,
			"app":     "etcd",
		}).WithError(exitError).Error("etcd stopped!")
		etcdChan <- success
	}
	etcdHandler := etcd.NewEtcdHandler(path.Join(dir, "etcddata"), etcdBin, etcdServer, etcdClient, etcdCA,
		etcdOutputHandler, etcdExitHandler)
	err = etcdHandler.Start()
	if err != nil {
		log.WithError(err).Fatal("Couldn't start etcd")
		os.Exit(-1)
	}
	defer etcdHandler.Stop()
	etcdHandler.EnableHealthChecks(etcdHealthChan, false)
	msg := <-etcdHealthChan
	if !msg.IsHealthy {
		log.WithError(msg.Error).Fatal("ETCD didn't become healthy in time!")
		return
	} else {
		log.Debug("ETCD ready")
	}

	// Start Kube APIServer
	log.Debug("Starting kube api server...")

	kubeApiLogParser := kube.NewKubeLogParser("kube-api")
	kubeAPIOutputHandler := func(output []byte) {
		err := kubeApiLogParser.HandleData(output)
		if err != nil {
			log.WithError(err).Warn("Couldn't parse log line!")
		}
	}
	kubeAPIChan := make(chan bool, 2)
	kubeAPIHealthChan := make(chan handlers.HealthMessage, 2)
	kubeAPIExitHandler := func(success bool, exitError *exec.ExitError) {
		log.WithFields(log.Fields{
			"success": success,
			"app":     "kube-api",
		}).WithError(exitError).Error("kube-apiserver stopped!")
		kubeAPIChan <- success
	}
	kubeAPIHandler := kube_apiserver.NewKubeAPIServerHandler(kubeApiBin, kubeServer, kubeClient, kubeCA, etcdClient,
		etcdCA, kubeAPIOutputHandler, kubeAPIExitHandler, dockerNetworkIP, serviceRange)
	err = kubeAPIHandler.Start()
	if err != nil {
		log.WithError(err).Fatal("Couldn't start kube apiserver")
		os.Exit(-1)
	}
	defer kubeAPIHandler.Stop()

	msg = handlers.HealthMessage{
		IsHealthy: false,
	}
	for retries := 0; retries < 8 && !msg.IsHealthy; retries++ {
		time.Sleep(1 * time.Second)
		kubeAPIHandler.EnableHealthChecks(kubeAPIHealthChan, false)
		msg = <-kubeAPIHealthChan
		log.WithFields(log.Fields{
			"app":    "kube-api",
			"health": msg.IsHealthy,
		}).Debug("Healthcheck")
	}
	if !msg.IsHealthy {
		log.WithError(msg.Error).Fatal("Kube APIServer didn't become healthy in time!")
		return
	}

	// Generate kubeconfig for kubelet and kubectl
	log.Debug("Generating kubeconfig...")
	kubeconfig := path.Join(dir, "kube/", "kubeconfig")
	_, err = os.Stat(kubeconfig)
	if err != nil {
		log.Debug("Creating kubeconfig")
		err = kube_apiserver.CreateClientKubeconfig(kubeCA, kubeClient, kubeconfig, dockerNetworkIP)
		if err != nil {
			log.WithError(err).Fatal("Couldn't create kubeconfig!")
			return
		}
	}

	// Start controller-manager
	log.Debug("Starting controller-manager...")
	kubeCtrlMgrParser := kube.NewKubeLogParser("kube-controller-manager")
	kubeCtrlMgrOutputHandler := func(output []byte) {
		err := kubeCtrlMgrParser.HandleData(output)
		if err != nil {
			log.WithError(err).Warn("Couldn't parse log line!")
		}
	}
	kubeCtrlMgrChan := make(chan bool, 2)
	kubeCtrlMgrHealthChan := make(chan handlers.HealthMessage, 2)
	kubeCtrlMgrExitHandler := func(success bool, exitError *exec.ExitError) {
		log.WithFields(log.Fields{
			"success": success,
			"app":     "controller-manager",
		}).WithError(exitError).Error("kubelet stopped!")
		kubeCtrlMgrChan <- success
	}
	kubeCtrlMgrHandler := controller_manager.NewControllerManagerHandler(ctrlMgrBin, path.Join(dir, "kube/", "kubeconfig"),
		dockerNetworkIP, kubeServer, kubeClient, kubeCA, kubeClusterCA, kubeSvcSignCert, podRange, kubeCtrlMgrOutputHandler, kubeCtrlMgrExitHandler)
	if err != nil {
		log.WithError(err).Fatal("Couldn't create controller-manager handler")
		os.Exit(-1)
	}
	err = kubeCtrlMgrHandler.Start()
	if err != nil {
		log.WithError(err).Fatal("Couldn't start controller-manager")
		os.Exit(-1)
	}
	defer kubeCtrlMgrHandler.Stop()

	msg = handlers.HealthMessage{
		IsHealthy: false,
	}
	for retries := 0; retries < 8 && !msg.IsHealthy; retries++ {
		time.Sleep(1 * time.Second)
		kubeCtrlMgrHandler.EnableHealthChecks(kubeCtrlMgrHealthChan, false)
		msg = <-kubeCtrlMgrHealthChan
		log.WithFields(log.Fields{
			"app":    "controller-manager",
			"health": msg.IsHealthy,
		}).Debug("Healthcheck")
	}
	if !msg.IsHealthy {
		log.WithError(msg.Error).Fatal("Controller-manager didn't become healthy in time!")
		return
	}

	// Start scheduler
	log.Debug("Starting kube-scheduler...")
	kubeSchedParser := kube.NewKubeLogParser("kube-scheduler")
	kubeSchedOutputHandler := func(output []byte) {
		err := kubeSchedParser.HandleData(output)
		if err != nil {
			log.WithError(err).Warn("Couldn't parse log line!")
		}
	}
	kubeSchedChan := make(chan bool, 2)
	kubeSchedHealthChan := make(chan handlers.HealthMessage, 2)
	kubeSchedExitHandler := func(success bool, exitError *exec.ExitError) {
		log.WithFields(log.Fields{
			"success": success,
			"app":     "kube-scheduler",
		}).WithError(exitError).Error("kube-scheduler stopped!")
		kubeSchedChan <- success
	}
	kubeSchedHandler, err := kube_scheduler.NewKubeSchedulerHandler(kubeSchedBin, path.Join(dir, "kubesched"), path.Join(dir, "kube/", "kubeconfig"), kubeSchedOutputHandler, kubeSchedExitHandler)
	if err != nil {
		log.WithError(err).Fatal("Couldn't create kube-scheduler handler")
		os.Exit(-1)
	}
	err = kubeSchedHandler.Start()
	if err != nil {
		log.WithError(err).Fatal("Couldn't start kube-scheduler")
		os.Exit(-1)
	}
	defer kubeSchedHandler.Stop()

	msg = handlers.HealthMessage{
		IsHealthy: false,
	}
	for retries := 0; retries < 8 && !msg.IsHealthy; retries++ {
		time.Sleep(1 * time.Second)
		kubeSchedHandler.EnableHealthChecks(kubeSchedHealthChan, false)
		msg = <-kubeSchedHealthChan
		log.WithFields(log.Fields{
			"app":    "kube-scheduler",
			"health": msg.IsHealthy,
		}).Debug("Healthcheck")
	}
	if !msg.IsHealthy {
		log.WithError(msg.Error).Fatal("Kube-scheduler didn't become healthy in time!")
		return
	}

	// Start kubelet
	log.Debug("Starting kubelet...")
	kubeletParser := kube.NewKubeLogParser("kube-scheduler")
	kubeletOutputHandler := func(output []byte) {
		err := kubeletParser.HandleData(output)
		if err != nil {
			log.WithError(err).Warn("Couldn't parse log line!")
		}
	}
	kubeletChan := make(chan bool, 2)
	kubeletHealthChan := make(chan handlers.HealthMessage, 2)
	kubeletExitHandler := func(success bool, exitError *exec.ExitError) {
		log.WithFields(log.Fields{
			"success": success,
			"app":     "kubelet",
		}).WithError(exitError).Error("kubelet stopped!")
		kubeletChan <- success
	}
	kubeletHandler, err := kubelet.NewKubeletHandler(kubeletBin, path.Join(dir, "kube"), path.Join(dir, "kube/", "kubeconfig"),
		dockerNetworkIP, kubeServer, kubeClient, kubeCA, kubeletOutputHandler, kubeletExitHandler)
	if err != nil {
		log.WithError(err).Fatal("Couldn't create kubelet handler")
		os.Exit(-1)
	}
	err = kubeletHandler.Start()
	if err != nil {
		log.WithError(err).Fatal("Couldn't start kubelet")
		os.Exit(-1)
	}
	defer kubeletHandler.Stop()

	msg = handlers.HealthMessage{
		IsHealthy: false,
	}
	for retries := 0; retries < 8 && !msg.IsHealthy; retries++ {
		time.Sleep(1 * time.Second)
		kubeletHandler.EnableHealthChecks(kubeletHealthChan, false)
		msg = <-kubeletHealthChan
		log.WithFields(log.Fields{
			"app":    "kubelet",
			"health": msg.IsHealthy,
		}).Debug("Healthcheck")
	}
	if !msg.IsHealthy {
		log.WithError(msg.Error).Fatal("Kubelet didn't become healthy in time!")
		return
	}

	// Start periodic health checks
	etcdHandler.EnableHealthChecks(etcdHealthChan, true)
	kubeAPIHandler.EnableHealthChecks(kubeAPIHealthChan, true)
	kubeletHandler.EnableHealthChecks(kubeletHealthChan, true)
	kubeCtrlMgrHandler.EnableHealthChecks(kubeCtrlMgrHealthChan, true)
	kubeSchedHandler.EnableHealthChecks(kubeSchedHealthChan, true)

	// Main loop
	for {
		select {
		case <-kubeAPIChan:
			log.Fatal("Kube API server exitted, aborting!")
			return
		case <-etcdChan:
			log.Fatal("ETCD exitted, aborting!")
			return
		case <-kubeletChan:
			log.Fatal("Kubelet exitted, aborting!")
			return
		case <-kubeCtrlMgrChan:
			log.Fatal("Controller/manager exitted, aborting!")
			return
		case <-kubeSchedChan:
			log.Fatal("Scheduler exitted, aborting!")
			return
		case msg = <-etcdHealthChan:
			if !msg.IsHealthy {
				log.WithField("app", "etcd").Warn("unhealthy!")
			} else {
				log.WithField("app", "etcd").Debug("healthy")
			}
		case msg = <-kubeAPIHealthChan:
			if !msg.IsHealthy {
				log.WithField("app", "kube-api").Warn("unhealthy!")
			} else {
				log.WithField("app", "kube-api").Debug("healthy")
			}
		case msg = <-kubeletHealthChan:
			if !msg.IsHealthy {
				log.WithField("app", "kubelet").Warn("unhealthy!")
			} else {
				log.WithField("app", "kubelet").Debug("healthy")
			}
		case msg = <-kubeCtrlMgrHealthChan:
			if !msg.IsHealthy {
				log.WithField("app", "controller-manager").Warn("unhealthy!")
			} else {
				log.WithField("app", "controller-manager").Debug("healthy")
			}
		case msg = <-kubeSchedHealthChan:
			if !msg.IsHealthy {
				log.WithField("app", "kube-scheduler").Warn("unhealthy!")
			} else {
				log.WithField("app", "kube-scheduler").Debug("healthy")
			}
		}
	}
}
