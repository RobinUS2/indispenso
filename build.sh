#!/bin/bash
export GOPATH=`pwd`
go get -u "github.com/julienschmidt/httprouter"
go get -u "github.com/RobinUS2/golang-jresp"
go build .
