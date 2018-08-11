package helpers

import (
	"os"
	"testing"
)

func TestAllBinariesPresent(t *testing.T) {
	binaries := []string{
		"kubelet",
		"kubectl",
		"etcd",
		"kube-apiserver",
		"kube-controller-manager",
		"kube-apiserver",
		"kube-proxy",
		"kube-scheduler",
	}
	for _, item := range binaries {
		path, err := FindBinary(item, "")
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
