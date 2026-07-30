#!/usr/bin/env sh
set -eu
export TZ=UTC
exec go run ./cmd/versiongen --format="${1:-both}"
