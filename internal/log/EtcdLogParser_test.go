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

// TestInfoMessage tests a single etcd info message
func TestInfoMessage(t *testing.T) {
	testStr := "2018-08-12 14:13:48.437712 I | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32\n"
	uut := NewETCDLogParser()
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

// TestInfoMessageSplit tests a single etcd info message but feeding it byte-for-byte
func TestInfoMessageSplit(t *testing.T) {
	testStr := "2018-08-12 14:13:48.437712 I | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32\n"
	uut := NewETCDLogParser()
	// Punch in message character-by-character to catch splitting bugs
	for _, character := range testStr {
		singleChar := string(character)
		err := uut.HandleData([]byte(singleChar))
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
	}
}

// TestInfoMessage tests multiple etcd info messages
func TestInfoMessageSplitMultiline(t *testing.T) {
	testStr := `2018-08-12 16:18:18.718670 I | etcdmain: etcd Version: 3.3.9
2018-08-12 16:18:18.718734 I | etcdmain: Git SHA: fca8add78
2018-08-12 16:18:18.718740 I | etcdmain: Go Version: go1.10.3
2018-08-12 16:18:18.718745 I | etcdmain: Go OS/Arch: linux/amd64
`
	uut := NewETCDLogParser()
	// Punch in message character-by-character to catch splitting bugs
	for _, character := range testStr {
		singleChar := string(character)
		err := uut.HandleData([]byte(singleChar))
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
	}
}
