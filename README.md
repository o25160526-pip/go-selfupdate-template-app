# Go Self-Update Template App

A reusable Go repository template for a six-target desktop/headless application with dual-source self-update, signed policy manifest, staged rollout, rollback, cache/prefetch, feature registry, TUI-style menu, tray build tag, and release gates.

## What is implemented

- Display version `1.YY.MMDD.HHmm` and SemVer-compatible Git tag `v1.YY.MDDHHmm`.
- UTC version generation with collision avoidance when a tag already exists.
- GitHub Releases source and Azure Blob `index.json` source queried concurrently.
- Stable, beta, and internal channels. Internal draft access requires `APP_UPDATE_TOKEN`.
- Exact OS/architecture asset selection for Windows, Linux, and macOS on amd64/arm64.
- Resumable downloads (`Range`), exponential retry, `.part` files, state file, SHA-256, minisign-format Ed25519 verification (with raw-signature backward compatibility), and proxy support through Go's default transport.
- Signed manifest policy: `force_update`, `min_supported`, blocked versions, stable rollout bucket, source order, expiry, and two embedded public keys for rotation.
- Atomic replacement via a vendored, source-compatible subset of `github.com/minio/selfupdate` v0.6.0, persistent local rollback, and an update lock.
- Binary blob cache with LRU pruning plus a 15-minute HTTP metadata cache with ETag revalidation; prefetch only considers versions newer than the running binary.
- Feature registry and `make new-feature NAME=...` generator without changing `main.go`.
- GitHub Actions and Azure Pipelines definitions, six release targets, real upgrade/rollback/upgrade smoke test, draft promotion gate, and signed release manifest.

## Quick start

```bash
go test ./...
./scripts/e2e-local.sh
make build
./dist/app version --json
```

Initialize a copied template:

```bash
make init APP=myapp MODULE=github.com/acme/myapp
make new-feature NAME=diagnostics
go test ./...
```

## Configuration

Priority is:

```text
CLI flag > APP_* environment > config file > manifest policy > defaults
```

Manifest `force_update` and `blocked` always override local preferences. The default file is `~/.config/app/config.yaml`. See `configs/config.example.yaml`.

Common environment variables:

```text
APP_CHANNEL=stable|beta|internal
APP_UPDATE_TOKEN=...             # mandatory for internal
APP_GITHUB_OWNER=your-org
APP_GITHUB_REPO=your-app
APP_GITHUB_API=https://api.github.com
APP_AZURE_INDEX_URL=https://.../index.json
APP_MANIFEST_URL=https://.../manifest.json
APP_PUBLIC_KEYS=currentBase64,nextBase64
APP_CACHE_DIR=~/.cache/app
APP_TIMEOUT=5m
```

## Commands

```text
app version [--json]
app update [--latest] [--version X] [--silent] [--list] [--dry-run] [--channel ...] [--timeout 5m]
app rollback [--target path]
app channel set stable|beta|internal
app cache list
app cache prune --keep 3
app cache prefetch --keep 3
app config show
app menu
app tray
app features
```

Silent update exit contract:

| Code | Meaning |
|---:|---|
| 0 | Updated successfully |
| 10 | Already up to date |
| 20 | Version/asset not found |
| 30 | Checksum, signature, or policy verification failed |
| 40 | Apply failed or rollback path failed |
| 50 | Every update source was unavailable |

## Release setup

Generate a signing pair:

```bash
go run ./cmd/keygen
```

Store the printed values as GitHub/Azure secrets or variables:

- `APP_BINARY_PRIVATE_KEY`: base64 Ed25519 private key used to emit four-line minisign signatures.
- `APP_MANIFEST_PRIVATE_KEY`: base64 Ed25519 private key; it may be the same rotation pair.
- `APP_CURRENT_PUBLIC_KEY`: base64 minisign public-key payload (raw 32-byte Ed25519 keys remain accepted for compatibility).
- `APP_NEXT_PUBLIC_KEY`: next rotation key; generate and embed it from the first release.
- `APP_MANIFEST_URL`: repository variable pointing to the separately hosted signed manifest.
- Azure additionally needs `AZURE_PUBLIC_BASE_URL`, `AZURE_SERVICE_CONNECTION`, `AZURE_STORAGE_ACCOUNT`, and `AZURE_CONTAINER`.

The release workflow derives a deterministic UTC version from the commit timestamp, creates a draft, executes upgrade → rollback → upgrade on Linux, Windows, and macOS, signs the policy manifest, then publishes the draft. The first release uses the local real-binary E2E path because no previous published binary exists.

## Desktop adapters

The default build is headless (`!tray`). `-tags tray` enables the tray adapter interface without forcing Linux CGO packages into server builds. The shipped adapter is intentionally dependency-free so this repository can be built and verified offline; `docs/DESKTOP_ADAPTERS.md` describes swapping it for `fyne.io/systray` and the menu for Bubble Tea when those GUI dependencies are available.

## Verification artifacts

- `reports/TEST_CASE_REPORT.md`
- `reports/IMPLEMENTATION_REVIEW.md`
- `scripts/audit-plan.sh`
