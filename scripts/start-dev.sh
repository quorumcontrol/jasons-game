#!/usr/bin/env bash

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

  local val=$(${command} | tee logs/${log_filename} | grep ${var_name} | cut -d '=' -f 2- | tr -C -d '[:print:]')

  echo ${val}
}

echo ''
echo 'DEV: Starting localnet'
echo ''
make localnet 2>&1 >logs/localnet.log &
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
echo "DEV: Starting a new inkwell service for ${INK_DID}:ink"
echo ''
make inkwell INK_DID=${INK_DID} 2>&1 >logs/inkwell.log &
sleep 15 # give inkwell time to bootstrap
INKWELL_DID=$(capture_value "make inkwelldid" "inkwelldid.log" "INKWELL_DID")

if [[ -z "${INKWELL_DID}" ]]; then
  echo 'DEV: Error capturing INKWELL_DID. Aborting.'
  exit 1
fi

echo ''
echo "DEV: Sending some devink to the ${INKWELL_DID} inkwell"
echo ''
TOKEN_PAYLOAD=$(capture_value "make devink INKWELL_DID=${INKWELL_DID}" "devink_send.log" "TOKEN_PAYLOAD")

if [[ -z "${TOKEN_PAYLOAD}" ]]; then
  echo 'DEV: Error sending devink to inkwell. Aborting.'
  exit 1
fi

echo ''
echo "DEV: Depositing devink in inkwell: ${TOKEN_PAYLOAD}"
echo ''
make inkwell TOKEN_PAYLOAD=${TOKEN_PAYLOAD} 2>&1 >logs/inkwell_deposit.log

tail -f logs/localnet.log
