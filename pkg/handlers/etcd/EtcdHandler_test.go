package etcd

import (
	"github.com/uubk/microkube/pkg/helpers"
	"os/exec"
	"testing"
)

// Test whether etcd actually starts correctly
func TestEtcdStartup(t *testing.T) {
	done := false
	exitHandler := func(success bool, exitError *exec.ExitError) {
		if !done {
			t.Fatal("etcd exit detected", exitError)
		}
	}
	handler, _, _, _, err := helpers.StartHandlerForTest("etcd", EtcdHandlerConstructor, exitHandler, false, 1)
	if err != nil {
		t.Fatal("Test failed:", err)
		return
	}
	done = true
	for _, item := range handler {
		item.Stop()
	}
}
