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
func ipArrForStringArr(candidatesStr []string) ([]net.IP, net.IP) {
	var candidates []net.IP
	var lastIP net.IP
	for _, addr := range candidatesStr {
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
	candidatesStr := []string{
		"100.64.1.1",
		"100.65.1.1",
		"192.168.10.1",
	}
	candidates, lastIP := ipArrForStringArr(candidatesStr)

	addr := findBindAddress(candidates)
	if addr.String() != lastIP.String() {
		t.Fatal("Invalid address returned!")
	}

	candidatesStr = []string{
		"127.0.0.1",
		"192.168.10.1",
	}
	candidates, lastIP = ipArrForStringArr(candidatesStr)

	addr = findBindAddress(candidates)
	if addr.String() != lastIP.String() {
		t.Fatal("Invalid address returned!")
	}

	candidatesStr = []string{
		"10.30.0.72",
		"172.17.0.1",
	}
	candidates, _ = ipArrForStringArr(candidatesStr)

	addr = findBindAddress(candidates)
	if addr.String() != candidates[0].String() {
		t.Fatal("Invalid address returned!")
	}
}

// TestAddressFallback tests whether findBindAddress correctly falls back to a public IPv4 if no private one is found
func TestAddressFallback(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	candidatesStr := []string{
		"100.64.1.1",
		"100.65.1.1",
	}
	candidates, _ := ipArrForStringArr(candidatesStr)

	addr := findBindAddress(candidates)
	if addr.String() != candidates[0].String() {
		t.Fatal("Invalid address returned!")
	}
}

// TestStandardIPRanges tests whether parsing the default IP ranges works
func TestStandardIPRanges(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	podRangeNet, serviceRangeNet, clusterIPRange, _, serviceRangeIP, err := CalculateIPRanges("10.233.42.1/24",
		"10.233.43.1/24")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if podRangeNet.String() != "10.233.42.0/24" {
		t.Fatalf("Pod range was parsed incorrectly %s", podRangeNet.String())
	}
	if serviceRangeNet.String() != "10.233.43.0/24" {
		t.Fatalf("Service range was parsed incorrectly %s", serviceRangeNet.String())
	}
	if serviceRangeIP.String() != "10.233.43.1" {
		t.Fatalf("Service IP was parsed incorrectly %s", serviceRangeIP.String())
	}
	if clusterIPRange.String() != "10.233.42.1/23" {
		t.Fatalf("Cluster IP range was parsed incorrectly %s", clusterIPRange.String())
	}
}

// TestDiscontinousIPRanges tests whether parsing two discontinous IP ranges works and results in a huge clusternet
func TestDiscontinousIPRanges(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	podRangeNet, serviceRangeNet, clusterIPRange, _, serviceRangeIP, err := CalculateIPRanges("192.168.1.1/24",
		"192.168.15.1/24")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if podRangeNet.String() != "192.168.1.0/24" {
		t.Fatalf("Pod range was parsed incorrectly %s", podRangeNet.String())
	}
	if serviceRangeNet.String() != "192.168.15.0/24" {
		t.Fatalf("Service range was parsed incorrectly %s", serviceRangeNet.String())
	}
	if serviceRangeIP.String() != "192.168.15.1" {
		t.Fatalf("Service IP was parsed incorrectly %s", serviceRangeIP.String())
	}
	if clusterIPRange.String() != "192.168.1.1/20" {
		t.Fatalf("Cluster IP range was parsed incorrectly %s", clusterIPRange.String())
	}
}

// TestIPParseError tests whether parsing invalid IP ranges returns the correct error codes
func TestIPParseError(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	_, _, _, _, _, err := CalculateIPRanges("192.168.1.1/33", "foobar")
	if err == nil {
		t.Fatalf("Expected error missing")
	}
	if err.Error() != "invalid CIDR address: 192.168.1.1/33" {
		t.Fatalf("Unexpected error: %s", err)
	}

	_, _, _, _, _, err = CalculateIPRanges("192.168.1.1/31", "foobar")
	if err == nil {
		t.Fatalf("Expected error missing")
	}
	if err.Error() != "invalid CIDR address: foobar" {
		t.Fatalf("Unexpected error: %s", err)
	}
}
