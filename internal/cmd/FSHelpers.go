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

// Package cmd (internal) provides functions that are needed to implement the commands
package cmd

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path"
)

// EnsureDir ensures that root/subdirectory exists, is a directory and has permissions 'permissions'
func EnsureDir(root, subdirectory string, permissions os.FileMode) error {
	dir := path.Join(root, subdirectory)

	// Errors in mkdir are ignored
	err := os.Mkdir(dir, permissions)
	if err == nil {
		log.WithField("dir", dir).Debug("Directory created")
	}

	info, err := os.Stat(dir)
	if err != nil {
		log.WithField("dir", dir).WithError(err).Warn("Couldn't stat directory")
		return err
	}
	if !info.IsDir() {
		log.WithField("dir", dir).WithError(err).Warn("Directory is not a directory")
		return err
	}
	return nil
}
