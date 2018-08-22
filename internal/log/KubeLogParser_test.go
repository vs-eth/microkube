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

package log

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"testing"
)

// TestWarningMessage tests a single warning message
func TestWarningMessage(t *testing.T) {
	var buffer bytes.Buffer
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(&buffer)
	logrus.SetFormatter(&logrus.JSONFormatter{
		DisableTimestamp: true,
	})
	testStr := "W0812 17:00:08.194751   25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.\n"
	uut := NewKubeLogParser("testkubeapp")
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	result := buffer.String()
	if result != "{\"app\":\"testkubeapp\",\"level\":\"warning\",\"location\":\"genericapiserver.go:319\",\"msg\":\"Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.\"}\n" {
		t.Fatalf("Unexpected output: %s", result)
	}
}

// TestWarningMessage tests a single 'restful' info message
func TestRestfulMessage(t *testing.T) {
	var buffer bytes.Buffer
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(&buffer)
	logrus.SetFormatter(&logrus.JSONFormatter{
		DisableTimestamp: true,
	})
	testStr := "[restful] 2018/08/12 17:00:09 log.go:33: [restful/swagger] listing is available at https://172.17.0.1:7443/swaggerapi\n"
	uut := NewKubeLogParser("testkubeapp")
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	result := buffer.String()
	if result != "{\"app\":\"testkubeapp\",\"component\":\"restful\",\"level\":\"info\",\"location\":\"log.go:33\",\"msg\":\"listing is available at https://172.17.0.1:7443/swaggerapi\"}\n" {
		t.Fatalf("Unexpected output: %s", result)
	}
}

// TestKubeMessageTypes tests all kube message types
func TestKubeMessageTypes(t *testing.T) {
	var buffer bytes.Buffer
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(&buffer)
	logrus.SetFormatter(&logrus.JSONFormatter{
		DisableTimestamp: true,
	})
	testStr := `I0812 17:00:08.194751   25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.
E0812 17:00:08.194751   25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.
W0812 17:00:08.194751   25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.
D0812 17:00:08.194751   25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.
N0812 17:00:08.194751   25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.
S0812 17:00:08.194751   25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.
`
	uut := NewKubeLogParser("testkubeapp")
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	result := buffer.String()
	if result != `{"app":"testkubeapp","level":"info","location":"genericapiserver.go:319","msg":"Skipping API scheduling.k8s.io/v1alpha1 because it has no resources."}
{"app":"testkubeapp","level":"error","location":"genericapiserver.go:319","msg":"Skipping API scheduling.k8s.io/v1alpha1 because it has no resources."}
{"app":"testkubeapp","level":"warning","location":"genericapiserver.go:319","msg":"Skipping API scheduling.k8s.io/v1alpha1 because it has no resources."}
{"app":"testkubeapp","level":"debug","location":"genericapiserver.go:319","msg":"Skipping API scheduling.k8s.io/v1alpha1 because it has no resources."}
{"app":"testkubeapp","level":"info","location":"genericapiserver.go:319","msg":"Skipping API scheduling.k8s.io/v1alpha1 because it has no resources."}
{"app":"testkubeapp","level":"error","location":"genericapiserver.go:319","msg":"Skipping API scheduling.k8s.io/v1alpha1 because it has no resources."}
` {
		t.Fatalf("Unexpected output: %s", result)
	}
}

// TestInvalidKubeMessageType tests an invalid kube log message type
func TestInvalidKubeMessageType(t *testing.T) {
	var buffer bytes.Buffer
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(&buffer)
	logrus.SetFormatter(&logrus.JSONFormatter{
		DisableTimestamp: true,
	})
	testStr := "X0812 17:00:08.194751   25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.\n"
	uut := NewKubeLogParser("testkubeapp")
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	result := buffer.String()
	if result != `{"app":"microkube","component":"KubeLogParser","fields.level":88,"level":"warning","msg":"Unknown severity level in kube log parser"}
{"app":"testkubeapp","level":"warning","msg":"X0812 17:00:08.194751 25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.\n"}
` {
		t.Fatalf("Unexpected output: %s", result)
	}
}
