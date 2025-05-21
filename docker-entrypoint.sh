#!/bin/sh
set -e

exec /app/node \
    -cfg-path="$CONFIG_PATH" \
    -wallet-path="$WALLET_PATH" \
    -wallet-password="$WALLET_PASSWORD" \
    -state-path="$STATE_PATH" \
    -bind-address="$BIND_ADDR" \
    -api-address="$API_ADDR" \
    -grpc-address="$GRPC_ADDR" \
    -build-extended-api \
    -serve-extended-api \
    -build-state-hashes \
    -enable-grpc-api \
    "$@"