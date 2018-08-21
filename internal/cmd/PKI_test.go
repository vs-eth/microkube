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
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// checkFilesExist checks whether a list of 'files' exist
func checkFilesExist(files []string, t *testing.T) {
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

// TestEnsureFullPKI checks whether EnsureFullPKI works correctly
func TestEnsureFullPKI(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	directory, err := ioutil.TempDir("", "microkube-helper-unittests")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	err = os.Remove(directory)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	// Test initial creation, should fail due to missing directory
	_, _, _, err = EnsureFullPKI(directory, "testpki", false, false, []string{"127.0.0.1"})
	if err == nil {
		t.Fatal("Expected error missing!")
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("Unexpected error: %s", err)
	}

	os.Mkdir(directory, 0777)

	// Test initial creation
	ca, server, client, err := EnsureFullPKI(directory, "testpki", false, false, []string{"127.0.0.1"})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	// EnsureFullPKI uses the CertManager class to generate the certificates, which is tested separately
	// We therefore assume that the individual certificates are generated correctly
	files_initial := []string{
		ca.CertPath,
		ca.KeyPath,
		server.CertPath,
		server.KeyPath,
		client.CertPath,
		client.KeyPath,
	}
	checkFilesExist(files_initial, t)

	// Test reload
	ca, server, client, err = EnsureFullPKI(directory, "testpki", false, false, []string{"127.0.0.1"})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	files_reload := []string{
		ca.CertPath,
		ca.KeyPath,
		server.CertPath,
		server.KeyPath,
		client.CertPath,
		client.KeyPath,
	}
	checkFilesExist(files_reload, t)

	for idx, _ := range files_initial {
		if files_initial[idx] != files_reload[idx] {
			t.Fatalf("Files didn't match: '%s' vs '%s'", files_initial[idx], files_reload[idx])
		}
	}

	ca, server, client, err = EnsureFullPKI(directory, "testpki2", true, true, []string{"127.0.0.1"})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	files_special_ca := []string{
		ca.CertPath,
		ca.KeyPath,
		server.CertPath,
		server.KeyPath,
		client.CertPath,
		client.KeyPath,
	}
	checkFilesExist(files_special_ca, t)
}

func TestEnsureSigningCert(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	directory, err := ioutil.TempDir("", "microkube-helper-unittests")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	err = os.Remove(directory)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	// Test initial creation, should fail due to missing directory
	_, err = EnsureSigningCert(directory, "testpki3")
	if err == nil {
		t.Fatal("Expected error missing!")
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("Unexpected error: %s", err)
	}

	os.Mkdir(directory, 0777)

	// Test initial creation
	cert, err := EnsureSigningCert(directory, "testpki3")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	// EnsureFullPKI uses the CertManager class to generate the certificates, which is tested separately
	// We therefore assume that the individual certificates are generated correctly
	files_initial := []string{
		cert.CertPath,
		cert.KeyPath,
	}
	checkFilesExist(files_initial, t)

	// Test reload
	cert, err = EnsureSigningCert(directory, "testpki3")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	files_reload := []string{
		cert.CertPath,
		cert.KeyPath,
	}
	checkFilesExist(files_reload, t)

	for idx, _ := range files_initial {
		if files_initial[idx] != files_reload[idx] {
			t.Fatalf("Files didn't match: '%s' vs '%s'", files_initial[idx], files_reload[idx])
		}
	}
}
