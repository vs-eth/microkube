package pki

import (
	"bufio"
	"crypto/x509/pkix"
	"os/exec"
	"strings"
	"testing"
	"os"
)

// checkCertProperties validates a given certificate's properties using the openssl commandline utility
func checkCertProperties(t *testing.T, cert *RSACertificate, serial, issuerCN, subjectCN, keyUsage, isCA, eku,
	sans string) {
	certCheckCmd := exec.Command("openssl", "x509", "-in", cert.CertPath, "-text", "-noout")
	certCheckBuf, err := certCheckCmd.Output()
	if err != nil {
		t.Error("Unexpected error when checking cert:", err)
		return
	}
	if len(certCheckBuf) == 0 {
		t.Error("Empty output when checking cert")
		return
	}

	numExpected := 0
	if serial != "" {
		numExpected++
	}
	if issuerCN != "" {
		numExpected++
	}
	if subjectCN != "" {
		numExpected++
	}
	if keyUsage != "" {
		numExpected++
	}
	if isCA != "" {
		numExpected++
	}
	if eku != "" {
		numExpected++
	}
	if sans != "" {
		numExpected++
	}

	certCheck := bufio.NewScanner(strings.NewReader(string(certCheckBuf)))
	checkMode := 0
	numChecked := 0
	for certCheck.Scan() {
		line := certCheck.Text()
		if checkMode == 0 {
			if strings.Contains(line, "Serial Number") {
				if serial != "" {
					if !strings.Contains(line, serial) {
						t.Error("Certificate didn't have expected serial")
					}
					numChecked++
				}
			}
			if strings.Contains(line, "Issuer: CN =") {
				if issuerCN != "" {
					if !strings.Contains(line, issuerCN) {
						t.Error("Certificate didn't have expected issuer CN")
					}
					numChecked++
				}
			}
			if strings.Contains(line, "Subject: ") {
				if subjectCN != "" {
					if !strings.Contains(line, subjectCN) {
						t.Error("Certificate didn't have expected subject CN")
					}
					numChecked++
				}
			}
			if strings.Contains(line, "X509v3 Key Usage: critical") {
				if keyUsage != "" {
					checkMode = 1
				}
			}
			if strings.Contains(line, "X509v3 Basic Constraints: critical") {
				if isCA != "" {
					checkMode = 2
				}
			}
			if strings.Contains(line, "X509v3 Extended Key Usage:") {
				if eku != "" {
					checkMode = 3
				}
			}
			if strings.Contains(line, "X509v3 Subject Alternative Name:") {
				if eku != "" {
					checkMode = 4
				}
			}
		} else if checkMode == 1 {
			if strings.Trim(line, " \t") != keyUsage {
				t.Error("Certificate didn't have expected key usage")
			}
			numChecked++
			checkMode = 0
		} else if checkMode == 2 {
			if strings.Trim(line, " \t") != isCA {
				t.Error("Certificate didn't have expected constraints")
			}
			numChecked++
			checkMode = 0
		} else if checkMode == 3 {
			if strings.Trim(line, " \t") != eku {
				t.Error("Certificate didn't have expected extended key usage")
			}
			numChecked++
			checkMode = 0
		} else if checkMode == 4 {
			if strings.Trim(line, " \t") != sans {
				t.Error("Certificate didn't have expected SANs")
			}
			numChecked++
			checkMode = 0
		}
	}
	if numChecked != numExpected {
		t.Error("Not all attributes could be checked")
	}
}

// checkCertKeyMatch checks whether a private key matches a certificate
func checkCertKeyMatch(t *testing.T, cert *RSACertificate) {
	certCheckCmd := exec.Command("openssl", "x509", "-in", cert.CertPath, "-modulus", "-noout")
	certCheckBuf, err := certCheckCmd.Output()
	if err != nil {
		t.Error("Unexpected error when checking cert", err)
	}
	certVal := string(certCheckBuf)
	keyCheckCmd := exec.Command("openssl", "rsa", "-in", cert.KeyPath, "-modulus", "-noout")
	keyCheckBuf, err := keyCheckCmd.Output()
	if err != nil {
		t.Error("Unexpected error when checking cert", err)
	}
	keyVal := string(keyCheckBuf)

	if certVal != keyVal {
		t.Error("Fingerprints do not match")
	}
	if !strings.Contains(certVal, "Modulus=") {
		t.Error("Unexpected openssl output")
	}
}

// TestSelfSignedCertProperties tests creation of a simple self-signed certificate and uses openssl to check it's
// attributes
func TestSelfSignedCertProperties(t *testing.T) {
	tempDir := os.TempDir()
	manager := NewManager(tempDir)
	// Conserve entropy during unit tests (NEVER DO THIS IN DEV OR PROD)
	// and generate extremely short certificates
	manager.keysize = 768
	cert, err := manager.NewSelfSignedCACert("Testcert", pkix.Name{
		CommonName: "Testcert",
	}, 123)
	if err != nil {
		t.Error("Unexpected error when generating cert", err)
	}

	checkCertProperties(t, cert, "Serial Number: 123 (0x7b)", "Issuer: CN = Testcert",
		"Subject: CN = Testcert", "Certificate Sign", "CA:TRUE", "", "")
}

// TestSelfSignedCertMatch tests creation of a simple self-signed certificate and checks whether it's public and private
// key are readable and match each other
func TestSelfSignedCertMatch(t *testing.T) {
	tempDir := os.TempDir()
	manager := NewManager(tempDir)
	// Conserve entropy during unit tests (NEVER DO THIS IN DEV OR PROD)
	// and generate extremely short certificates
	manager.keysize = 768
	cert, err := manager.NewSelfSignedCACert("Testcert", pkix.Name{
		CommonName: "Testcert",
	}, 123)
	if err != nil {
		t.Error("Unexpected error when generating cert", err)
	}

	checkCertKeyMatch(t, cert)
}

// TestCASignedClientCert tests creation of a CA and a CA-signed client cert. The properties of the client cert are then
// examined
func TestCASignedClientCert(t *testing.T) {
	tempDir := os.TempDir()
	manager := NewManager(tempDir)
	// Conserve entropy during unit tests (NEVER DO THIS IN DEV OR PROD)
	// and generate extremely short certificates
	manager.keysize = 768
	caCert, err := manager.NewSelfSignedCACert("Testcert", pkix.Name{
		CommonName: "Testcert",
	}, 123)
	if err != nil {
		t.Error("Unexpected error when generating CA cert", err)
	}
	cert, err := manager.NewCert("Testclient", pkix.Name{
		Organization: []string{"system:masters"},
		CommonName:   "Testclient",
	}, 124, false, true, nil, caCert)
	if err != nil {
		t.Error("Unexpected error when generating client cert", err)
	}

	checkCertProperties(t, cert, "Serial Number: 124 (0x7c)", "Issuer: CN = Testcert",
		"Subject: O = system:masters, CN = Testclient", "Digital Signature, Key Encipherment",
		"CA:FALSE", "TLS Web Client Authentication", "")
	checkCertKeyMatch(t, cert)
	checkCertKeyMatch(t, caCert)
}

// TestCASignedServerCert tests creation of a CA and a CA-signed server cert. The properties of the server cert are then
// examined
func TestCASignedServerCert(t *testing.T) {
	tempDir := os.TempDir()
	manager := NewManager(tempDir)
	// Conserve entropy during unit tests (NEVER DO THIS IN DEV OR PROD)
	// and generate extremely short certificates
	manager.keysize = 768
	caCert, err := manager.NewSelfSignedCACert("Testcert", pkix.Name{
		CommonName: "Testcert",
	}, 123)
	if err != nil {
		t.Error("Unexpected error when generating CA cert", err)
	}
	cert, err := manager.NewCert("Testserver", pkix.Name{
		CommonName: "Testserver",
	}, 124, true, false, []string{
		"127.0.0.1",
		"example.com",
	}, caCert)
	if err != nil {
		t.Error("Unexpected error when generating client cert", err)
	}

	checkCertProperties(t, cert, "Serial Number: 124 (0x7c)", "Issuer: CN = Testcert",
		"Subject: CN = Testserver", "Digital Signature, Key Encipherment", "CA:FALSE",
		"TLS Web Server Authentication", "DNS:example.com, IP Address:127.0.0.1")
	checkCertKeyMatch(t, cert)
	checkCertKeyMatch(t, caCert)
}
