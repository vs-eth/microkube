package kube_proxy

import (
	"github.com/pkg/errors"
	"os"
	"text/template"
)

type KubeSchedulerConfigData struct {
	Kubeconfig  string
	ClusterCIDR string
}

func CreateKubeProxyConfig(path, clusterCIDR, kubeconfig string) error {
	data := KubeSchedulerConfigData{
		Kubeconfig:  kubeconfig,
		ClusterCIDR: clusterCIDR,
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
healthzBindAddress: 127.0.0.1:10256
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
metricsBindAddress: 127.0.0.1:10249
mode: ""
nodePortAddresses: null
oomScoreAdj: -999
portRange: ""
resourceContainer: /kube-proxy
udpIdleTimeout: 250ms
`
	// clusterDNS, clusterDomain
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
