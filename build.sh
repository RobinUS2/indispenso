#!/bin/bash
export GOPATH=`pwd`
go get -u "github.com/julienschmidt/httprouter"
go get -u "github.com/RobinUS2/golang-jresp"
go get -u "github.com/nu7hatch/gouuid"
go get -u "github.com/antonholmquist/jason"
go build .
