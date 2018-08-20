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

import (
	"github.com/pkg/errors"
	"testing"
)

// TestErrors tests whether we correctly bail in case of a parse error
func TestErrors(t *testing.T) {
	uut := NewBaseLogParser(func(s string) error {
		return errors.New("testerror")
	})
	err := uut.HandleData([]byte("\n\n"))
	if err == nil {
		t.Fatal("Expected error missing!")
	}
	if err.Error() != "Couldn't decode buffer: testerror" {
		t.Fatalf("Unexpected error: %s!", err)
	}
}
