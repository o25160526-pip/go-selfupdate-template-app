#!/usr/bin/env sh
set -eu
OUT=${1:-dist/matrix}
VERSION=${VERSION:-1.26.0729.1930}
COMMIT=${COMMIT:-local}
BUILD_DATE=${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}
BINARY_NAME=${APP_BINARY_NAME:-app}
mkdir -p "$OUT"
for OS in windows linux darwin; do
  for ARCH in amd64 arm64; do
    EXT=""; [ "$OS" != windows ] || EXT=.exe
    echo "build $OS/$ARCH"
    CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build -trimpath -ldflags "-s -w -X github.com/your-org/go-selfupdate-template/internal/version.Display=$VERSION -X github.com/your-org/go-selfupdate-template/internal/version.Commit=$COMMIT -X github.com/your-org/go-selfupdate-template/internal/version.BuildDate=$BUILD_DATE" -o "$OUT/${BINARY_NAME}_${OS}_${ARCH}${EXT}" ./cmd/app
  done
done
