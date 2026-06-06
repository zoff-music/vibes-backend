#!/usr/bin/env sh
set -eu

/app/migrator-main
exec /app/main
