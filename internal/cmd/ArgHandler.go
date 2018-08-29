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
	BaseDir         string
	ExtraBinDir     string
	PodRangeNet     *net.IPNet
	ServiceRangeNet *net.IPNet
	ClusterIPRange  *net.IPNet
}

// HandleArgs registers and evaluates command-line arguments
func (a *ArgHandler) HandleArgs() *handlers.ExecutionEnvironment {
	verbose := flag.Bool("verbose", true, "Enable verbose output")
	root := flag.String("root", "~/.mukube", "Microkube root directory")
	extraBinDir := flag.String("extra-bin-dir", "", "Additional directory to search for executables")
	podRange := flag.String("pod-range", "10.233.42.1/24", "Pod IP range to use")
	serviceRange := flag.String("service-range", "10.233.43.1/24", "Service IP range to use")
	sudoMethod := flag.String("sudo", "/usr/bin/pkexec", "Sudo tool to use")
	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}
	var err error
	a.BaseDir, err = homedir.Expand(*root)
	if err != nil {
		log.WithError(err).WithField("root", *root).Fatal("Couldn't expand root directory")
	}
	a.ExtraBinDir, err = homedir.Expand(*extraBinDir)
	if err != nil {
		log.WithError(err).WithField("extraBinDir", *extraBinDir).Fatal("Couldn't expand extraBin directory")
	}

	var serviceRangeIP net.IP
	var bindAddr net.IP
	a.PodRangeNet, a.ServiceRangeNet, a.ClusterIPRange, bindAddr, serviceRangeIP, err = CalculateIPRanges(*podRange, *serviceRange)
	if err != nil {
		log.Fatal("IP calculation returned error, aborting now!")
	}
	dnsIP := net.IPv4(0, 0, 0, 0)
	copy(dnsIP, serviceRangeIP)
	dnsIP[15]++

	file, err := os.Stat(*sudoMethod)
	if err != nil || !file.Mode().IsRegular() {
		log.WithError(err).WithField("sudo", *sudoMethod).Fatal("Sudo method is not a regular file!")
	}

	baseExecEnv := handlers.ExecutionEnvironment{}
	baseExecEnv.ListenAddress = bindAddr
	baseExecEnv.ServiceAddress = serviceRangeIP
	baseExecEnv.DNSAddress = dnsIP
	baseExecEnv.SudoMethod = *sudoMethod
	baseExecEnv.InitPorts(7000)
	return &baseExecEnv
}
