#!/bin/sh
set -e

# Если аргументов нет — запускаем с параметрами по умолчанию из ENV
if [ $# -eq 0 ]; then
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
    -enable-grpc-api
else
  # Если аргументы переданы — используем их
  exec /app/node "$@"
fi