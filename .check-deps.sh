#!/bin/bash

if [ ! -f "$GOPATH/bin/ldetool" ] ; then
    echo "ldetool missing, install? (CTRL+C to cancel)"
    read TESTVAR
    go get -u github.com/sirkon/ldetool
fi