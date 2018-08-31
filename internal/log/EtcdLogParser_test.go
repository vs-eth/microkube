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

// TestETCDMessageTypes tests all etcd message types
func TestETCDMessageTypes(t *testing.T) {
	var buffer bytes.Buffer
	testStr := `2018-08-12 14:13:48.437712 I | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32
2018-08-12 14:13:48.437712 E | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32
2018-08-12 14:13:48.437712 W | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32
2018-08-12 14:13:48.437712 D | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32
2018-08-12 14:13:48.437712 N | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32
2018-08-12 14:13:48.437712 C | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32
`
	uut := NewETCDLogParser()
	uut.log.SetLevel(logrus.DebugLevel)
	uut.log.SetOutput(&buffer)
	uut.log.Formatter = &logrus.JSONFormatter{
		DisableTimestamp: true,
	}
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	result := buffer.String()
	if result != `{"app":"etcd","component":"etcdserver","level":"info","msg":"published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32"}
{"app":"etcd","component":"etcdserver","level":"error","msg":"published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32"}
{"app":"etcd","component":"etcdserver","level":"warning","msg":"published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32"}
{"app":"etcd","component":"etcdserver","level":"debug","msg":"published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32"}
{"app":"etcd","component":"etcdserver","level":"info","msg":"published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32"}
{"app":"etcd","component":"etcdserver","level":"error","msg":"published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32"}
` {
		t.Fatalf("Unexpected output: %s", result)
	}
}

// TestInvalidETCDMessage tests an invalid etcd message
func TestInvalidETCDMessage(t *testing.T) {
	var buffer bytes.Buffer
	testStr := "2018-08-12 14:13:48.437712 X |\n"
	uut := NewETCDLogParser()
	uut.log.SetLevel(logrus.DebugLevel)
	uut.log.SetOutput(&buffer)
	uut.log.Formatter = &logrus.JSONFormatter{
		DisableTimestamp: true,
	}
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	result := buffer.String()
	if result != "{\"app\":\"etcd\",\"component\":\"EtcdLogParser\",\"level\":\"warning\",\"msg\":\"2018-08-12 14:13:48.437712 X |\"}\n" {
		t.Fatalf("Unexpected output: %s", result)
	}
}

// TestInvalidETCDMessageType tests an invalid etcd message type
func TestInvalidETCDMessageType(t *testing.T) {
	var buffer bytes.Buffer
	testStr := "2018-08-12 14:13:48.437712 X | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32\n"
	uut := NewETCDLogParser()
	uut.log.SetLevel(logrus.DebugLevel)
	uut.log.SetOutput(&buffer)
	uut.log.Formatter = &logrus.JSONFormatter{
		DisableTimestamp: true,
	}
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	result := buffer.String()
	if result != `{"app":"microkube","component":"EtcdLogParser","fields.level":"X","level":"warning","msg":"Unknown severity level in etcd log parser"}
{"app":"etcd","component":"etcdserver","level":"warning","msg":"published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32"}
` {
		t.Fatalf("Unexpected output: %s", result)
	}
}

// TestETCDSpamDrop tests whether some etcd log messages are dropped correctly
func TestETCDSystemdSpamDrop(t *testing.T) {
	var buffer bytes.Buffer
	testStr := `2018-08-20 14:43:34.123265 I | embed: rejected connection from "127.0.0.1:35606" (error "EOF", ServerName "")
2018-08-20 14:43:34.786265 E | etcdmain: forgot to set Type=notify in systemd service file?
`
	uut := NewETCDLogParser()
	uut.log.SetLevel(logrus.DebugLevel)
	uut.log.SetOutput(&buffer)
	uut.log.Formatter = &logrus.JSONFormatter{
		DisableTimestamp: true,
	}
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	result := buffer.String()
	if result != "" {
		t.Fatalf("Unexpected output: %s", result)
	}
}

// TestInfoMessage tests a single etcd info message
func TestInfoMessage(t *testing.T) {
	var buffer bytes.Buffer
	testStr := "2018-08-12 14:13:48.437712 I | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32\n"
	uut := NewETCDLogParser()
	uut.log.SetLevel(logrus.DebugLevel)
	uut.log.SetOutput(&buffer)
	uut.log.Formatter = &logrus.JSONFormatter{
		DisableTimestamp: true,
	}
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	result := buffer.String()
	if result != "{\"app\":\"etcd\",\"component\":\"etcdserver\",\"level\":\"info\",\"msg\":\"published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32\"}\n" {
		t.Fatalf("Unexpected output: %s", result)
	}
}

// TestInfoMessageSplit tests a single etcd info message but feeding it byte-for-byte
func TestInfoMessageSplit(t *testing.T) {
	var buffer bytes.Buffer
	testStr := "2018-08-12 14:13:48.437712 I | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32\n"
	uut := NewETCDLogParser()
	uut.log.SetLevel(logrus.DebugLevel)
	uut.log.SetOutput(&buffer)
	uut.log.Formatter = &logrus.JSONFormatter{
		DisableTimestamp: true,
	}
	// Punch in message character-by-character to catch splitting bugs
	for _, character := range testStr {
		singleChar := string(character)
		err := uut.HandleData([]byte(singleChar))
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
	}
	result := buffer.String()
	if result != "{\"app\":\"etcd\",\"component\":\"etcdserver\",\"level\":\"info\",\"msg\":\"published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32\"}\n" {
		t.Fatalf("Unexpected output: %s", result)
	}
}

// TestInfoMessage tests multiple etcd info messages
func TestInfoMessageSplitMultiline(t *testing.T) {
	var buffer bytes.Buffer
	testStr := `2018-08-12 16:18:18.718670 I | etcdmain: etcd Version: 3.3.9
2018-08-12 16:18:18.718734 I | etcdmain: Git SHA: fca8add78
2018-08-12 16:18:18.718740 I | etcdmain: Go Version: go1.10.3
2018-08-12 16:18:18.718745 I | etcdmain: Go OS/Arch: linux/amd64
`
	uut := NewETCDLogParser()
	uut.log.SetLevel(logrus.DebugLevel)
	uut.log.SetOutput(&buffer)
	uut.log.Formatter = &logrus.JSONFormatter{
		DisableTimestamp: true,
	}
	// Punch in message character-by-character to catch splitting bugs
	for _, character := range testStr {
		singleChar := string(character)
		err := uut.HandleData([]byte(singleChar))
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
	}
	result := buffer.String()
	cmpStr := `{"app":"etcd","component":"etcdmain","level":"info","msg":"etcd Version: 3.3.9"}
{"app":"etcd","component":"etcdmain","level":"info","msg":"Git SHA: fca8add78"}
{"app":"etcd","component":"etcdmain","level":"info","msg":"Go Version: go1.10.3"}
{"app":"etcd","component":"etcdmain","level":"info","msg":"Go OS/Arch: linux/amd64"}
`
	if result != cmpStr {
		t.Fatalf("Unexpected output: %s", result)
	}
}
