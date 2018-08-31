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

package cmd

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

// TestBasicArgParse checks whether 'normal' arg parsing works successfully
func TestBasicArgParse(t *testing.T) {
	flag.CommandLine.Args()
	uut := NewArgHandler(true)
	uut.setupArgs()

	args := []string{
		"-root",
		"/tmp",
		"-extra-bin-dir",
		"/tmp/bin",
		"-verbose",
		"true",
	}
	flag.CommandLine.Parse(args)

	execEnv := uut.evalArgs()
	assert.Equal(t, "/tmp", uut.BaseDir, "Unexpected base dir value")
	assert.Equal(t, "/tmp/bin", uut.ExtraBinDir, "Unexpected extra bin dir value")
	assert.Equal(t, true, gs.verbose, "Unexpected verbosity value")
	assert.Equal(t, net.IPv4(10, 233, 43, 2), execEnv.DNSAddress, "Unexpected dns address")
	assert.Equal(t, net.IPv4(10, 233, 43, 1), execEnv.ServiceAddress, "Unexpected service address")
}

// TestAllArgParse checks whether arg parsing with all arguments works successfully
func TestAllArgParse(t *testing.T) {
	uut := NewArgHandler(true)
	uut.setupArgs()

	args := []string{
		"-root",
		"/tmp",
		"-extra-bin-dir",
		"/tmp/bin",
		"-pod-range",
		"192.168.10.1/24",
		"-service-range",
		"192.168.11.1/24",
		"-verbose",
		"1",
	}
	flag.CommandLine.Parse(args)

	execEnv := uut.evalArgs()
	assert.Equal(t, "/tmp", uut.BaseDir, "Unexpected base dir value")
	assert.Equal(t, "/tmp/bin", uut.ExtraBinDir, "Unexpected extra bin dir value")
	assert.Equal(t, true, gs.verbose, "Unexpected verbosity value")
	assert.Equal(t, net.IPv4(192, 168, 11, 2), execEnv.DNSAddress, "Unexpected dns address")
	assert.Equal(t, net.IPv4(192, 168, 11, 1), execEnv.ServiceAddress, "Unexpected service address")
}
