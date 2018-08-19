package kube

import (
	"github.com/pkg/errors"
	"os"
	"text/template"
)


// kubeSchedulerConfigData contains data used when templating a kube scheduler config. For internal use only.
type kubeSchedulerConfigData struct {
	Kubeconfig string
}

// CreateKubeSchedulerConfig creates a proxy config with most things hardcoded and stores it in 'path'
func CreateKubeSchedulerConfig(path, kubeconfig string) error {
	data := kubeSchedulerConfigData{
		Kubeconfig: kubeconfig,
	}
	tmplStr := `algorithmSource:
  provider: DefaultProvider
apiVersion: componentconfig/v1alpha1
clientConnection:
  acceptContentTypes: ""
  burst: 100
  contentType: application/vnd.kubernetes.protobuf
  kubeconfig: "{{ .Kubeconfig }}"
  qps: 50
disablePreemption: false
enableContentionProfiling: false
enableProfiling: false
failureDomains: kubernetes.io/hostname,failure-domain.beta.kubernetes.io/zone,failure-domain.beta.kubernetes.io/region
hardPodAffinitySymmetricWeight: 1
healthzBindAddress: 127.0.0.1:10251
kind: KubeSchedulerConfiguration
leaderElection:
  leaderElect: true
  leaseDuration: 15s
  lockObjectName: kube-scheduler
  lockObjectNamespace: kube-system
  renewDeadline: 10s
  resourceLock: endpoints
  retryPeriod: 2s
metricsBindAddress: 127.0.0.1:10251
schedulerName: default-scheduler
`
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
