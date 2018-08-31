package pki

import (
	"crypto/x509/pkix"
	"fmt"
	"net"
	"os"
	"path"
)

// MicrokubeCredentials manages all credentials needed for the different components of Microkube using PKI
type MicrokubeCredentials struct {
	// CA certificate for etcd
	EtcdCA *RSACertificate
	// Client certificate for etcd
	EtcdClient *RSACertificate
	// Server certificate for etcd
	EtcdServer *RSACertificate
	// CA certificate for kubernetes
	KubeCA *RSACertificate
	// Client certificate for kubernetes
	KubeClient *RSACertificate
	// Server certificate for kubernetes
	KubeServer *RSACertificate
	// CA certificate for kubernetes in-cluster CA
	KubeClusterCA *RSACertificate
	// Signing certificate for kubernetes service account tokens
	KubeSvcSignCert *RSACertificate

	// Path to kubernetes client config file
	Kubeconfig string

	// Weak certificates, testing only, you have been warned
	uutMode bool
}

// CreateOrLoadCertificates creates certificates if they don't already exist or loads them if they do exist
func (m *MicrokubeCredentials) CreateOrLoadCertificates(baseDir string, bindAddr, serviceAddr net.IP) error {
	var err error
	os.Mkdir(path.Join(baseDir, "etcdtls"), 0750)
	m.EtcdCA, m.EtcdServer, m.EtcdClient, err = m.ensureFullPKI(path.Join(baseDir, "etcdtls"), "Microkube ETCD",
		false, true, []string{bindAddr.String()})
	if err != nil {
		return fmt.Errorf("etcd pki creation failed: %s", err)
	}
	os.Mkdir(path.Join(baseDir, "kubetls"), 0750)
	m.KubeCA, m.KubeServer, m.KubeClient, err = m.ensureFullPKI(path.Join(baseDir, "kubetls"), "Microkube Kubernetes",
		true, false, []string{bindAddr.String(), serviceAddr.String()})
	if err != nil {
		return fmt.Errorf("kube pki creation failed: %s", err)
	}
	os.Mkdir(path.Join(baseDir, "kubectls"), 0750)
	m.KubeClusterCA, err = m.ensureCA(path.Join(baseDir, "kubectls"), "Microkube Cluster CA")
	if err != nil {
		return fmt.Errorf("kube cluster pki creation failed: %s", err)
	}
	os.Mkdir(path.Join(baseDir, "kubestls"), 0750)
	m.KubeSvcSignCert, err = m.ensureSigningCert(path.Join(baseDir, "kubestls"), "Microkube Cluster SVC Signcert")
	if err != nil {
		return fmt.Errorf("kube service signing cert creation failed: %s", err)
	}
	return nil
}

// ensureFullPKI ensures that a full PKI for 'name' exists in 'root', that is:
//  - A CA certificate with name 'name CA' in ca.pem and ca.key
//  - A server certificate with SANs 'ip' and name 'name Server' in server.pem and server.key
//  - A client certificate with name 'name Client' in 'client.pem' and 'client.key', optionally containing
//    'system:masters' as O when 'isKubeCA' is set to true
func (m *MicrokubeCredentials) ensureFullPKI(root, name string, isKubeCA, isETCDCA bool,
	ip []string) (ca *RSACertificate, server *RSACertificate, client *RSACertificate, err error) {

	caFile := path.Join(root, "ca.pem")
	_, err = os.Stat(caFile)
	if err != nil {
		// File doesn't exist
		certMgr := NewManager(root)
		if m.uutMode {
			certMgr.UutMode()
		}
		// Reuse CA code ;)
		ca, err := m.ensureCA(root, name)
		if err != nil {
			// Already logged
			return nil, nil, nil, err
		}

		hostname, err := os.Hostname()
		if err != nil {
			return nil, nil, nil, err
		}

		ip = append(ip, "127.0.0.1", "localhost", hostname)
		server, err := certMgr.NewCert("server", pkix.Name{
			CommonName: name + " Server",
		}, 2, true, isETCDCA, ip, ca)
		if err != nil {
			return nil, nil, nil, err
		}

		clientName := pkix.Name{
			CommonName: name + " Client",
		}
		if isKubeCA {
			clientName.Organization = []string{"system:masters"}
		}
		client, err := certMgr.NewCert("client", clientName, 3, false, true, nil, ca)
		if err != nil {
			return nil, nil, nil, err
		}

		return ca, server, client, nil
	}

	// Certs already exist
	return &RSACertificate{
			KeyPath:  path.Join(root, "ca.key"),
			CertPath: path.Join(root, "ca.pem"),
		}, &RSACertificate{
			KeyPath:  path.Join(root, "server.key"),
			CertPath: path.Join(root, "server.pem"),
		}, &RSACertificate{
			KeyPath:  path.Join(root, "client.key"),
			CertPath: path.Join(root, "client.pem"),
		}, nil
}

// EnsureCA ensures that a full CA for 'name' exists in 'root', that is:
//  - A CA certificate with name 'name CA' in ca.pem and ca.key
func (m *MicrokubeCredentials) ensureCA(root, name string) (ca *RSACertificate, err error) {
	caFile := path.Join(root, "ca.pem")
	_, err = os.Stat(caFile)
	if err != nil {
		// File doesn't exist
		certMgr := NewManager(root)
		if m.uutMode {
			certMgr.UutMode()
		}
		ca, err := certMgr.NewSelfSignedCACert("ca", pkix.Name{
			CommonName: name + " CA",
		}, 1)
		if err != nil {
			return nil, err
		}
		return ca, nil
	}

	// Certs already exist
	return &RSACertificate{
		KeyPath:  path.Join(root, "ca.key"),
		CertPath: path.Join(root, "ca.pem"),
	}, nil
}

// EnsureSigningCert ensures that a signing cert for 'name' exists in 'root', that is:
//  - A CA-like certificate (self-signed) with name 'name CA' in cert.pem and cert.key
func (m *MicrokubeCredentials) ensureSigningCert(root, name string) (ca *RSACertificate, err error) {
	caFile := path.Join(root, "cert.pem")
	_, err = os.Stat(caFile)
	if err != nil {
		// File doesn't exist
		certMgr := NewManager(root)
		if m.uutMode {
			certMgr.UutMode()
		}
		ca, err := certMgr.NewSelfSignedCert("cert", pkix.Name{
			CommonName: name + " Signing Cert",
		}, 1)
		if err != nil {
			return nil, err
		}
		return ca, nil
	}

	// Certs already exist
	return &RSACertificate{
		KeyPath:  path.Join(root, "cert.key"),
		CertPath: path.Join(root, "cert.pem"),
	}, nil
}
