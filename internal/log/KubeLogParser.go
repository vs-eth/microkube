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

//go:generate ldetool generate --package log --go-string logs.lde

package log

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

// KubeLogParser handles kubernetes-like log output
type KubeLogParser struct {
	// Base ref
	BaseLogParser

	// Application this belongs to
	app string
	// Regex used to unindent logs
	regexpInstance *regexp.Regexp
}

// NewKubeLogParser creates a KubeLogParser for the application named by 'app'
func NewKubeLogParser(app string) *KubeLogParser {
	obj := KubeLogParser{
		app:            app,
		regexpInstance: regexp.MustCompile("[ ]+"),
	}
	obj.BaseLogParser = *NewBaseLogParser(obj.handleLine)
	return &obj
}

// handleLine handles a single line of log output
func (h *KubeLogParser) handleLine(lineStr string) error {
	if strings.HasPrefix(lineStr, "[restful]") {
		// Ugh. [restful] means that this line is actually a different format
		line := KubeLogLineRestful{}
		ok, err := line.Extract(lineStr)
		if err != nil {
			return errors.Wrap(err, "Couldn't decode line '"+lineStr+"'!")
		}
		if !ok {
			return errors.New("Coudln't decode 'restful' line '" + lineStr + "'")
		}
		logrus.WithFields(logrus.Fields{
			"component": "restful",
			"location":  line.Location,
			"app":       h.app,
		}).Info(line.Message)
	} else {
		// Hopefully this is a normal log line
		line := KubeLogLine{}
		// Fix multi-whitespaces as kube logs are intended for consoles...
		lineStr = h.regexpInstance.ReplaceAllString(lineStr, " ")

		ok, err := line.Extract(lineStr)
		if err != nil {
			return errors.Wrap(err, "Couldn't decode line '"+lineStr+"'!")
		}
		if ok {
			// Yay, this is a normal log entry!
			entry := logrus.WithFields(logrus.Fields{
				"app":      h.app,
				"location": line.Location,
			})

			switch line.SeverityID[0] {
			case 'I':
				entry.Info(line.Message)
			case 'E':
				entry.Error(line.Message)
			case 'W':
				entry.Warning(line.Message)
			case 'D':
				entry.Debug(line.Message)
			case 'N': // Notice is handled as info
				entry.Info(line.Message)
			case 'S': // Severe is handled as error
				entry.Error(line.Message)
			default:
				logrus.WithFields(logrus.Fields{
					"component": "KubeLogParser",
					"app":       "microkube",
					"level":     line.SeverityID[0],
				}).Warn("Unknown severity level in kube log parser")
				logrus.WithFields(logrus.Fields{
					"app": h.app,
				}).Warn(lineStr)
			}
		} else {
			// Whelp. Normal format didn't work out, assume this line is simply unformatted...
			logrus.WithFields(logrus.Fields{
				"app": h.app,
			}).Warn(lineStr)
		}
	}

	return nil
}
