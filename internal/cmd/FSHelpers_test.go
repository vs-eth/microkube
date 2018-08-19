package cmd

import (
	"testing"
	"os"
	"path"
	"io/ioutil"
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
}