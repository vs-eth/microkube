package etcd

import (
	"github.com/pkg/errors"
	"os/exec"
	"testing"
)

func TestEtcdStartup(t *testing.T) {
	done := false
	exitHandler := func(success bool, exitError *exec.ExitError) {
		if !done {
			panic(errors.Wrap(exitError, "etcd exit detected"))
		}
	}
	handler, _, _, err := StartETCDForTest(exitHandler)
	if err != nil {
		t.Error("Test failed:", err)
		return
	}
	done = true
	handler.Stop()
}
