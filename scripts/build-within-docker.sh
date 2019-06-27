#!/bin/bash
set -eo pipefail

mkdir -p ~/.ssh
git config --global url."ssh://git@github.com/".insteadOf "https://github.com/"
ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts

if [[ "${CI}" == "true" ]]; then
  sudo ./scripts/install-protoc.sh
  sudo ./scripts/install-node.sh
fi

export GOPROXY='https://proxy.golang.org'
go mod download github.com/dgraph-io/badger/v2@v2.0.0-rc2
unset GOPROXY
go mod download 

make lint
if [[ "${CI}" == "true" ]]; then
  make ci-test
else
  make test
fi
