package kube

import "testing"

func TestWarningMessage(t *testing.T) {
	testStr := "W0812 17:00:08.194751   25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.\n"
	uut := NewKubeLogParser("testkubeapp")
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

func TestRestfulMessage(t *testing.T) {
	testStr := "[restful] 2018/08/12 17:00:09 log.go:33: [restful/swagger] listing is available at https://172.17.0.1:7443/swaggerapi\n"
	uut := NewKubeLogParser("testkubeapp")
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}
