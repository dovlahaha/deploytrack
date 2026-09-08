#!/usr/bin/env bash
# Block until the database accepts TCP connections, or give up.
#
# A fixed `sleep 10` is the usual shortcut here and it is wrong in both
# directions: too short on a slow runner, and wasted seconds on a fast one.
# Polling means the pipeline waits exactly as long as it needs to.
set -euo pipefail

HOST="${1:-localhost}"
PORT="${2:-5432}"
TIMEOUT="${3:-60}"

echo "waiting for ${HOST}:${PORT} (timeout ${TIMEOUT}s)"

for i in $(seq 1 "$TIMEOUT"); do
  if (exec 3<>"/dev/tcp/${HOST}/${PORT}") 2>/dev/null; then
    exec 3>&- 2>/dev/null || true
    echo "database is accepting connections after ${i}s"
    exit 0
  fi
  sleep 1
done

echo "database did not become ready within ${TIMEOUT}s" >&2
exit 1
