package controller_manager

import (
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/handlers/kube-apiserver"
	"github.com/uubk/microkube/pkg/helpers"
	"github.com/uubk/microkube/pkg/pki"
	"io"
	"io/ioutil"
	"path"
	"strings"
)

type ControllerManagerHandler struct {
	handlers.BaseServiceHandler
	cmd *helpers.CmdHandler

	// Path to kube-controller-manager binary
	binary string
	// Path to kube server certificate
	kubeServerCert string
	// Path to kube server certificate's key
	kubeServerKey string
	// Path to kube cluster CA certificate
	kubeClusterCACert string
	// Path to kube cluster CA certificate key
	kubeClusterCAKey string
	// Path to a key used to sign service account tokens
	kubeSvcKey string
	// IP range for pods (CIDR)
	podRange string
	// Path to kubeconfig
	kubeconfig string
	// Address to bind on
	bindAddress string
	// Output handler
	out handlers.OutputHander
}

func NewControllerManagerHandler(binary, kubeconfig, listenAddress string, server, client, ca, clusterCA, svcAcctCert *pki.RSACertificate, podRange string, out handlers.OutputHander, exit handlers.ExitHandler) *ControllerManagerHandler {
	obj := &ControllerManagerHandler{
		binary:            binary,
		kubeServerCert:    server.CertPath,
		kubeServerKey:     server.KeyPath,
		cmd:               nil,
		out:               out,
		kubeconfig:        kubeconfig,
		bindAddress:       listenAddress,
		kubeClusterCACert: clusterCA.CertPath,
		kubeClusterCAKey:  clusterCA.KeyPath,
		podRange:          podRange,
		kubeSvcKey:svcAcctCert.KeyPath,
	}

	obj.BaseServiceHandler = *handlers.NewHandler(exit, obj.healthCheckFun, "https://"+listenAddress+":7000/healthz",
		obj.stop, obj.Start, ca, client)
	return obj
}

func (handler *ControllerManagerHandler) stop() {
	if handler.cmd != nil {
		handler.cmd.Stop()
	}
}

func (handler *ControllerManagerHandler) Start() error {
	handler.cmd = helpers.NewCmdHandler(handler.binary, []string{
		"--allocate-node-cidrs",
		"--cluster-cidr",
		handler.podRange,
		"--bind-address",
		handler.bindAddress,
		"--cluster-name",
		"microkube",
		"--cluster-signing-cert-file",
		handler.kubeClusterCACert,
		"--cluster-signing-key-file",
		handler.kubeClusterCAKey,
		"--enable-hostpath-provisioner",
		"--secure-port",
		"7000",
		"--kubeconfig",
		handler.kubeconfig,
		"--tls-cert-file",
		handler.kubeServerCert,
		"--tls-private-key-file",
		handler.kubeServerKey,
		"--service-account-private-key-file",
		handler.kubeSvcKey,
	}, handler.BaseServiceHandler.HandleExit, handler.out, handler.out)
	return handler.cmd.Start()
}

func (handler *ControllerManagerHandler) healthCheckFun(responseBin *io.ReadCloser) error {
	str, err := ioutil.ReadAll(*responseBin)
	if err != nil {
		return err
	}
	if strings.Trim(string(str), " \r\n") != "ok" {
		return errors.New("Health != ok: " + string(str))
	}
	return nil
}

// This function is supposed to be only used for testing
func KubeControllerManagerConstructor(ca, server, client *pki.RSACertificate, binary, workdir string, outputHandler handlers.OutputHander, exitHandler handlers.ExitHandler) ([]handlers.ServiceHandler, error) {
	// Start apiserver (and etcd)
	handlerList, kubeCA, kubeClient, kubeServer, err := helpers.StartHandlerForTest("kube-apiserver", kube_apiserver.KubeApiServerConstructor, exitHandler, false, 30)
	if err != nil {
		return handlerList, errors.Wrap(err, "kube-apiserver startup prereq failed")
	}
	// Generate kubeconfig
	tmpdir, err := ioutil.TempDir("", "microkube-unittests-kubeconfig")
	if err != nil {
		errors.Wrap(err, "tempdir creation failed")
	}
	kubeconfig := path.Join(tmpdir, "kubeconfig")
	err = kube_apiserver.CreateClientKubeconfig(ca, client, kubeconfig, "127.0.0.1")
	if err != nil {
		return handlerList, errors.Wrap(err, "kubeconfig creation failed")
	}

	handlerList = append(handlerList, NewControllerManagerHandler(binary, kubeconfig, "127.0.0.1", kubeServer, kubeClient, kubeCA, ca, server, "127.10.11.0/24", outputHandler, exitHandler))

	return handlerList, nil
}
