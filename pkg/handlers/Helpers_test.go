package handlers

import (
	"os/exec"
	"strings"
	"testing"
)

func TestEchoInvocation(t *testing.T) {
	exitWaiter := make(chan bool)
	exitHandler := func(rc bool, error *exec.ExitError) {
		exitWaiter <- rc
	}
	handler := NewHandler("/bin/bash", []string{
		"-c",
		"echo test",
	}, exitHandler, nil, nil)
	handler.Start()
	rc := <-exitWaiter
	if !rc {
		t.Error("Couldn't execute echo!")
	}
}

func TestEcho(t *testing.T) {
	exitWaiter := make(chan bool)
	exitStdout := make(chan string, 1)
	exitHandler := func(rc bool, error *exec.ExitError) {
		exitWaiter <- rc
	}
	stdoutHandler := func(value []byte) {
		exitStdout <- string(value)
	}
	handler := NewHandler("/bin/bash", []string{
		"-c",
		"echo test",
	}, exitHandler, stdoutHandler, nil)
	handler.Start()
	rc := <-exitWaiter
	if !rc {
		t.Error("Couldn't execute echo!")
	}
	str := <-exitStdout
	if strings.Trim(str, " \t\r\n") != "test" {
		t.Error("Unexpected stdout: '", str, "'")
	}
}
