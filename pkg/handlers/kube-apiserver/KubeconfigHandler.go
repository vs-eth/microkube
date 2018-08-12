package kube_apiserver

import (
	"encoding/base64"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/pki"
	"html/template"
	"io/ioutil"
	"os"
)

// Data used when templating a kubeconfig. For internal use only.
type clientTemplateData struct {
	// CA certificate of the kube api server (base64'd PEM)
	Ca string
	// Client certificate (base64'd PEM)
	Clientcert string
	// Client certificate key (base64'd PEM)
	Clientkey string
	// Address of api server (IP/DNS as string)
	Address string
}

// Encode file 'src' as base64 and return it as string
func Base64EncodedPem(src string) (string, error) {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		return "", errors.Wrap(err, "unable to read file")
	}
	return base64.StdEncoding.EncodeToString(content), nil
}

// Create a certificate-based kubeconfig with an apiserver at "https://<host>:7443" and store it in 'path'
func CreateClientKubeconfig(ca, cert *pki.RSACertificate, path, host string) error {
	data := clientTemplateData{
		Address: host,
	}
	var err error
	data.Ca, err = Base64EncodedPem(ca.CertPath)
	if err != nil {
		return errors.Wrap(err, "ca encode failed")
	}
	data.Clientcert, err = Base64EncodedPem(cert.CertPath)
	if err != nil {
		return errors.Wrap(err, "client cert encode failed")
	}
	data.Clientkey, err = Base64EncodedPem(cert.KeyPath)
	if err != nil {
		return errors.Wrap(err, "client key encode failed")
	}
	tmplStr := `apiVersion: v1
kind: Config
clusters:
- name: microkube
  cluster:
    server: https://{{ .Address }}:7443
    certificate-authority-data: {{ .Ca }} 
users:
- name: admin
  user:
    client-certificate-data: {{ .Clientcert }}
    client-key-data: {{ .Clientkey }}
contexts:
- context:
    cluster: microkube
    user: admin`
	tmpl, err := template.New("Client").Parse(tmplStr)
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