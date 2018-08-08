package etcd

import (
	"os/exec"
	"testing"
)

func TestEtcdStartup(t *testing.T) {
	done := false
	exitHandler := func(success bool, exitError *exec.ExitError) {
		if !done {
			t.Error("etcd exit detected", exitError)
		}
	}
	handler, _, _, err := StartETCDForTest(exitHandler)
	if err != nil {
		t.Error("Test failed:", err)
	}
	done = true
	handler.Stop()
}
