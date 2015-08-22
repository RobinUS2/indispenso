#!/bin/bash
export GOPATH=`pwd`
go get "github.com/julienschmidt/httprouter"
go get "github.com/RobinUS2/golang-jresp"
go get "github.com/nu7hatch/gouuid"
go get "github.com/antonholmquist/jason"
go get "github.com/kylelemons/go-gypsy/yaml"
go build .
