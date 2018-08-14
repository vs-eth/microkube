package main

import (
	"context"
	"flag"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/uubk/microkube/internal/cmd"
	log2 "github.com/uubk/microkube/internal/log"
	etcd2 "github.com/uubk/microkube/internal/log/etcd"
	"github.com/uubk/microkube/internal/log/kube"
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
	"github.com/uubk/microkube/pkg/handlers/kube-proxy"
)

type serviceConstructor func(handlers.OutputHander, handlers.ExitHandler) (handlers.ServiceHandler, error)

func startService(name string, constructor serviceConstructor, logParser log2.LogParser) (handlers.ServiceHandler, chan bool, chan handlers.HealthMessage) {
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
		stateChan <- success
	}

	serviceHandler, err := constructor(outputHandler, exitHandler)
	if err != nil {
		log.WithError(err).Fatal("Couldn't create " + name + " handler")
		os.Exit(-1)
	}
	err = serviceHandler.Start()
	if err != nil {
		log.WithError(err).Fatal("Couldn't start " + name)
		os.Exit(-1)
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
		os.Exit(-1)
	}

	return serviceHandler, stateChan, healthChan
}

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
	kubeProxyBin, err := helpers.FindBinary("kube-proxy", dir)
	if err != nil {
		log.WithError(err).Fatal("Couldn't find kube-proxy binary")
		os.Exit(-1)
	}

	// Start etcd
	etcdHandler, etcdChan, etcdHealthChan := startService("etcd", func(etcdOutputHandler handlers.OutputHander,
		etcdExitHandler handlers.ExitHandler) (handlers.ServiceHandler, error) {
		return etcd.NewEtcdHandler(path.Join(dir, "etcddata"), etcdBin, etcdServer, etcdClient, etcdCA,
			etcdOutputHandler, etcdExitHandler), nil
	}, etcd2.NewETCDLogParser())
	defer etcdHandler.Stop()
	log.Debug("ETCD ready")

	// Start Kube APIServer
	log.Debug("Starting kube api server...")
	kubeAPIHandler, kubeAPIChan, kubeAPIHealthChan := startService("kube-apiserver",
		func(kubeAPIOutputHandler handlers.OutputHander,
			kubeAPIExitHandler handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return kube_apiserver.NewKubeAPIServerHandler(kubeApiBin, kubeServer, kubeClient, kubeCA, etcdClient,
				etcdCA, kubeAPIOutputHandler, kubeAPIExitHandler, dockerNetworkIP, serviceRange), nil
		}, kube.NewKubeLogParser("kube-api"))
	defer kubeAPIHandler.Stop()
	log.Debug("Kube api server ready")

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
	kubeCtrlMgrHandler, kubeCtrlMgrChan, kubeCtrlMgrHealthChan := startService("kube-controller-manager",
		func(kubeCtrlMgrOutputHandler handlers.OutputHander,
			kubeCtrlMgrExitHandler handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return controller_manager.NewControllerManagerHandler(ctrlMgrBin, path.Join(dir, "kube/", "kubeconfig"),
				dockerNetworkIP, kubeServer, kubeClient, kubeCA, kubeClusterCA, kubeSvcSignCert, podRange,
				kubeCtrlMgrOutputHandler, kubeCtrlMgrExitHandler), nil
		}, kube.NewKubeLogParser("kube-controller-manager"))
	defer kubeCtrlMgrHandler.Stop()
	log.Debug("Kube controller-manager ready")

	// Start scheduler
	log.Debug("Starting kube-scheduler...")
	kubeSchedHandler, kubeSchedChan, kubeSchedHealthChan := startService("kube-scheduler",
		func(kubeSchedOutputHandler handlers.OutputHander,
			kubeSchedExitHandler handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return kube_scheduler.NewKubeSchedulerHandler(kubeSchedBin, path.Join(dir, "kubesched"),
				path.Join(dir, "kube/", "kubeconfig"), kubeSchedOutputHandler, kubeSchedExitHandler)
		}, kube.NewKubeLogParser("kube-scheduler"))
	defer kubeSchedHandler.Stop()
	log.Debug("Kube-scheduler ready")

	// Start kubelet
	log.Debug("Starting kubelet...")
	kubeletHandler, kubeletChan, kubeletHealthChan := startService("kubelet",
		func(kubeletOutputHandler handlers.OutputHander,
			kubeletExitHandler handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return kubelet.NewKubeletHandler(kubeletBin, path.Join(dir, "kube"), path.Join(dir, "kube/", "kubeconfig"),
				dockerNetworkIP, kubeServer, kubeClient, kubeCA, kubeletOutputHandler, kubeletExitHandler)
		}, kube.NewKubeLogParser("kubelet"))
	defer kubeSchedHandler.Stop()
	log.Debug("Kubelet ready")

	// Start kube-proxy
	log.Debug("Starting kube-prox...")
	kubeProxyHandler, kubeProxyChan, kubeProxyHealthChan := startService("kube-proxy",
		func(output handlers.OutputHander, exit handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return kube_proxy.NewKubeProxyHandler(kubeProxyBin, path.Join(dir, "kube"),
				path.Join(dir, "kube/", "kubeconfig"), "", output, exit)
		}, kube.NewKubeLogParser("kube-proxy"))
	defer kubeProxyHandler.Stop()
	log.Debug("kube-proxy ready")

	// Start periodic health checks
	etcdHandler.EnableHealthChecks(etcdHealthChan, true)
	kubeAPIHandler.EnableHealthChecks(kubeAPIHealthChan, true)
	kubeletHandler.EnableHealthChecks(kubeletHealthChan, true)
	kubeCtrlMgrHandler.EnableHealthChecks(kubeCtrlMgrHealthChan, true)
	kubeSchedHandler.EnableHealthChecks(kubeSchedHealthChan, true)

	// Main loop
	var msg handlers.HealthMessage
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
		case <-kubeProxyChan:
			log.Fatal("Proxy exitted, aborting!")
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
		case msg = <-kubeProxyHealthChan:
			if !msg.IsHealthy {
				log.WithField("app", "kube-proxy").Warn("unhealthy!")
			} else {
				log.WithField("app", "kube-proxy").Debug("healthy")
			}
		}
	}
}
