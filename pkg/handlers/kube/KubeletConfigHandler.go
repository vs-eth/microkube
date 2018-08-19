package kube

import (
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/pki"
	"html/template"
	"os"
)

// kubeletConfigData contains data used when templating a kubelet config. For internal use only.
type kubeletConfigData struct {
	CAFile        string
	CertFile      string
	KeyFile       string
	StaticPodPath string
}

// CreateKubeletConfig creates a kubelet config from the arguments provided and stores it in 'path'
func CreateKubeletConfig(path string, server, ca *pki.RSACertificate, staticPodPath string) error {
	data := kubeletConfigData{
		CAFile:        ca.CertPath,
		StaticPodPath: staticPodPath,
		CertFile:      server.CertPath,
		KeyFile:       server.KeyPath,
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
kubeletCgroups: "/systemd/system.slice"
tlsCertFile: {{ .CertFile }}
tlsPrivateKeyFile: {{ .KeyFile }}
`
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
