package kubelet

import (
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/pki"
	"html/template"
	"os"
)

type KubeletConfigData struct {
	CAFile        string
	StaticPodPath string
}

func CreateKubeletConfig(path string, ca *pki.RSACertificate, staticPodPath string) error {
	data := KubeletConfigData{
		CAFile:        ca.CertPath,
		StaticPodPath: staticPodPath,
	}
	tmplStr := `kind: KubeletConfiguration
apiVersion: kubelet.config.k8s.io/v1beta1
evictionHard:
    memory.available:  "2Gi"
authentication:
  anonymous:
    enabled: false
  x509:
    clientCAFile: {{ .CAFile }}
staticPodPath: {{ .StaticPodPath }}
healthzBindAddress: 127.0.0.1
healthzPort: 10248
hostnameOverride: thisNode`
	// clusterDNS, clusterDomain
	tmpl, err := template.New("Kubelet").Parse(tmplStr)
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
