#!/usr/bin/env bash

set -e

pushd $(dirname $0)/../
trap popd EXIT

mkdir -p ./pb/jasonsgame
mkdir -p ./frontend/jasons-game/src/js/frontend/remote

protoc -I=./pb -I=$GOPATH/src -I=${GOPATH}/src/github.com/gogo/protobuf/protobuf jasonsgame.proto \
--plugin=protoc-gen-ts=./frontend/jasons-game/node_modules/.bin/protoc-gen-ts \
--ts_out=service=true:./frontend/jasons-game/src/js/frontend/remote \
--js_out=import_style=commonjs,binary:./frontend/jasons-game/src/js/frontend/remote \
--gogofaster_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,plugins=grpc:./pb/jasonsgame

protoc -I=./network -I=$GOPATH/src -I=${GOPATH}/src/github.com/gogo/protobuf/protobuf messages.proto --gogofaster_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,plugins=grpc:./network/

protoc -I=./game -I=$GOPATH/src types.proto --gogofaster_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,plugins=grpc:./game/

protoc -I=./inkfaucet/inkfaucet -I=$GOPATH/src messages.proto --gogofaster_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,plugins=grpc:./inkfaucet/inkfaucet
