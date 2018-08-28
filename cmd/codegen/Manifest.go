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

// Package main contains the code generator for kubernetes manifests
package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/uubk/microkube/internal/manifests"
)

// main executes the code generator
func main() {
	pkgArg := flag.String("package", "", "Package that the generated sources should be placed in")
	nameArg := flag.String("name", "", "Name of the type to generate")
	srcArg := flag.String("src", "", "YAML manifest to parse")
	dstArg := flag.String("dest", "", "Destination of source file")
	dstMainArg := flag.String("main", "", "Destination of main file (optional)")
	mainPkgBase := flag.String("package-base", "github.com/uubk/microkube/internal", "Destination of main file (optional)")

	flag.Parse()

	if *pkgArg == "" || *srcArg == "" || *nameArg == "" {
		flag.PrintDefaults()
		log.WithFields(log.Fields{
			"pkg": *pkgArg,
			"src": *srcArg,
			"dst": *dstArg,
		}).Fatal("Required parameter missing!")
	}

	cg := manifests.NewManifestCodegen(*srcArg, *pkgArg, *nameArg, *dstArg, *dstMainArg, *mainPkgBase)
	log.Info("Reading file...")
	err := cg.ParseFile()
	if err != nil {
		log.WithError(err).Fatal("Couldn't load file!")
	}
	log.Info("Writing results...")
	err = cg.WriteFiles()
	if err != nil {
		log.WithError(err).Fatal("Couldn't write file!")
	}
}
