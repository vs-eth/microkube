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

import "testing"

// TestStandardIPRanges tests whether parsing the default IP ranges works
func TestStandardIPRanges(t *testing.T) {
	uut := Microkubed{}
	err := uut.calculateIPRanges("10.233.42.1/24", "10.233.43.1/24")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if uut.podRangeNet.String() != "10.233.42.0/24" {
		t.Fatalf("Pod range was parsed incorrectly %s", uut.podRangeNet.String())
	}
	if uut.serviceRangeNet.String() != "10.233.43.0/24" {
		t.Fatalf("Service range was parsed incorrectly %s", uut.serviceRangeNet.String())
	}
	if uut.serviceRangeIP.String() != "10.233.43.1" {
		t.Fatalf("Service IP was parsed incorrectly %s", uut.serviceRangeIP.String())
	}
	if uut.clusterIPRange.String() != "10.233.42.1/23" {
		t.Fatalf("Cluster IP range was parsed incorrectly %s", uut.clusterIPRange.String())
	}
}

// TestDiscontinousIPRanges tests whether parsing two discontinous IP ranges works and results in a huge clusternet
func TestDiscontinousIPRanges(t *testing.T) {
	uut := Microkubed{}
	err := uut.calculateIPRanges("192.168.1.1/24", "192.168.15.1/24")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if uut.podRangeNet.String() != "192.168.1.0/24" {
		t.Fatalf("Pod range was parsed incorrectly %s", uut.podRangeNet.String())
	}
	if uut.serviceRangeNet.String() != "192.168.15.0/24" {
		t.Fatalf("Service range was parsed incorrectly %s", uut.serviceRangeNet.String())
	}
	if uut.serviceRangeIP.String() != "192.168.15.1" {
		t.Fatalf("Service IP was parsed incorrectly %s", uut.serviceRangeIP.String())
	}
	if uut.clusterIPRange.String() != "192.168.1.1/20" {
		t.Fatalf("Cluster IP range was parsed incorrectly %s", uut.clusterIPRange.String())
	}
}

// TestIPParseError tests whether parsing invalid IP ranges returns the correct error codes
func TestIPParseError(t *testing.T) {
	uut := Microkubed{}
	err := uut.calculateIPRanges("192.168.1.1/33", "foobar")
	if err == nil {
		t.Fatalf("Expected error missing")
	}
	if err.Error() != "invalid CIDR address: 192.168.1.1/33" {
		t.Fatalf("Unexpected error: %s", err)
	}

	err = uut.calculateIPRanges("192.168.1.1/31", "foobar")
	if err == nil {
		t.Fatalf("Expected error missing")
	}
	if err.Error() != "invalid CIDR address: foobar" {
		t.Fatalf("Unexpected error: %s", err)
	}

}