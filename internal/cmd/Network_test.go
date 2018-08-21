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
	"github.com/sirupsen/logrus"
	"net"
	"testing"
)

// ipArrForStringArr converts an array of strings to an array of IPs
func ipArrForStringArr(candidates_str []string) ([]net.IP, net.IP) {
	var candidates []net.IP
	var lastIP net.IP
	for _, addr := range candidates_str {
		lastIP = net.ParseIP(addr)
		candidates = append(candidates, lastIP)
	}
	return candidates, lastIP
}

// TestFindBindAddress tests whether FindBindAddress returns a valid address
func TestFindBindAddress(t *testing.T) {
	addr := FindBindAddress()
	if addr.IsLoopback() || addr.IsMulticast() || addr.IsUnspecified() {
		t.Fatal("Invalid address returned")
	}
}

// TestAddressSelection tests whether findBindAddress selects the correct address from a list of candidates
func TestAddressSelection(t *testing.T) {
	candidates_str := []string{
		"100.64.1.1",
		"100.65.1.1",
		"192.168.10.1",
	}
	candidates, lastIP := ipArrForStringArr(candidates_str)

	addr := findBindAddress(candidates)
	if addr.String() != lastIP.String() {
		t.Fatal("Invalid address returned!")
	}

	candidates_str = []string{
		"127.0.0.1",
		"192.168.10.1",
	}
	candidates, lastIP = ipArrForStringArr(candidates_str)

	addr = findBindAddress(candidates)
	if addr.String() != lastIP.String() {
		t.Fatal("Invalid address returned!")
	}

	candidates_str = []string{
		"10.30.0.72",
		"172.17.0.1",
	}
	candidates, _ = ipArrForStringArr(candidates_str)

	addr = findBindAddress(candidates)
	if addr.String() != candidates[0].String() {
		t.Fatal("Invalid address returned!")
	}
}

// TestAddressFallback tests whether findBindAddress correctly falls back to a public IPv4 if no private one is found
func TestAddressFallback(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	candidates_str := []string{
		"100.64.1.1",
		"100.65.1.1",
	}
	candidates, _ := ipArrForStringArr(candidates_str)

	addr := findBindAddress(candidates)
	if addr.String() != candidates[0].String() {
		t.Fatal("Invalid address returned!")
	}
}
