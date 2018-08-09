package kube_apiserver

import (
	"github.com/pkg/errors"
	"os/exec"
	"testing"
)

func TestAPIServerStartup(t *testing.T) {
	done := false
	exitHandler := func(success bool, exitError *exec.ExitError) {
		if !done {
			panic(errors.Wrap(exitError, "unexpected exit detected"))
		}
	}
	uut, etcd, _, _, err := StartKubeAPIServerForTest(exitHandler)
	if err != nil {
		t.Error("kube apiserver didn't start", err)
		return
	}
	done = true
	uut.Stop()
	etcd.Stop()
}
