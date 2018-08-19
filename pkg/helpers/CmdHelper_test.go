/*
 * Copyright 2018 The microkube authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package helpers

import (
	"os/exec"
	"strings"
	"testing"
	"os"
)

// TestInvalidInvocation tests the invocation of a non-existent program
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

// TestEchoInvocation tests running echo
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

// TestEcho tests running echo and comparing it's output
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

// TestAllBinariesPresent tries to find all binaries required during tests
func TestAllBinariesPresent(t *testing.T) {
	binaries := []string{
		"etcd",
		"hyperkube",
	}
	for _, item := range binaries {
		path, err := FindBinary(item, "", "")
		if err != nil {
			t.Fatal("Didn't find " + item)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal("Coudln't stat " + item)
		}
		if !info.Mode().IsRegular() {
			t.Fatal(item + "isn't a regular file")
		}
	}
}