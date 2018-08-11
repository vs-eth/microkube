package controller_manager

import (
	"testing"
	"os/exec"
	"github.com/uubk/microkube/pkg/helpers"
)

func TestControllerManagerStartup(t *testing.T) {
	done := false
	exitHandler := func(success bool, exitError *exec.ExitError) {
		if !done {
			t.Fatal("exit detected", exitError)
		}
	}
	handler, _, _, _, err := helpers.StartHandlerForTest("kube-controller-manager", KubeControllerManagerConstructor, exitHandler, true, 30)
	if err != nil {
		t.Fatal("Test failed:", err)
		return
	}
	done = true
	for _, item := range handler {
		item.Stop()
	}
}