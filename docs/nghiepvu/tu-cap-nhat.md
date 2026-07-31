# Nghiệp vụ: Tự cập nhật (tu-cap-nhat)

> Slug: `tu-cap-nhat` · Label: `nghiepvu:tu-cap-nhat`
> ALLOWED: internal/updater/** internal/version/** third_party/minio-selfupdate/** docs/nghiepvu/tu-cap-nhat.md

## Mô tả

Engine tự cập nhật của ứng dụng: tra cứu version từ nhiều nguồn, resolve artifact theo
hệ điều hành/kiến trúc, tải về an toàn, kiểm tra toàn vẹn, thay binary nguyên tử và hỗ trợ
rollback. Đây là nghiệp vụ lõi của template.

## Thư mục / file liên quan

- `internal/updater/` — toàn bộ engine:
  - `updater.go` — orchestrator `Engine.Update`, hợp đồng exit code (0/10/20/30/40/50).
  - `source.go`, `source_github.go`, `source_azure.go`, `source_http.go` — interface `Source` và 3 nguồn.
  - `resolver.go` — chọn artifact theo `runtime.GOOS/GOARCH`, version, source.
  - `download.go` — HTTP Range resume, retry backoff, `.part`, SHA-256 + chữ ký Ed25519.
  - `apply.go` — swap binary nguyên tử + rollback (kết hợp `third_party/minio-selfupdate/`).
  - `cache.go`, `metadata_cache.go` — cache binary LRU, metadata 15 phút có ETag.
  - `manifest.go` — parse + verify manifest policy (channel, blocked, rollout, sources).
- `internal/version/` — version `1.YY.MMDD.HHmm`, tag `v1.YY.MDDHHmm`, so sánh.
- `third_party/minio-selfupdate/` — vendored atomic apply/rollback.

## Quy tắc bắt buộc

- KHÔNG bao giờ nhét token vào binary phát hành; kênh `internal` chỉ hoạt động khi có
  `APP_UPDATE_TOKEN`.
- Verify thứ tự: SHA-256 trước, Ed25519 (minisign) sau.
- Một nguồn chết → vẫn hoạt động với nguồn còn lại (fail source, không fail cả engine).
- File sửa đổi trong phạm vi nghiệp vụ này nằm dưới `internal/updater/`, `internal/version/`,
  `third_party/minio-selfupdate/` và test tương ứng (`*_test.go`).
- Mọi thay đổi phải kèm test; chạy `go test ./internal/updater/ ./internal/version/`.
