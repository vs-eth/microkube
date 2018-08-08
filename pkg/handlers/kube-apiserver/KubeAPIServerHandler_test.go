package kube_apiserver

import (
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/handlers/etcd"
	"os/exec"
	"testing"
	"io/ioutil"
	"fmt"
)

func TestAPIServerStartup(t *testing.T) {
	done := false
	exitHandler := func(success bool, exitError *exec.ExitError) {
		if !done {
			t.Error("etcd exit detected", exitError)
		}
	}
	etcdHandler, etcdCA, etcdClientCert, err := etcd.StartETCDForTest(exitHandler)
	if err != nil {
		t.Error("ETCD startup failed:", err)
		return
	}
	defer etcdHandler.Stop()

	tmpdir, err := ioutil.TempDir("", "microkube-unittests-kubeapi")
	if err != nil {
		t.Error("tempdir creation failed", err)
		return
	}

	kubeCA, kubeServer, kubeClient, err := handlers.CertHelper(tmpdir, "kubeapi-unittest")
	if err != nil {
		t.Error("kube CA setup failed:", err)
		return
	}

	bin, err := etcd.GetBinary("kube-apiserver")
	if err != nil {
		t.Error("kube apiserver binary not found", err)
		return
	}

	outputHandler := func(output []byte) {
		fmt.Println("kube   |", string(output))
		// Nop
	}

	uut := NewHandler(bin, kubeServer.CertPath, kubeServer.KeyPath, kubeClient.CertPath, kubeClient.KeyPath,
		kubeCA.CertPath, etcdClientCert.CertPath, etcdClientCert.KeyPath, etcdCA.CertPath, outputHandler, exitHandler)
	err = uut.Start()
	if err != nil {
		t.Error("kube apiserver didn't launch", err)
		return
	}
	defer uut.Stop()

	msgChan := make(chan handlers.HealthMessage, 1)
	uut.EnableHealthChecks(kubeCA, kubeClient, msgChan, false)
	msg := <- msgChan

	if !msg.IsHealthy {
		t.Error("kube apiserver health check failed", msg.Error)
	}
	done = true
}
