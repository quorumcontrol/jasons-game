#!/usr/bin/env bash

mkdir -p ./pb/books

protoc -I=./pb books.proto \
--js_out=import_style=commonjs:./frontend/jasons-game/src/js/frontend \
--plugin=protoc-gen-go=${GOPATH}/bin/protoc-gen-go \
--grpc-web_out=import_style=commonjs,mode=grpcwebtext:./frontend/jasons-game/src/js/frontend \
--go_out=plugins=grpc:./pb/books