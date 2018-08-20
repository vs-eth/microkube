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
	"io/ioutil"
	"os"
	"testing"
)

// TestCertHelper checks whether the CertHelper method creates all certificates
func TestCertHelper(t *testing.T) {
	directory, err := ioutil.TempDir("", "microkube-helper-unittests")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	ca, server, client, err := CertHelper(directory, "foobar")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	// CertHelper uses the CertManager class to generate the certificates, which is tested separately
	// We therefore assume that the individual certificates are generated correctly

	files := []string{
		ca.CertPath,
		ca.KeyPath,
		server.CertPath,
		server.KeyPath,
		client.CertPath,
		client.KeyPath,
	}

	for _, file := range files {
		status, err := os.Stat(file)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
		if !status.Mode().IsRegular() {
			t.Fatalf("Expected '%s' to be a regular file!", file)
		}
		if status.Size() < 512 {
			t.Fatalf("Expected '%s' to be at least 512 bytes!", file)
		}
	}
}
