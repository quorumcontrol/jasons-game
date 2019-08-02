#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

function cleanup () {
  # in theory we shouldn't need to delete this state every time; but I haven't gotten it working reliably o/w
  rm -rf devdocker/devink_state/*
  make down
}

trap cleanup EXIT

function capture_value () {
  local command=$1
  local log_filename=$2
  local var_name=$3
  local val

  IFS=$' '
  val=$(${command} | tee logs/"${log_filename}" | grep -e "${var_name}=\S" | cut -d '=' -f 2- | tr -C -d '[:print:]')
  IFS=$'\n\t'

  echo "${val}"
}

# cleanup any log files before we start
rm -f logs/*

echo ''
echo 'DEV: Starting localnet'
echo ''
make localnet >logs/localnet.log 2>&1 &
sleep 15 # give localnet some time to come up; sleeps are gross; loop until port 34001 is open instead?

echo ''
echo 'DEV: Bootstrapping devink'
echo ''
INK_DID=$(capture_value "make devink" "devink_bootstrap.log" "INK_DID")

if [[ -z "${INK_DID}" ]]; then
  echo 'DEV: Error bootstrapping devink. Aborting.'
  exit 1
fi

echo ''
echo "DEV: Starting a new inkfaucet service for ${INK_DID}:ink"
echo ''
INK_FAUCET_KEY=$(capture_value "make inkfaucet INK_DID=${INK_DID}" "inkfaucetkey.log" "INK_FAUCET_KEY")

if [[ -z "${INK_FAUCET_KEY}" ]]; then
  echo 'DEV: Error capturing INK_FAUCET_KEY. Aborting.'
  exit 1
fi

echo ''
echo "DEV: Ink faucet key: ${INK_FAUCET_KEY}"
echo ''

make inkfaucet INK_DID=${INK_DID} INK_FAUCET_KEY=${INK_FAUCET_KEY} >logs/inkfaucet.log 2>&1 &
sleep 30 # give inkfaucet time to bootstrap

echo ''
echo "DEV: Sending some devink to the inkfaucet"
echo ''
TOKEN_PAYLOAD=$(capture_value "make devink INK_FAUCET_KEY=${INK_FAUCET_KEY}" "devink_send.log" "TOKEN_PAYLOAD")

if [[ -z "${TOKEN_PAYLOAD}" ]]; then
  echo 'DEV: Error sending devink to inkfaucet. Aborting.'
  exit 1
fi

echo ''
echo "DEV: Depositing devink in inkfaucet"
echo ''
make inkfaucet TOKEN_PAYLOAD=${TOKEN_PAYLOAD} INK_FAUCET_KEY=${INK_FAUCET_KEY} >logs/inkfaucet_deposit.log 2>&1

echo ''
echo "DEV: Starting game"
echo ''
make game-server INK_DID=${INK_DID}
