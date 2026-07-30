#!/usr/bin/env sh
# Load CI signing values without overwriting real secrets already supplied by the runner.
set -eu
KEY_FILE=${APP_CI_KEY_FILE:-ci/keys/dev.env}
[ -f "$KEY_FILE" ] || exit 0
read_value() { sed -n "s/^$1=//p" "$KEY_FILE" | head -n 1; }
: "${APP_BINARY_PRIVATE_KEY:=$(read_value APP_BINARY_PRIVATE_KEY)}"
: "${APP_MANIFEST_PRIVATE_KEY:=$(read_value APP_MANIFEST_PRIVATE_KEY)}"
: "${APP_CURRENT_PUBLIC_KEY:=$(read_value APP_CURRENT_PUBLIC_KEY)}"
: "${APP_NEXT_PUBLIC_KEY:=$(read_value APP_NEXT_PUBLIC_KEY)}"
export APP_BINARY_PRIVATE_KEY APP_MANIFEST_PRIVATE_KEY APP_CURRENT_PUBLIC_KEY APP_NEXT_PUBLIC_KEY
