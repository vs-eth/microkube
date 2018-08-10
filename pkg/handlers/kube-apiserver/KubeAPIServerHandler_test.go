package kube_apiserver

import (
	"bufio"
	"bytes"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers/etcd"
	"github.com/uubk/microkube/pkg/helpers"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"
)

func TestAPIServerStartup(t *testing.T) {
	done := false
	exitHandler := func(success bool, exitError *exec.ExitError) {
		if !done {
			panic(errors.Wrap(exitError, "unexpected exit detected"))
		}
	}
	uut, etcdRef, _, _, err := StartKubeAPIServerForTest(exitHandler)
	if err != nil {
		t.Error("kube apiserver didn't start", err)
		return
	}
	done = true
	uut.Stop()
	etcdRef.Stop()
}

func TestAPIServerKubeconfig(t *testing.T) {
	done := false
	exitHandler := func(success bool, exitError *exec.ExitError) {
		if !done {
			panic(errors.Wrap(exitError, "unexpected exit detected"))
		}
	}
	uut, etcdRef, ca, client, err := StartKubeAPIServerForTest(exitHandler)
	if err != nil {
		t.Error("kube apiserver didn't start", err)
		return
	}
	defer etcdRef.Stop()
	defer uut.Stop()

	tmpdir, err := ioutil.TempDir("", "microkube-unittests-config")
	if err != nil {
		t.Error("tempdir creation failed", err)
		return
	}

	kubeconfig := path.Join(tmpdir, "kubeconfig")
	err = CreateClientKubeconfig(ca, client, kubeconfig, "127.0.0.1")
	if err != nil {
		t.Error("kubeconfig creation failed", err)
		return
	}

	bin, err := etcd.GetBinary("kubectl")
	if err != nil {
		t.Error("kubectl not found", err)
		return
	}

	var buf bytes.Buffer
	outputHandler := func(output []byte) {
		buf.Write(output)
	}

	time.Sleep(2 * time.Second)

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

	done = true
}
