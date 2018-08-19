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
	"github.com/sirupsen/logrus"
	"strings"
)

// ETCDLogParser handles etcd-like log output
type ETCDLogParser struct {
	// Base ref
	BaseLogParser
}

// NewETCDLogParser creates a ETCDLogParser
func NewETCDLogParser() *ETCDLogParser {
	obj := ETCDLogParser{}
	obj.BaseLogParser = *NewBaseLogParser(obj.handleLine)
	return &obj
}

// handleLine handles a single line of log output
func (h *ETCDLogParser) handleLine(lineStr string) error {
	line := ETCDLogLine{}
	ok, err := line.Extract(lineStr)
	if err != nil {
		return errors.Wrap(err, "Couldn't decode line '"+lineStr+"'!")
	}
	if !ok {
		return errors.New("Couldn't decode line '" + lineStr + "'")
	}

	entry := logrus.WithFields(logrus.Fields{
		"app":       "etcd",
		"component": string(line.Component),
	})

	// TODO(uubk): https://github.com/coreos/etcd/issues/9285 / https://github.com/kubernetes/kubernetes/issues/63316
	// Basically kubernetes healthchecks etcd by only opening a TCP connection without completing the TLS handshake
	// This will result in a warning *every single time* *every 10 seconds*
	// At the moment, we simply drop those messages here :/
	if line.Component == "embed" && strings.HasPrefix(line.Message, "rejected connection from \"127.0.0.1:") {
		if strings.HasSuffix(line.Message, "\" (error \"EOF\", ServerName \"\")") {
			return nil
		}
	}
	// This warning _can not be disabled_. Drop it...
	if line.Component == "etcdmain" && line.Message == "forgot to set Type=notify in systemd service file?" {
		return nil
	}

	switch line.Severity {
	case "I":
		entry.Info(line.Message)
	case "E":
		entry.Error(line.Message)
	case "W":
		entry.Warning(line.Message)
	case "D":
		entry.Debug(line.Message)
	case "N": // Notice is handled as info...
		entry.Info(line.Message)
	default:
		logrus.WithFields(logrus.Fields{
			"component": "EtcdLogParser",
			"app":       "microkube",
			"level":     line.Severity,
		}).Warn("Unknown severity level in etcd log parser")
		entry.Warn(line.Message)
	}

	return nil
}
