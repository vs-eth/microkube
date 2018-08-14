package main

import (
	"flag"
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
	"os/signal"
)

var isDoneStarting bool

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
		if !isDoneStarting {
			log.WithFields(log.Fields{
				"success": success,
				"app":     name,
			}).WithError(exitError).Fatal("App exitted during startup phase, bailing out _now_")
			os.Exit(-1)
		}
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

func main() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	isDoneStarting = false

	verbose := flag.Bool("verbose", true, "Enable verbose output")
	root := flag.String("root", "~/.mukube", "Microkube root directory")
	podRange := flag.String("pod-range", "10.233.42.1/24", "Pod IP range to use")
	serviceRange := flag.String("service-range", "10.233.43.1/24", "Service IP range to use")
	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	dir, err := homedir.Expand(*root)
	if err != nil {
		log.WithError(err).WithField("root", *root).Fatal("Couldn't expand root directory")
		os.Exit(-1)
	}

	podRangeIP, podRangeNet, err := net.ParseCIDR(*podRange)
	if err != nil {
		log.WithFields(log.Fields{
			"range": *podRange,
		}).WithError(err).Fatal("Couldn't parse pod CIDR range")
		return
	}
	serviceRangeIP, serviceRangeNet, err := net.ParseCIDR(*serviceRange)
	if err != nil {
		log.WithFields(log.Fields{
			"range": *podRange,
		}).WithError(err).Fatal("Couldn't parse service CIDR range")
		return
	}
	bindAddr := cmd.FindBindAddress()

	// To combine pod and service range to form the cluster range, find first diverging bit
	baseOffset := 0
	serviceBelowPod := false
	for idx, octet := range serviceRangeNet.IP {
		if podRangeNet.IP[idx] != octet {
			// This octet diverges -> find bit
			baseOffset = idx*8
			for mask := byte(0x80) ; mask > 0 ; mask /= 2 {
				baseOffset++
				if (podRangeNet.IP[idx] & mask) != (octet & mask) {
					// Found it
					serviceBelowPod = octet < podRangeNet.IP[idx]
					break
				}
			}
			baseOffset--
		}
	}
	clusterIPRange := net.IPNet{
		IP: podRangeIP,
	}
	if serviceBelowPod {
		clusterIPRange.IP = serviceRangeIP
	}
	clusterIPRange.Mask = net.CIDRMask(baseOffset, 32)
	log.WithFields(log.Fields{
		"podRange":     podRangeNet.String(),
		"serviceRange": serviceRangeNet.String(),
		"clusterRange": clusterIPRange.String(),
		"hostIP":       bindAddr,
	}).Info("IP ranges calculated")

	//bindAddr, podRange, serviceRange := getDockerIPRanges()

	// Handle certs
	cmd.EnsureDir(dir, "", 0770)
	cmd.EnsureDir(dir, "kube", 0770)
	cmd.EnsureDir(dir, "etcdtls", 0770)
	cmd.EnsureDir(dir, "kubesched", 0770)
	cmd.EnsureDir(dir, "kubetls", 0770)
	cmd.EnsureDir(dir, "kubectls", 0770)
	cmd.EnsureDir(dir, "kubestls", 0770)
	cmd.EnsureDir(dir, "etcddata", 0770)
	etcdCA, etcdServer, etcdClient, err := cmd.EnsureFullPKI(path.Join(dir, "etcdtls"), "Microkube ETCD", false, true, []string{bindAddr})
	if err != nil {
		log.WithError(err).Fatal("Couldn't create etcd PKI")
		os.Exit(-1)
	}
	kubeCA, kubeServer, kubeClient, err := cmd.EnsureFullPKI(path.Join(dir, "kubetls"), "Microkube Kubernetes", true, false, []string{bindAddr, serviceRangeIP.String()})
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
			return kube_apiserver.NewKubeAPIServerHandler(kubeApiBin, kubeServer, kubeClient, kubeCA, kubeSvcSignCert, etcdClient,
				etcdCA, kubeAPIOutputHandler, kubeAPIExitHandler, bindAddr, *serviceRange), nil
		}, kube.NewKubeLogParser("kube-api"))
	defer kubeAPIHandler.Stop()
	log.Debug("Kube api server ready")

	// Generate kubeconfig for kubelet and kubectl
	log.Debug("Generating kubeconfig...")
	kubeconfig := path.Join(dir, "kube/", "kubeconfig")
	_, err = os.Stat(kubeconfig)
	if err != nil {
		log.Debug("Creating kubeconfig")
		err = kube_apiserver.CreateClientKubeconfig(kubeCA, kubeClient, kubeconfig, bindAddr)
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
				bindAddr, kubeServer, kubeClient, kubeCA, kubeClusterCA, kubeSvcSignCert, *podRange,
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
				bindAddr, kubeServer, kubeClient, kubeCA, kubeletOutputHandler, kubeletExitHandler)
		}, kube.NewKubeLogParser("kubelet"))
	defer kubeSchedHandler.Stop()
	log.Debug("Kubelet ready")

	// Start kube-proxy
	log.Debug("Starting kube-proxy...")
	kubeProxyHandler, kubeProxyChan, kubeProxyHealthChan := startService("kube-proxy",
		func(output handlers.OutputHander, exit handlers.ExitHandler) (handlers.ServiceHandler, error) {
			return kube_proxy.NewKubeProxyHandler(kubeProxyBin, path.Join(dir, "kube"),
				path.Join(dir, "kube/", "kubeconfig"), clusterIPRange.String(), output, exit)
		}, kube.NewKubeLogParser("kube-proxy"))
	defer kubeProxyHandler.Stop()
	log.Debug("kube-proxy ready")

	kCl, err := cmd.NewKubeClient(path.Join(dir, "kube/", "kubeconfig"))
	if err != nil {
		log.WithError(err).Fatalf("Couldn't init kube client")
		return
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
	// Unregister "terminate immediately" handlers set during startup
	signal.Reset(os.Interrupt, os.Kill)
	// Register ordinary exit handler
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	log.Info("Exit handler enabled...")
	isDoneStarting = true

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
		case <-exitChan:
			log.WithField("app", "microkube").Info("Exit signal received, stopping now.")
			return
		}
	}
}
