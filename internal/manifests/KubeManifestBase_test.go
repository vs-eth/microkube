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
	"errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

// TestBaseFunctions tests whether KubeManifestBase follows state transitions correctly
func TestBaseFunctions(t *testing.T) {
	uut := KubeManifestBase{}
	uut.SetName("test")
	uut.Register("manifest")
	uut.RegisterHO(testDeployment)

	assert.Equal(t, "test", uut.Name(), "wrong name")
	assert.Equal(t, []string{"manifest"}, uut.objects, "wrong object")
	assert.Equal(t, testDeployment, uut.healthObj, "wrong health object")

	file, err := uut.dumpToFile()
	assert.NotEmpty(t, file, "unexpected empty file return")
	assert.NoError(t, err, "unexpected error")

	err = uut.InitHealthCheck("")
	if assert.Error(t, err) {
		assert.Equal(t, "invalid configuration: no configuration has been provided", err.Error(), "wrong error returned")
	}
	uut.client = fake.NewSimpleClientset()

	health, err := uut.IsHealthy()
	if err == nil {
		assert.Equal(t, errors.New(""), err, "error missing")
	}
	assert.Equal(t, false, health, "unexpected health")
}
