package kube_apiserver

import (
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/handlers/etcd"
	"github.com/uubk/microkube/pkg/helpers"
	"github.com/uubk/microkube/pkg/pki"
	"io/ioutil"
)

func StartKubeAPIServerForTest(exitHandler helpers.ExitHandler) (*KubeAPIServerHandler, *etcd.EtcdHandler, *pki.RSACertificate, *pki.RSACertificate, error) {
	etcdHandler, etcdCA, etcdClientCert, err := etcd.StartETCDForTest(exitHandler)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "ETCD startup failed")
	}

	tmpdir, err := ioutil.TempDir("", "microkube-unittests-kubeapi")
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "tempdir creation failed")
	}

	kubeCA, kubeServer, kubeClient, err := helpers.CertHelper(tmpdir, "kubeapi-unittest")
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
	uut.EnableHealthChecks(kubeCA, kubeClient, msgChan, false)
	msg := <-msgChan

	if !msg.IsHealthy {
		return nil, nil, nil, nil, errors.Wrap(msg.Error, "kube apiserver health check failed")
	}

	return uut, etcdHandler, kubeCA, kubeClient, nil
}
