#!/usr/bin/env sh
set -eu
APP=${APP:-${1:-}}
MODULE=${MODULE:-${2:-}}
[ -n "$APP" ] || { echo "APP is required" >&2; exit 2; }
[ -n "$MODULE" ] || { echo "MODULE is required" >&2; exit 2; }
exec go run ./cmd/templateinit --app "$APP" --module "$MODULE"
