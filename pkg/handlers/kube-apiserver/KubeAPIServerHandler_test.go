package kube_apiserver

import (
	"bufio"
	"bytes"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/helpers"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"
	"testing"
)

func TestAPIServerStartup(t *testing.T) {
	done := false
	exitHandler := func(success bool, exitError *exec.ExitError) {
		if !done {
			t.Fatal("exit detected", exitError)
		}
	}
	handler, _, _, _, err := helpers.StartHandlerForTest("kube-apiserver", KubeApiServerConstructor, exitHandler, true, 30)
	if err != nil {
		t.Fatal("Test failed:", err)
		return
	}
	done = true
	for _, item := range handler {
		item.Stop()
	}
}

func TestAPIServerKubeconfig(t *testing.T) {
	done := false
	exitHandler := func(success bool, exitError *exec.ExitError) {
		if !done {
			t.Fatal("exit detected", exitError)
		}
	}
	handlers, ca, client, _, err := helpers.StartHandlerForTest("kube-apiserver", KubeApiServerConstructor, exitHandler, false, 30)
	if err != nil {
		t.Fatal("Test failed:", err)
		return
	}
	defer func() {
		done = true
		for _, item := range handlers {
			item.Stop()
		}
	}()
	// Kube-Apiserver running

	// Generate kubeconfig so that we can use kubectl
	tmpdir, err := ioutil.TempDir("", "microkube-unittests-kubeconfig")
	if err != nil {
		errors.Wrap(err, "tempdir creation failed")
	}
	kubeconfig := path.Join(tmpdir, "kubeconfig")
	err = CreateClientKubeconfig(ca, client, kubeconfig, "127.0.0.1")
	if err != nil {
		t.Error("kubeconfig creation failed", err)
		return
	}

	bin, err := helpers.FindBinary("kubectl", "")
	if err != nil {
		t.Error("kubectl not found", err)
		return
	}

	var buf bytes.Buffer
	outputHandler := func(output []byte) {
		buf.Write(output)
	}

	// Start kubectl and read the cluster's version. (Successfully reading the server version requires an API call.)
	exitWaiter := make(chan bool)
	kubeCtlExitHandler := func(success bool, exitError *exec.ExitError) {
		exitWaiter <- success
	}
	handler := helpers.NewCmdHandler(bin, []string{
		"--kubeconfig",
		kubeconfig,
		"version",
	}, kubeCtlExitHandler, outputHandler, outputHandler)
	err = handler.Start()
	if err != nil {
		t.Error("Couldn't start program", err)
		return
	}
	rc := <-exitWaiter
	if !rc {
		t.Error("Couldn't execute program!", err)
	}

	checkScanner := bufio.NewScanner(strings.NewReader(string(buf.String())))
	for checkScanner.Scan() {
		line := checkScanner.Text()
		if !(strings.HasPrefix(line, "Client Version: version.Info{") ||
			strings.HasPrefix(line, "Server Version: version.Info{")) {
			t.Error("Unexpected version output: " + line)
		}
	}
}