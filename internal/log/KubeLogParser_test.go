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

package log

import "testing"

// TestWarningMessage tests a single warning message
func TestWarningMessage(t *testing.T) {
	testStr := "W0812 17:00:08.194751   25997 genericapiserver.go:319] Skipping API scheduling.k8s.io/v1alpha1 because it has no resources.\n"
	uut := NewKubeLogParser("testkubeapp")
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

// TestWarningMessage tests a single 'restful' info message
func TestRestfulMessage(t *testing.T) {
	testStr := "[restful] 2018/08/12 17:00:09 log.go:33: [restful/swagger] listing is available at https://172.17.0.1:7443/swaggerapi\n"
	uut := NewKubeLogParser("testkubeapp")
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}
