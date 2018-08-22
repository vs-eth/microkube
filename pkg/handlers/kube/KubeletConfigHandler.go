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
	"github.com/uubk/microkube/pkg/handlers"
	"github.com/uubk/microkube/pkg/pki"
	"html/template"
	"os"
)

// kubeletConfigData contains data used when templating a kubelet config. For internal use only.
type kubeletConfigData struct {
	CAFile            string
	CertFile          string
	KeyFile           string
	StaticPodPath     string
	KubeletHealthPort int
}

// CreateKubeletConfig creates a kubelet config from the arguments provided and stores it in 'path'
func CreateKubeletConfig(path string, creds *pki.MicrokubeCredentials, execEnv handlers.ExecutionEnvironment, staticPodPath string) error {
	data := kubeletConfigData{
		CAFile:            creds.KubeCA.CertPath,
		StaticPodPath:     staticPodPath,
		CertFile:          creds.KubeServer.CertPath,
		KeyFile:           creds.KubeServer.KeyPath,
		KubeletHealthPort: execEnv.KubeletHealthPort,
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
healthzPort: {{ .KubeletHealthPort }}
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
