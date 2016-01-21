#!/bin/bash
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}
go get "github.com/julienschmidt/httprouter"
go get "github.com/RobinUS2/golang-jresp"
go get "github.com/nu7hatch/gouuid"
go get "github.com/antonholmquist/jason"
go get "github.com/kylelemons/go-gypsy/yaml"
go get "golang.org/x/crypto/bcrypt"
go get "github.com/dgryski/dgoogauth"
go get "github.com/petar/rsc/qr"
go get "github.com/spf13/pflag"
go get "github.com/spf13/viper"
go get "github.com/stretchr/testify/assert"
go get "github.com/jmcvetta/randutil"
go fmt .
go test && go build .