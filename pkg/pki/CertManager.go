package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/pkg/errors"
	"math/big"
	"net"
	"os"
	"path"
	"time"
)

type CertManager struct {
	workdir  string
	keysize  int
	validity time.Duration
}

type RSACertificate struct {
	cert     *x509.Certificate
	key      *rsa.PrivateKey
	pubkey   *rsa.PublicKey
	CertPath string
	KeyPath  string
}

func NewManager(workdir string) *CertManager {
	return &CertManager{
		workdir:  workdir,
		keysize:  2048,
		validity: time.Hour * 24 * 365,
	}
}

func (manager *CertManager) writeCertToFiles(name string, privateKey *rsa.PrivateKey, cert *[]byte, certTmpl *x509.Certificate) (*RSACertificate, error) {
	// Write two PEM files
	// Key
	keypath := path.Join(manager.workdir, name+".key")
	keyOut, err := os.OpenFile(keypath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return nil, errors.Wrap(err, "keyfile creation failed")
	}
	pem.Encode(keyOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	keyOut.Close()
	// Cert
	certpath := path.Join(manager.workdir, name+".pem")
	certOut, err := os.OpenFile(certpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "certificate file creation failed")
	}
	pem.Encode(certOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: *cert,
	})
	keyOut.Close()

	return &RSACertificate{
		key:      privateKey,
		pubkey:   &privateKey.PublicKey,
		cert:     certTmpl,
		CertPath: certpath,
		KeyPath:  keypath,
	}, nil
}

func (manager *CertManager) NewSelfSignedCACert(name string, x509Name pkix.Name, serial int64) (*RSACertificate, error) {
	// Generate cert
	privateKey, err := rsa.GenerateKey(rand.Reader, manager.keysize)
	if err != nil {
		return nil, errors.Wrap(err, "key creation failed")
	}
	certTmpl := x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               x509Name,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(manager.validity),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &certTmpl, &certTmpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "certificate template creation failed")
	}

	return manager.writeCertToFiles(name, privateKey, &cert, &certTmpl)
}

func (manager *CertManager) NewSelfSignedCert(name string, x509Name pkix.Name, serial int64) (*RSACertificate, error) {
	// Generate cert
	privateKey, err := rsa.GenerateKey(rand.Reader, manager.keysize)
	if err != nil {
		return nil, errors.Wrap(err, "key creation failed")
	}
	certTmpl := x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               x509Name,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(manager.validity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &certTmpl, &certTmpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "certificate template creation failed")
	}

	return manager.writeCertToFiles(name, privateKey, &cert, &certTmpl)
}

func (manager *CertManager) NewCert(name string, x509Name pkix.Name, serial int64, isServer bool, isClient bool, sans []string, ca *RSACertificate) (*RSACertificate, error) {
	// Generate cert
	privateKey, err := rsa.GenerateKey(rand.Reader, manager.keysize)
	if err != nil {
		return nil, errors.Wrap(err, "key creation failed")
	}
	certTmpl := x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               x509Name,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(manager.validity),
		BasicConstraintsValid: true,
		IsCA: false,
		ExtKeyUsage: []x509.ExtKeyUsage{},
	}
	if isServer {
		certTmpl.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		certTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

		for _, item := range sans {
			if ip := net.ParseIP(item); ip != nil {
				certTmpl.IPAddresses = append(certTmpl.IPAddresses, ip)
			} else {
				certTmpl.DNSNames = append(certTmpl.DNSNames, item)
			}
		}
	}
	if isClient {
		certTmpl.KeyUsage |= x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		certTmpl.ExtKeyUsage = append(certTmpl.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	}

	cert, err := x509.CreateCertificate(rand.Reader, &certTmpl, ca.cert, &privateKey.PublicKey, ca.key)
	if err != nil {
		return nil, errors.Wrap(err, "certificate template creation failed")
	}

	return manager.writeCertToFiles(name, privateKey, &cert, &certTmpl)
}
