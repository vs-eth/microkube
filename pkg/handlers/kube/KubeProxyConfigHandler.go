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

package kube

import (
	"github.com/pkg/errors"
	"github.com/vs-eth/microkube/pkg/handlers"
	"os"
	"text/template"
)

// kubeProxyConfigData contains data used when templating a kube proxy config. For internal use only.
type kubeProxyConfigData struct {
	Kubeconfig           string
	ClusterCIDR          string
	KubeProxyHealthPort  int
	KubeProxyMetricsPort int
}

// CreateKubeProxyConfig creates a proxy config with most things hardcoded and stores it in 'path'
func CreateKubeProxyConfig(path, clusterCIDR, kubeconfig string, execEnv handlers.ExecutionEnvironment) error {
	data := kubeProxyConfigData{
		Kubeconfig:           kubeconfig,
		ClusterCIDR:          clusterCIDR,
		KubeProxyHealthPort:  execEnv.KubeProxyHealthPort,
		KubeProxyMetricsPort: execEnv.KubeProxyMetricsPort,
	}
	tmplStr := `apiVersion: kubeproxy.config.k8s.io/v1alpha1
bindAddress: 0.0.0.0
clientConnection:
  acceptContentTypes: ""
  burst: 10
  contentType: application/vnd.kubernetes.protobuf
  kubeconfig: "{{ .Kubeconfig }}"
  qps: 5
configSyncPeriod: 15m0s
clusterCIDR: "{{ .ClusterCIDR }}"
conntrack:
  max: 0
  maxPerCore: 32768
  min: 131072
  tcpCloseWaitTimeout: 1h0m0s
  tcpEstablishedTimeout: 24h0m0s
enableProfiling: false
healthzBindAddress: 127.0.0.1:{{ .KubeProxyHealthPort }}
hostnameOverride: ""
iptables:
  masqueradeAll: false
  masqueradeBit: 14
  minSyncPeriod: 0s
  syncPeriod: 30s
ipvs:
  excludeCIDRs: null
  minSyncPeriod: 0s
  scheduler: ""
  syncPeriod: 30s
kind: KubeProxyConfiguration
metricsBindAddress: 127.0.0.1:{{ .KubeProxyMetricsPort }}
nodePortAddresses: null
oomScoreAdj: -999
portRange: ""
resourceContainer: /kube-proxy
udpIdleTimeout: 250ms
`
	// TODO(uubk): clusterDNS, clusterDomain
	tmpl, err := template.New("KubeScheduler").Parse(tmplStr)
	if err != nil {
		return errors.Wrap(err, "template init failed")
	}
	file, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "file creation failed")
	}
	defer file.Close()
	return tmpl.Execute(file, data)
}
