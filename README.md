# Go Self-Update Template App

Template ứng dụng Go headless/desktop có cơ chế tự cập nhật an toàn, hỗ trợ GitHub Releases và Azure Blob Storage. Dự án tập trung vào kiểm tra tính toàn vẹn, chữ ký, chính sách phát hành, rollout từng phần, cache, rollback và smoke test khi release.

[![Build](https://github.com/o25160526-pip/go-selfupdate-template-app/actions/workflows/build.yml/badge.svg)](https://github.com/o25160526-pip/go-selfupdate-template-app/actions/workflows/build.yml)

## Mục lục

- [Tính năng](#tính-năng)
- [Kiến trúc](#kiến-trúc)
- [Cài đặt nhanh](#cài-đặt-nhanh)
- [Cấu hình](#cấu-hình)
- [Các lệnh](#các-lệnh)
- [Tài liệu](#tài-liệu)

## Tính năng

- Version build theo UTC: `1.YY.MMDD.HHmm`; tag release tương ứng là `v1.YY.MDDHHmm`.
- Tra cứu đồng thời GitHub Releases và Azure Blob `index.json`.
- Kênh `stable`, `beta`, `internal`; kênh internal yêu cầu `APP_UPDATE_TOKEN`.
- Chọn artifact chính xác theo hệ điều hành và kiến trúc Windows, Linux, macOS trên amd64/arm64.
- Tải tiếp tục bằng HTTP Range, retry exponential backoff, file `.part`, SHA-256 và chữ ký Ed25519 dạng minisign.
- Manifest có chữ ký, giới hạn version, blocked versions, rollout, thứ tự source và thời hạn.
- Thay binary nguyên tử, lưu rollback và khóa update.
- Cache binary LRU, cache metadata HTTP 15 phút với ETag và prefetch version mới hơn bản đang chạy.
- Build headless mặc định; build tag `tray` bật adapter tray không phụ thuộc GUI.

## Kiến trúc

`cmd/app` là entrypoint CLI. `internal/app` xử lý command và cấu hình; `internal/version` parse/so sánh version; `internal/updater` chứa source, resolver, manifest policy, downloader, cache và apply/rollback; `internal/signing` xử lý public key; `internal/buildinfo` nhận metadata qua ldflags. CI dùng GitHub Actions để test, build sáu target, ký artifact, tạo draft release, smoke test upgrade/rollback và publish.

## Cài đặt nhanh

```bash
git clone https://github.com/o25160526-pip/go-selfupdate-template-app.git
cd go-selfupdate-template-app
go test ./...
make build
./dist/app version --json
```

Chạy E2E local:

```bash
./scripts/e2e-local.sh
```

## Cấu hình

Thứ tự ưu tiên là `CLI flag > APP_* environment > config file > manifest policy > defaults`. File mặc định là `~/.config/app/config.yaml`; file mẫu ở [`configs/config.example.yaml`](configs/config.example.yaml).

```text
APP_CHANNEL=stable|beta|internal
APP_UPDATE_TOKEN=...
APP_GITHUB_OWNER=your-org
APP_GITHUB_REPO=your-app
APP_GITHUB_API=https://api.github.com
APP_AZURE_INDEX_URL=https://.../index.json
APP_MANIFEST_URL=https://.../manifest.json
APP_PUBLIC_KEYS=currentBase64,nextBase64
APP_CACHE_DIR=~/.cache/app
APP_TIMEOUT=5m
```

## Các lệnh

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

Khi dùng `update --silent`, mã thoát là: `0` cập nhật thành công, `10` đã mới nhất, `20` không tìm thấy version/artifact, `30` lỗi checksum/chữ ký/policy, `40` lỗi apply/rollback, `50` tất cả source không khả dụng.

## Tài liệu

- [`docs/SPEC.md`](docs/SPEC.md): đặc tả kỹ thuật và format manifest.
- [`docs/DEPLOY.md`](docs/DEPLOY.md): build, release và CI/CD.
- [`docs/AZURE_SETUP.md`](docs/AZURE_SETUP.md): thiết lập Azure DevOps Artifacts một lần (UI hoặc REST API).
- [`docs/SETUP.md`](docs/SETUP.md): môi trường phát triển.
- [`docs/EXAMPLES.md`](docs/EXAMPLES.md): ví dụ chạy app và self-update.
- [`docs/DESKTOP_ADAPTERS.md`](docs/DESKTOP_ADAPTERS.md): adapter desktop.
- [`docs/ADDING_A_FEATURE.md`](docs/ADDING_A_FEATURE.md): thêm feature bằng generator.
- [`CHANGELOG.md`](CHANGELOG.md): lịch sử thay đổi.
