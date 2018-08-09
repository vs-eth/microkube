package etcd

import (
	"github.com/pkg/errors"
	"github.com/uubk/microkube/pkg/helpers"
	"github.com/uubk/microkube/pkg/pki"
	"io/ioutil"
	"os"
	"path"
	)

func GetBinary(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "couldn't read cwd")
	}
	wd = path.Dir(path.Dir(path.Dir(wd)))
	wd = path.Join(wd, "third_party", name)
	return wd, nil
}

func StartETCDForTest(exitHandler helpers.ExitHandler) (*EtcdHandler, *pki.RSACertificate, *pki.RSACertificate, error) {
	tmpdir, err := ioutil.TempDir("", "microkube-unittests-etcd")
	if err != nil {
		errors.Wrap(err, "tempdir creation failed")
	}
	ca, server, client, err := helpers.CertHelper(tmpdir, "etcd-unittest")
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error in PKI handler")
	}

	outputHandler := func(output []byte) {
		// fmt.Println("etcd   |", string(output))
		// Nop
	}

	wd, err := GetBinary("etcd")
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error while searching for etcd binary")
	}

	// UUT
	handler := NewEtcdHandler(tmpdir, wd, server.CertPath, server.KeyPath, ca.CertPath, outputHandler, 0, exitHandler)
	err = handler.Start()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "etcd startup failed")
	}

	healthMessage := make(chan helpers.HealthMessage, 1)
	handler.EnableHealthChecks(ca, client, healthMessage, false)
	msg := <-healthMessage
	if !msg.IsHealthy {
		return nil, nil, nil, errors.Wrap(msg.Error, "ETCD unhealthy: ")
	}

	return handler, ca, client, nil
}
