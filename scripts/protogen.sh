#!/usr/bin/env bash

set -e

pushd $(dirname $0)/../
trap popd EXIT

mkdir -p ./pb/jasonsgame
mkdir -p ./frontend/jasons-game/src/js/frontend/remote

protoc -I=./pb -I=$GOPATH/src jasonsgame.proto \
--plugin=protoc-gen-ts=./frontend/jasons-game/node_modules/.bin/protoc-gen-ts \
--ts_out=service=true:./frontend/jasons-game/src/js/frontend/remote \
--js_out=import_style=commonjs,binary:./frontend/jasons-game/src/js/frontend/remote \
--go_out=plugins=grpc:./pb/jasonsgame

protoc -I=./network -I=$GOPATH/src messages.proto --go_out=plugins=grpc:./network/

protoc -I=./game/trees/ -I=$GOPATH/src types.proto --go_out=./game/trees/ && protoc-go-inject-tag -input=./game/trees/types.pb.go -XXX_skip=cbor,refmt