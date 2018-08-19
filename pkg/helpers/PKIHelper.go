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

package helpers

import (
	"crypto/x509/pkix"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/pki"
	"os"
)

// CertHelper generates a CA with a server and a client certificate for _unit testing only_ (weak certificates)
func CertHelper(pkidir, pkiname string) (*pki.RSACertificate, *pki.RSACertificate, *pki.RSACertificate, error) {
	certMgr := pki.NewManager(pkidir)
	certMgr.UutMode()
	ca, err := certMgr.NewSelfSignedCACert(pkiname+"-CA", pkix.Name{
		CommonName: pkiname + "-CA",
	}, 1)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "ca creation failed")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Couldn't read hostname")
	}

	server, err := certMgr.NewCert(pkiname+"-Server", pkix.Name{
		CommonName: pkiname + "-Server",
	}, 2, true, false, []string{
		"127.0.0.1",
		"localhost",
		"0.0.0.0",
		hostname,
	}, ca)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "server certificate creation failed")
	}
	client, err := certMgr.NewCert(pkiname+"-Client", pkix.Name{
		CommonName: pkiname + "-Client",
	}, 3, false, true, nil, ca)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "client certificate creation failed")
	}

	return ca, server, client, nil
}
