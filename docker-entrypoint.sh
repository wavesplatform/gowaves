#!/bin/sh
set -e

# If no arguments provided, use defaults from CMD
if [ $# -eq 0 ]; then
  exec /app/node "$@"
else
  # Merge CMD + user-provided args
  exec /app/node "$@" "-state-path=/home/gowaves/state" "-bind-address=0.0.0.0:6868" "-api-address=0.0.0.0:6869" "-build-extended-api" "-serve-extended-api" "-build-state-hashes" "-enable-grpc-api" "-grpc-address=0.0.0.0:7470"
fi
