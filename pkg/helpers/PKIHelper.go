package helpers

import (
	"crypto/x509/pkix"
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/pki"
	"os"
)

func CertHelper(pkidir, pkiname string) (*pki.RSACertificate, *pki.RSACertificate, *pki.RSACertificate, error) {
	certmgr := pki.NewManager(pkidir)
	ca, err := certmgr.NewSelfSignedCACert(pkiname+"-CA", pkix.Name{
		CommonName: pkiname + "-CA",
	}, 1)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "ca creation failed")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Couldn't read hostname")
	}

	server, err := certmgr.NewCert(pkiname+"-Server", pkix.Name{
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
	client, err := certmgr.NewCert(pkiname+"-Client", pkix.Name{
		CommonName: pkiname + "-Client",
	}, 3, false, true, nil, ca)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "client certificate creation failed")
	}

	return ca, server, client, nil
}
