# Microkube
A small tool to quickly bootstrap a kubernetes cluster against a local docker daemon

[![Build Status](https://travis-ci.com/uubk/microkube.svg?branch=master)](https://travis-ci.com/uubk/microkube)
[![Go Report Card](https://goreportcard.com/badge/github.com/uubk/microkube?style=flat)](https://goreportcard.com/report/github.com/uubk/microkube)
[![codecov](https://codecov.io/gh/uubk/microkube/branch/master/graph/badge.svg)](https://codecov.io/gh/uubk/microkube)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](http://godoc.org/github.com/uubk/microkube)
[![Release](https://img.shields.io/github/tag/uubk/microkube.svg?style=flat)](https://github.com/uubk/microkube/releases/latest)

## Motivation
##### Debugging 'Kubernetes Apps' 
Traditionally, when debugging kubernetes applications locally, you'd use minikube.
However, this quickly results in quite some overhead:
* Local directories are only accessible from the cluster if you set up NFS
* Service IPs are inaccessible, you have to manually use `kubectl proxy` or expose them on the minikube node
* Cluster DNS is hidden
* Local images have to be pushed to some registry which is nontrivial as unencrypted docker registries normally trigger Dockers Insecure registry logic

## How it works
Microkube generates configuration and certificates and then starts kubernetes
*directly* against a local docker daemon. At the moment it only supports Linux
due to directly launching kube-proxy, but in theory Windows support should be
possible at some point

## Setup
Until I get around to make a debian package or something, there are some things
left to do manually:
* You need `etcd`, `hyperkube` and the default CNI plugins. The easiest way to get them is to use the [microkube-deps](https://github.com/uubk/microkube-deps) repo and invoking `./build.sh`. This will build kubernetes, so you'll require about 15 GB of free disk space
* If you want to run tests or run microkube from the repository, create a folder `third_party` in the repository root and copy all binaries there
* If you're only interested in running `microkubed` from the command line, you can also specify the folder with the binaries as `-extra-bin-dir`
* Since there is no `Makefile` right now, `go build github.com/uubk/microkube/cmd/microkubed` will give you the main binary
* Running it requires `pkexec` from Polkit (for obtaining root for `kube-proxy` and `kubelet`) and `conntrack` + `iptables` for `kube-proxy`
* Unittests additionally require the `openssl` command line utility
* Regenerating the log parser requires [ldetool](https://github.com/sirkon/ldetool)

## License
Apache 2.0