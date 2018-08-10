package main

import (
	"context"
	"crypto/x509/pkix"
	"flag"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uubk/microkube/pkg/handlers/etcd"
	"github.com/uubk/microkube/pkg/handlers/kube-apiserver"
	"github.com/uubk/microkube/pkg/helpers"
	"github.com/uubk/microkube/pkg/pki"
	"os"
	"os/exec"
	"path"
	"time"
	"github.com/mitchellh/go-homedir"
)

func ensureDir(root, sub string) {
	dir := path.Join(root, sub)

	// Errors in mkdir are ignored
	err := os.Mkdir(dir, 0770)
	if err == nil {
		log.WithField("dir", dir).Debug("Directory created")
	}

	info, err := os.Stat(dir)
	if err != nil {
		log.WithError(err).Fatal("Couldn't stat microkube root directory")
		os.Exit(-1)
	}
	if !info.IsDir() {
		log.WithError(err).Fatal("Microkube root directory is not a directory")
		os.Exit(-1)
	}
}

func ensureCerts(root, name string, isKubeCA bool, ip string) (*pki.RSACertificate, *pki.RSACertificate, *pki.RSACertificate) {
	cafile := path.Join(root, "ca.pem")
	_, err := os.Stat(cafile)
	if err != nil {
		// File doesn't exist
		certmgr := pki.NewManager(root)
		ca, err := certmgr.NewSelfSignedCert("ca", pkix.Name{
			CommonName: name + " CA",
		}, 1)
		if err != nil {
			log.WithError(err).WithField("root", root).Fatal("Couldn't create CA")
			os.Exit(-1)
		}

		server, err := certmgr.NewCert("server", pkix.Name{
			CommonName: name + " Server",
		}, 2, true, []string{
			"127.0.0.1",
			"localhost",
			ip,
		}, ca)
		if err != nil {
			log.WithError(err).WithField("root", root).Fatal("Couldn't create server cert")
			os.Exit(-1)
		}

		clientname := pkix.Name{
			CommonName: name + " Client",
		}
		if isKubeCA {
			clientname.Organization = []string{"system:masters"}
		}
		client, err := certmgr.NewCert("client", clientname, 3, false, nil, ca)
		if err != nil {
			log.WithError(err).WithField("root", root).Fatal("Couldn't create client cert")
			os.Exit(-1)
		}

		return ca, server, client
	}

	// Certs already exist
	return &pki.RSACertificate{
			KeyPath:  "",
			CertPath: path.Join(root, "ca.pem"),
		}, &pki.RSACertificate{
			KeyPath:  path.Join(root, "server.key"),
			CertPath: path.Join(root, "server.pem"),
		}, &pki.RSACertificate{
			KeyPath:  path.Join(root, "client.key"),
			CertPath: path.Join(root, "client.pem"),
		}
}

func GetBinary(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "couldn't read cwd")
	}
	wd = path.Join(wd, "third_party", name)
	return wd, nil
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

	// Handle certs
	ensureDir(dir, "")
	ensureDir(dir, "kube")
	ensureDir(dir, "etcdtls")
	ensureDir(dir, "kubetls")
	ensureDir(dir, "etcddata")
	etcdCA, etcdServer, etcdClient := ensureCerts(path.Join(dir, "etcdtls"), "Microkube ETCD", false, dockerNetworkIP)
	kubeCA, kubeServer, kubeClient := ensureCerts(path.Join(dir, "kubetls"), "Microkube Kubernetes", true, dockerNetworkIP)

	// Find binaries
	etcdBin, err := GetBinary("etcd")
	if err != nil {
		log.WithError(err).Fatal("Couldn't find etcd binary")
		os.Exit(-1)
	}
	kubeApiBin, err := GetBinary("kube-apiserver")
	if err != nil {
		log.WithError(err).Fatal("Couldn't find kube apiserver binary")
		os.Exit(-1)
	}

	// Start etcd
	log.Debug("Starting etcd...")
	etcdOutputHandler := func(output []byte) {
		log.WithField("app", "etcd").Info(string(output))
	}
	etcdChan := make(chan bool, 2)
	etcdHealthChan := make(chan helpers.HealthMessage, 2)
	etcdExitHandler := func(success bool, exitError *exec.ExitError) {
		log.WithFields(log.Fields{
			"success": success,
			"app":     "etcd",
		}).WithError(exitError).Error("etcd stopped!")
		etcdChan <- success
	}
	etcdHandler := etcd.NewEtcdHandler(path.Join(dir, "etcddata"), etcdBin, etcdServer.CertPath, etcdServer.KeyPath,
		etcdCA.CertPath, etcdOutputHandler, 3, etcdExitHandler)
	err = etcdHandler.Start()
	if err != nil {
		log.WithError(err).Fatal("Couldn't start etcd")
		os.Exit(-1)
	}
	defer etcdHandler.Stop()
	etcdHandler.EnableHealthChecks(etcdCA, etcdClient, etcdHealthChan, false)
	msg := <-etcdHealthChan
	if !msg.IsHealthy {
		log.WithError(msg.Error).Fatal("ETCD didn't become healthy in time!")
		return
	}

	// Start Kube APIServer
	log.Debug("Starting kube api server...")
	kubeAPIOutputHandler := func(output []byte) {
		log.WithField("app", "kube-api").Info(string(output))
	}
	kubeAPIChan := make(chan bool, 2)
	kubeAPIHealthChan := make(chan helpers.HealthMessage, 2)
	kubeAPIExitHandler := func(success bool, exitError *exec.ExitError) {
		log.WithFields(log.Fields{
			"success": success,
			"app":     "kube-api",
		}).WithError(exitError).Error("kube-apiserver stopped!")
		kubeAPIChan <- success
	}
	kubeAPIHandler := kube_apiserver.NewKubeAPIServerHandler(kubeApiBin, kubeServer.CertPath, kubeServer.KeyPath,
		kubeClient.CertPath, kubeClient.KeyPath, kubeCA.CertPath, etcdClient.CertPath, etcdClient.KeyPath,
		etcdCA.CertPath, kubeAPIOutputHandler, kubeAPIExitHandler, dockerNetworkIP)
	err = kubeAPIHandler.Start()
	if err != nil {
		log.WithError(err).Fatal("Couldn't start kube apiserver")
		os.Exit(-1)
	}
	defer kubeAPIHandler.Stop()

	msg = helpers.HealthMessage{
		IsHealthy: false,
	}
	for retries := 0; retries < 8 && !msg.IsHealthy; retries++ {
		time.Sleep(1 * time.Second)
		kubeAPIHandler.EnableHealthChecks(kubeCA, kubeClient, kubeAPIHealthChan, false)
		msg = <-kubeAPIHealthChan
		log.WithFields(log.Fields{
			"app": "kube-api",
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


	// Start periodic health checks
	etcdHandler.EnableHealthChecks(etcdCA, etcdClient, etcdHealthChan, true)
	kubeAPIHandler.EnableHealthChecks(kubeCA, kubeClient, kubeAPIHealthChan, true)

	// Main loop
	for {
		select {
		case <-kubeAPIChan:
			log.Fatal("Kube API server exitted, aborting!")
			return
		case <-etcdChan:
			log.Fatal("ETCD exitted, aborting!")
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
		}
	}
}
