#!/bin/bash
export GOPATH=`pwd`
go get -u "github.com/julienschmidt/httprouter"
go build .
