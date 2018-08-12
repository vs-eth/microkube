package helpers

import (
	"os/exec"
	"strings"
	"testing"
)

func TestInvalidInvocation(t *testing.T) {
	exitWaiter := make(chan bool)
	exitHandler := func(rc bool, error *exec.ExitError) {
		exitWaiter <- rc
	}
	handler := NewCmdHandler("/bin/FooBarBazBash", []string{
		"-c",
		"echo test",
	}, exitHandler, nil, nil)
	err := handler.Start()
	if err == nil {
		t.Error("Invalid command executed?")
		return
	}
}

func TestEchoInvocation(t *testing.T) {
	exitWaiter := make(chan bool)
	exitHandler := func(rc bool, error *exec.ExitError) {
		exitWaiter <- rc
	}
	handler := NewCmdHandler("/bin/bash", []string{
		"-c",
		"echo test",
	}, exitHandler, nil, nil)
	err := handler.Start()
	if err != nil {
		t.Error("Coudln't start program")
		return
	}
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
	handler := NewCmdHandler("/bin/bash", []string{
		"-c",
		"echo test",
	}, exitHandler, stdoutHandler, nil)
	err := handler.Start()
	if err != nil {
		t.Error("Coudln't start program")
		return
	}
	rc := <-exitWaiter
	if !rc {
		t.Error("Couldn't execute echo!")
	}
	str := <-exitStdout
	if strings.Trim(str, " \t\r\n") != "test" {
		t.Error("Unexpected stdout: '", str, "'")
	}
}