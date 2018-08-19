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

// Package log contains log parsers for output of subprocesses
package log

import (
	"bufio"
	"bytes"
	"github.com/pkg/errors"
	"strings"
)

// LineHandlerFunc describes a function that is able to consume a log line
type LineHandlerFunc func(string) error

// Parser is the interface type of all log-parsing classes in this package.
// A Parser is used to handle the output of child processes and re-log it using the logger of the main process,
// unifying all logs into a single log with a coherent structure
type Parser interface {
	// HandleData takes all necessary actions with the provided buffer
	HandleData(data []byte) error
}

// BaseLogParser provides utilities often needed to implement log parsers, that is line reassembly functionality
type BaseLogParser struct {
	buf         *bytes.Buffer
	bufReader   *bufio.Scanner
	lineHandler LineHandlerFunc
}

// NewBaseLogParser creates a new log parser that uses the provided line handler
func NewBaseLogParser(lineHandler LineHandlerFunc) *BaseLogParser {
	obj := &BaseLogParser{
		buf:         &bytes.Buffer{},
		lineHandler: lineHandler,
	}
	return obj
}

// HandleData is invoked for each new buffer of data, see docs of interface type
func (lp *BaseLogParser) HandleData(data []byte) error {
	lp.buf.Write(data)

	if strings.Contains(lp.buf.String(), "\n") {
		line, err := lp.buf.ReadString('\n')
		if err == nil {
			err := lp.lineHandler(line)
			if err != nil {
				return errors.Wrap(err, "Couldn't decode buffer")
			}
		} else {
			return errors.New("Buffer contained '\\n' but no line could be read?")
		}
	}

	return nil
}
