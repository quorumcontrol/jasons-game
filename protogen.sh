#!/usr/bin/env bash

mkdir -p ./pb/jasonsgame
mkdir -p ./frontend/jasons-game/src/js/frontend/remote

protoc -I=./pb -I=$GOPATH/src -I=$GOPATH/src/github.com/gogo/protobuf/protobuf jasonsgame.proto \
--plugin=protoc-gen-ts=./frontend/jasons-game/node_modules/.bin/protoc-gen-ts \
--ts_out=service=true:./frontend/jasons-game/src/js/frontend/remote \
--js_out=import_style=commonjs,binary:./frontend/jasons-game/src/js/frontend/remote \
--plugin=protoc-gen-go=${GOPATH}/bin/protoc-gen-go \
--grpc-web_out=import_style=commonjs,mode=binary:./frontend/jasons-game/src/js/frontend/remote \
--gogofaster_out=plugins=grpc:./pb/jasonsgame