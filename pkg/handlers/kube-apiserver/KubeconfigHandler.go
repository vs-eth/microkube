package kube_apiserver

import (
	"github.com/uubk/microkube/pkg/pki"
	"html/template"
	"github.com/pkg/errors"
	"os"
	"io/ioutil"
	"encoding/base64"
)

type ClientTemplateData struct {
	Ca string
	Clientcert string
	Clientkey string
}

func Base64EncodedPem(src string) (string, error) {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		return "", errors.Wrap(err, "unable to read file")
	}
	return base64.StdEncoding.EncodeToString(content), nil
}

func CreateClientKubeconfig(ca, cert *pki.RSACertificate, path string) error {
	data := ClientTemplateData{}
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
- name: visprod
  cluster:
    server: https://127.0.0.1:7443
    certificate-authority-data: {{ .Ca }} 
users:
- name: kubelet
  user:
    client-certificate-data: {{ .Clientcert }}
    client-key-data: {{ .Clientkey }}
contexts:
- context:
    cluster: visprod
    user: kubelet`
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