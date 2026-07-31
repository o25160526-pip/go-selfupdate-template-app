# Nghiệp vụ: CLI, cấu hình và giao diện (cli-giao-dien)

> Slug: `cli-giao-dien` · Label: `nghiepvu:cli-giao-dien`
> ALLOWED: cmd/app/** internal/app/** internal/config/** internal/ui/** internal/tray/** internal/buildinfo/** docs/nghiepvu/cli-giao-dien.md

## Mô tả

Entrypoint CLI, xử lý lệnh, cấu hình và các giao diện người dùng: menu tương tác (TUI),
tray icon (build tag `tray`), output `--json` và hợp đồng exit code. Đây là lớp mỏng nối
user với engine `tu-cap-nhat`.

## Thư mục / file liên quan

- `cmd/app/main.go` — entrypoint; chỉ gọi `app.Run`.
- `internal/app/app.go` — command handling, wire config → updater, kênh `stable|beta|internal`.
- `internal/config/config.go` — thứ tự ưu tiên: CLI flag > `APP_*` env > config file > manifest policy.
- `internal/ui/menu.go` — TUI chọn version theo nguồn/cache.
- `internal/tray/` — adapter tray (`tray`/`notray` build tags).
- `internal/buildinfo/` — metadata build (owner, repo, manifest URL, keys).

## Quy tắc bắt buộc

- `--silent` phải không TTY, không prompt, log ra stderr + file; exit code chuẩn
  (0/10/20/30/40/50 — xem `internal/updater/updater.go`).
- `--list`, `--dry-run`, `--version X`, `--channel`, `--timeout` phải luôn tồn tại.
- Kênh `internal` không có token → lỗi rõ ràng, không im lặng fallback về stable.
- Thứ tự ưu tiên config bất biến; `force_update`/`blocked` trong manifest luôn thắng.
- File sửa đổi trong phạm vi: `cmd/app/`, `internal/app/`, `internal/config/`, `internal/ui/`,
  `internal/tray/`, `internal/buildinfo/`.
