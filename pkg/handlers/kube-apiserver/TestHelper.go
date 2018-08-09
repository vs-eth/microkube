package kube_apiserver

import (
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers/etcd"
	"github.com/uubk/microkube/pkg/helpers"
	"github.com/uubk/microkube/pkg/pki"
	"io/ioutil"
	"crypto/x509/pkix"
		"time"
)

func CertHelper(pkidir, pkiname string) (*pki.RSACertificate, *pki.RSACertificate, *pki.RSACertificate, error) {
	certmgr := pki.NewManager(pkidir)
	ca, err := certmgr.NewSelfSignedCert(pkiname+"-CA", 1)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "ca creation failed")
	}
	server, err := certmgr.NewCert(pkiname+"-Server", pkix.Name{
		CommonName: pkiname+"-Server",
	}, 2, true, []string{
		"127.0.0.1",
		"localhost",
	}, ca)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "server certificate creation failed")
	}
	client, err := certmgr.NewCert(pkiname+"-Client", pkix.Name{
		CommonName: pkiname+"-Client",
		Organization: []string{"system:masters"}, // THIS FIXES RBAC PERMISSIONS!
	}, 3, false, nil, ca)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "client certificate creation failed")
	}

	return ca, server, client, nil
}


func StartKubeAPIServerForTest(exitHandler helpers.ExitHandler) (*KubeAPIServerHandler, *etcd.EtcdHandler, *pki.RSACertificate, *pki.RSACertificate, error) {
	etcdHandler, etcdCA, etcdClientCert, err := etcd.StartETCDForTest(exitHandler)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "ETCD startup failed")
	}

	tmpdir, err := ioutil.TempDir("", "microkube-unittests-kubeapi")
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "tempdir creation failed")
	}

	kubeCA, kubeServer, kubeClient, err := CertHelper(tmpdir, "kubeapi-unittest")
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "kube CA setup failed")
	}

	bin, err := etcd.GetBinary("kube-apiserver")
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "kube apiserver binary not found")
	}

	outputHandler := func(output []byte) {
		// fmt.Println("kube   |", string(output))
		// Nop
	}

	uut := NewKubeAPIServerHandler(bin, kubeServer.CertPath, kubeServer.KeyPath, kubeClient.CertPath, kubeClient.KeyPath,
		kubeCA.CertPath, etcdClientCert.CertPath, etcdClientCert.KeyPath, etcdCA.CertPath, outputHandler, exitHandler)
	err = uut.Start()
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "kube apiserver didn't launch")
	}

	msgChan := make(chan helpers.HealthMessage, 1)
	msg := helpers.HealthMessage{
		IsHealthy: false,
	}

	for i := 0; i < 10 && !msg.IsHealthy; i++ {
		time.Sleep(2* time.Second)
		uut.EnableHealthChecks(kubeCA, kubeClient, msgChan, false)
		msg = <-msgChan
	}
	if !msg.IsHealthy {
		return nil, nil, nil, nil, errors.Wrap(msg.Error, "kube apiserver health check failed")
	}

	return uut, etcdHandler, kubeCA, kubeClient, nil
}
