#!/usr/bin/env sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
TMP=$(mktemp -d)
SERVER_PID=""
cleanup() { [ -z "$SERVER_PID" ] || kill "$SERVER_PID" 2>/dev/null || true; rm -rf "$TMP"; }
trap cleanup EXIT INT TERM
OLD=1.26.0729.1900
NEW=1.26.0729.2000
TAG=v1.26.7292000
OS=$(go env GOOS)
ARCH=$(go env GOARCH)
BINARY_NAME=${APP_BINARY_NAME:-app}
EXT=""; [ "$OS" != windows ] || EXT=.exe
TARGET="$TMP/app$EXT"
ASSET_NAME="${BINARY_NAME}_${OS}_${ARCH}${EXT}"
ASSET="$TMP/$ASSET_NAME"
cd "$ROOT"
go build -ldflags "-X github.com/your-org/go-selfupdate-template/internal/version.Display=$OLD -X github.com/your-org/go-selfupdate-template/internal/version.Commit=e2e-old" -o "$TARGET" ./cmd/app
go build -ldflags "-X github.com/your-org/go-selfupdate-template/internal/version.Display=$NEW -X github.com/your-org/go-selfupdate-template/internal/version.Commit=e2e-new" -o "$ASSET" ./cmd/app
test -s "$TARGET"
test -s "$ASSET"
printf 'asset=%s\n' "$ASSET_NAME"
printf 'version_before=%s\n' "$($TARGET version)"
go run ./cmd/testupdateserver --asset "$ASSET" --name "$ASSET_NAME" --tag "$TAG" --ready "$TMP/url" >"$TMP/server.log" 2>&1 &
SERVER_PID=$!
for _ in $(seq 1 100); do [ -s "$TMP/url" ] && break; sleep 0.05; done
test -s "$TMP/url"
BASE=$(cat "$TMP/url")
export APP_GITHUB_OWNER=test APP_GITHUB_REPO=app APP_GITHUB_API="$BASE" APP_CHANNEL=internal APP_UPDATE_TOKEN=e2e-token APP_CACHE_DIR="$TMP/cache" APP_LOG_FILE="$TMP/update.log" APP_MANIFEST_URL=
"$TARGET" update --silent --version "$NEW" --timeout 30s
test "$($TARGET version)" = "$NEW"
printf 'version_after=%s\n' "$($TARGET version)"
"$TARGET" rollback --target "$TARGET"
test "$($TARGET version)" = "$OLD"
"$TARGET" update --silent --version "$NEW" --timeout 30s
test "$($TARGET version)" = "$NEW"
echo "E2E PASS [$OS/$ARCH]: $OLD -> $NEW -> rollback -> $OLD -> $NEW"
