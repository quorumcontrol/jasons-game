#!/usr/bin/env bash

set -e

pushd $(dirname $0)/../
trap popd EXIT

mkdir -p ./pb/jasonsgame
mkdir -p ./frontend/jasons-game/src/js/frontend/remote

protoc -I=./pb -I=$GOPATH/src -I=${GOPATH}/src/github.com/gogo/protobuf/protobuf jasonsgame.proto \
--plugin=protoc-gen-ts=./node_modules/.bin/protoc-gen-ts \
--ts_out=service=true:./frontend/jasons-game/src/js/frontend/remote \
--js_out=import_style=commonjs_strict,binary:./frontend/jasons-game/src/js/frontend/remote \
--gogofaster_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,plugins=grpc:./pb/jasonsgame

# TODO: Move this into protoc-gen-ts itself and PR it upstream.
# This is needed to make the generated service code support the commonjs_strict import_style
# specified above. Otherwise it assumes import_style=commonjs, but that generates code
# that requires 'unsafe-eval' in your Content-Security-Policy, which is less bueno.
sed_cmd=("sed" "-i")
if [[ $(uname) == "Darwin" ]]; then
  sed_cmd=("${sed_cmd[@]}" '')
fi
"${sed_cmd[@]}" -e 's/jasonsgame_pb\./jasonsgame_pb.jasonsgame./g' frontend/jasons-game/src/js/frontend/remote/jasonsgame_pb_service.*

protoc -I=./network -I=$GOPATH/src -I=${GOPATH}/src/github.com/gogo/protobuf/protobuf messages.proto --gogofaster_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,plugins=grpc:./network/

protoc -I=./game -I=$GOPATH/src types.proto --gogofaster_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,plugins=grpc:./game/

protoc -I=./inkfaucet/inkfaucet -I=$GOPATH/src messages.proto --gogofaster_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,plugins=grpc:./inkfaucet/inkfaucet
