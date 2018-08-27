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

package manifests

import (
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	cmd2 "k8s.io/kubernetes/pkg/kubectl/cmd"
	"os"
)

type KubeManifestBase struct {
	objects []string
}

func (m *KubeManifestBase) Register(manifest string) {
	m.objects = append(m.objects, manifest)
}

func (m *KubeManifestBase) ApplyToCluster(kubeconfig string) error {
	file, err := ioutil.TempFile("", "kube-apply-manifest")
	if err != nil {
		file.Close()
		return err
	}

	for _, obj := range m.objects {
		for pos := 0; pos < len(obj); {
			n, err := file.Write([]byte(obj))
			if err != nil {
				panic(err)
			}
			pos += n
		}
	}

	file.Close()

	clientcmd.ClusterDefaults.Server = ""

	cmd := cmd2.NewKubectlCommand(nil, os.Stdout, os.Stderr)
	args := []string{
		"--kubeconfig=" + kubeconfig,
		/*"config",
		"view",*/
		//"version",
		"apply",
		"-f",
		file.Name(),
	}
	cmd.SetArgs(args)

	return cmd.Execute()
}
