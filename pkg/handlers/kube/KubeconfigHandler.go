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
	"encoding/base64"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/pki"
	"html/template"
	"io/ioutil"
	"os"
)

// clientTemplateData contains data used when templating a kubeconfig. For internal use only.
type clientTemplateData struct {
	// CA certificate of the kube api server (base64'd PEM)
	Ca string
	// Client certificate (base64'd PEM)
	Clientcert string
	// Client certificate key (base64'd PEM)
	Clientkey string
	// Address of api server (IP/DNS as string)
	Address string
	// Kube API port
	ApiPort int
}

// Base64EncodedPem encodes file 'src' as base64 and return it as string
func Base64EncodedPem(src string) (string, error) {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		return "", errors.Wrap(err, "unable to read file")
	}
	return base64.StdEncoding.EncodeToString(content), nil
}

// CreateClientKubeconfig creates a certificate-based kubeconfig with an apiserver at "https://<host>:7443" and stores
// it in 'path'
func CreateClientKubeconfig(execEnv handlers.ExecutionEnvironment, creds *pki.MicrokubeCredentials, path,
	host string) error {

	data := clientTemplateData{
		Address: host,
		ApiPort: execEnv.KubeApiPort,
	}
	var err error
	data.Ca, err = Base64EncodedPem(creds.KubeCA.CertPath)
	if err != nil {
		return errors.Wrap(err, "ca encode failed")
	}
	data.Clientcert, err = Base64EncodedPem(creds.KubeClient.CertPath)
	if err != nil {
		return errors.Wrap(err, "client cert encode failed")
	}
	data.Clientkey, err = Base64EncodedPem(creds.KubeClient.KeyPath)
	if err != nil {
		return errors.Wrap(err, "client key encode failed")
	}
	tmplStr := `apiVersion: v1
kind: Config
clusters:
- name: microkube
  cluster:
    server: https://{{ .Address }}:{{ .ApiPort }}
    certificate-authority-data: {{ .Ca }} 
users:
- name: admin
  user:
    client-certificate-data: {{ .Clientcert }}
    client-key-data: {{ .Clientkey }}
contexts:
- context:
    cluster: microkube
    user: admin
  name: default-ctx
current-context: default-ctx`
	tmpl, err := template.New("Client").Parse(tmplStr)
	if err != nil {
		return errors.Wrap(err, "template init failed")
	}
	file, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "file creation failed")
	}
	defer file.Close()
	creds.Kubeconfig = path
	return tmpl.Execute(file, data)
}
