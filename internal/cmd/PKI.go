package cmd

import (
	"crypto/x509/pkix"
	log "github.com/sirupsen/logrus"
	"github.com/uubk/microkube/pkg/pki"
	"os"
	"path"
)

// Ensure that a full PKI for 'name' exists in 'root', that is:
// * A CA certificate with name 'name CA' in ca.pem and ca.key
// * A server certificate with SANs 'ip' and name 'name Server' in server.pem and server.key
// * A client certificate with name 'name Client' in 'client.pem' and 'client.key', optionally containing
//   'system:masters' as O when 'isKubeCA' is set to true
func EnsureFullPKI(root, name string, isKubeCA, isETCDCA bool, ip []string) (ca *pki.RSACertificate, server *pki.RSACertificate, client *pki.RSACertificate, err error) {
	caFile := path.Join(root, "ca.pem")
	_, err = os.Stat(caFile)
	if err != nil {
		// File doesn't exist
		certMgr := pki.NewManager(root)
		// Reuse CA code ;)
		ca, err := EnsureCA(root, name)

		ip = append(ip, "127.0.0.1", "localhost")
		server, err := certMgr.NewCert("server", pkix.Name{
			CommonName: name + " Server",
		}, 2, true, isETCDCA, ip, ca)
		if err != nil {
			log.WithError(err).WithField("root", root).Fatal("Couldn't create server cert")
			return nil, nil, nil, err
		}

		clientName := pkix.Name{
			CommonName: name + " Client",
		}
		if isKubeCA {
			clientName.Organization = []string{"system:masters"}
		}
		client, err := certMgr.NewCert("client", clientName, 3, false,true, nil, ca)
		if err != nil {
			log.WithError(err).WithField("root", root).Fatal("Couldn't create client cert")
			return nil, nil, nil, err
		}

		return ca, server, client, nil
	}

	// Certs already exist
	return &pki.RSACertificate{
			KeyPath:  path.Join(root, "ca.key"),
			CertPath: path.Join(root, "ca.pem"),
		}, &pki.RSACertificate{
			KeyPath:  path.Join(root, "server.key"),
			CertPath: path.Join(root, "server.pem"),
		}, &pki.RSACertificate{
			KeyPath:  path.Join(root, "client.key"),
			CertPath: path.Join(root, "client.pem"),
		}, nil
}

// Ensure that a full CA for 'name' exists in 'root', that is:
// * A CA certificate with name 'name CA' in ca.pem and ca.key
func EnsureCA(root, name string) (ca *pki.RSACertificate, err error) {
	caFile := path.Join(root, "ca.pem")
	_, err = os.Stat(caFile)
	if err != nil {
		// File doesn't exist
		certMgr := pki.NewManager(root)
		ca, err := certMgr.NewSelfSignedCACert("ca", pkix.Name{
			CommonName: name + " CA",
		}, 1)
		if err != nil {
			log.WithError(err).WithField("root", root).Fatal("Couldn't create CA")
			return nil, err
		}
		return ca, nil
	}

	// Certs already exist
	return &pki.RSACertificate{
		KeyPath:  path.Join(root, "ca.key"),
		CertPath: path.Join(root, "ca.pem"),
	}, nil
}

// Ensure that a signing cert for 'name' exists in 'root', that is:
// * A CA-like certificate (self-signed) with name 'name CA' in cert.pem and cert.key
// As opposed to 'normal' CA certificates, this certificate can be used to
func EnsureSigningCert(root, name string) (ca *pki.RSACertificate, err error) {
	caFile := path.Join(root, "cert.pem")
	_, err = os.Stat(caFile)
	if err != nil {
		// File doesn't exist
		certMgr := pki.NewManager(root)
		ca, err := certMgr.NewSelfSignedCert("cert", pkix.Name{
			CommonName: name + " Signing Cert",
		}, 1)
		if err != nil {
			log.WithError(err).WithField("root", root).Fatal("Couldn't create signing cert")
			return nil, err
		}
		return ca, nil
	}

	// Certs already exist
	return &pki.RSACertificate{
		KeyPath:  path.Join(root, "cert.key"),
		CertPath: path.Join(root, "cert.pem"),
	}, nil
}
