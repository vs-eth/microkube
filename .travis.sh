#!/bin/bash

set -e


if [ "${PACKAGE}" = "Yes" ] ; then
    # Install packaging dependencies
    sudo apt update && sudo apt install -y -q rsync golang-go binutils dpkg-dev build-essential fakeroot debhelper dh-golang
fi

# Do normal test build
go test -race -coverprofile=cover1 -covermode=atomic -run 'Test[^9]' -v ./...
sudo killall etcd || /bin/true
sudo killall hyperkube || /bin/true
sudo rm -Rf /tmp/microkube* || /bin/true
go test -coverprofile=cover2 -covermode=atomic -run 'Test9' -v ./...
cat cover1 cover2 > coverage.txt
bash <(curl -s https://codecov.io/bash)

if [ "${PACKAGE}" = "Yes" ] ; then
    # Do additional package build
    dpkg-buildpackage -us -uc -b
fi