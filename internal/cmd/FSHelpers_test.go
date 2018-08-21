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
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
)

// Test whether EnsureDir works
func TestEnsureDir(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "TestEnsureDir")
	if err != nil {
		t.Fatal("tempDir creation failed", err)
	}
	err = os.Remove(tempDir)
	if err != nil {
		t.Fatal("tempDir remove failed", err)
	}
	err = EnsureDir(tempDir, "", 0770)
	if err != nil {
		t.Fatal("ensure dir failed", err)
	}
	err = EnsureDir(tempDir, "abc", 0770)
	if err != nil {
		t.Fatal("ensure dir failed", err)
	}
	info, err := os.Stat(path.Join(tempDir, "abc"))
	if err != nil {
		t.Fatal("dir did not exist", err)
	}
	if !info.IsDir() {
		t.Fatal("dir is not a directory")
	}

	err = EnsureDir(tempDir, "a\0000b", 0770)
	if err == nil {
		t.Fatal("Expected error missing")
	}
	if !strings.Contains(err.Error(), "invalid argument") {
		t.Fatalf("Unexpected error: %s", err)
	}
}
