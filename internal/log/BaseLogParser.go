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
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
)

// loggerList contains a global map of loggers used so that instances can be associated with a logger
var loggerList = make(map[string]*logrus.Logger)

// loggerListMutex secures access to loggerList
var loggerListMutex = sync.Mutex{}

// GetLoggerFor creates (if necessary) and returns a logger for a log parser of name 'name'
func GetLoggerFor(name string) *logrus.Logger {
	loggerListMutex.Lock()
	logPtr := loggerList[name]
	if logPtr == nil {
		logPtr = logrus.New()
		loggerList[name] = logPtr
	}
	loggerListMutex.Unlock()

	return logPtr
}

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
	mutex       *sync.Mutex
	log         *logrus.Logger
}

// NewBaseLogParser creates a new log parser that uses the provided line handler and a logger 'name'
func NewBaseLogParser(lineHandler LineHandlerFunc, name string) *BaseLogParser {
	obj := &BaseLogParser{
		buf:         &bytes.Buffer{},
		lineHandler: lineHandler,
		mutex:       &sync.Mutex{},
		log:         GetLoggerFor(name),
	}

	return obj
}

func (lp *BaseLogParser) SetLevel(level logrus.Level) {
	lp.log.SetLevel(level)
}

// HandleData is invoked for each new buffer of data, see docs of interface type
func (lp *BaseLogParser) HandleData(data []byte) error {
	lp.mutex.Lock()

	if data != nil {
		lp.buf.Write(data)
	}

	consumedData := false
	if strings.Contains(lp.buf.String(), "\n") {
		// Read string cannot return an error since we just checked for delim and hold a lock on the buffer
		line, _ := lp.buf.ReadString('\n')
		consumedData = true
		err := lp.lineHandler(line)
		if err != nil {
			lp.mutex.Unlock()
			return errors.Wrap(err, "Couldn't decode buffer")
		}
	}

	if consumedData {
		lp.mutex.Unlock()
		return lp.HandleData(nil)
	}

	lp.mutex.Unlock()
	return nil
}
