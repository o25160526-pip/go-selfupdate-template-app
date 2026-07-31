# Nghiệp vụ: Chữ ký và toàn vẹn (an-toan-chu-ky)

> Slug: `an-toan-chu-ky` · Label: `nghiepvu:an-toan-chu-ky`
> ALLOWED: internal/signing/** cmd/keygen/** cmd/binarysign/** cmd/manifestsign/** internal/buildinfo/** configs/** scripts/ci-keys.sh docs/nghiepvu/an-toan-chu-ky.md

## Mô tả

Hạ tầng ký/kiểm tra chữ ký Ed25519 theo định dạng minisign (không prehash), quản lý public
key (current + next để xoay khoá không làm gãy client cũ), ký binary, ký manifest policy và
sinh cặp khoá dùng trong CI.

## Thư mục / file liên quan

- `internal/signing/minisign.go` — wire format minisign: `EncodePublicKey`, `ParsePublicKey`,
  `Sign`, `Verify`, `DecodePrivateKey` (seed 32B hoặc expanded 64B).
- `cmd/binarysign/` — ký binary release.
- `cmd/manifestsign/` — ký `manifest.json` trước khi upload.
- `cmd/keygen/` — sinh cặp khoá.
- `internal/buildinfo/` — nhúng `CurrentPublicKey`/`NextPublicKey`/`DefaultManifestURL` qua ldflags.
- `scripts/ci-keys.sh` — fallback key cho local/CI không production.
- `configs/manifest.example.json` — mẫu manifest policy (channel, blocked, rollout, signature).

## Quy tắc bắt buộc

- Không commit private key thật; luôn dùng GitHub secret/Azure pipeline variable.
- Nhúng 2 public key (current + next) từ đầu; không có điều này thì mất private key = brick client.
- Verify SHA-256 trước, chữ ký Ed25519 sau (xem `internal/updater/download.go`).
- Manifest policy dùng `blocked` và `force_update` làm kill switch — luôn thắng config local.
- File sửa đổi trong phạm vi: `internal/signing/`, `cmd/keygen/`, `cmd/binarysign/`,
  `cmd/manifestsign/`, `internal/buildinfo/`, `configs/`, `scripts/ci-keys.sh`.
