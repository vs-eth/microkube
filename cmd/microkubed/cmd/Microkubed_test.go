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
	"github.com/uubk/microkube/internal/cmd"
	"io/ioutil"
	"testing"
)

// FullMicrokubedIntegrationTest runs a full integration test, that is, it bootstraps a full cluster and waits until it
// is healthy. This requires:
//  - passwordless sudo
//  - iptables rules that do not restrict the pod/service networks
//  - access to the docker socket
//  - Linux
func TestIntegrationMicrokubed(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	obj := Microkubed{}

	// Emulate handleArgs
	rootdir, err := ioutil.TempDir("", "microkube-integration-test")
	if err != nil {
		t.Fatalf("tempdir creation failed: '%s'", err)
	}
	obj.baseDir = rootdir
	obj.podRangeNet, obj.serviceRangeNet, obj.clusterIPRange, obj.bindAddr, obj.serviceRangeIP, err =
		cmd.CalculateIPRanges("192.168.250.1/24", "192.168.251.1/24")
	if err != nil {
		t.Fatalf("ipcalc failed: '%s'", err)
	}
	obj.sudoMethod = "/usr/bin/sudo"
	obj.gracefulTerminationMode = false
	obj.start()
	obj.waitUntilNodeReady()
	// Cluster is running, node is healthy, we're done here
}
