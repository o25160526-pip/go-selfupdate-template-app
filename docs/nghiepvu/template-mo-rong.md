# Nghiệp vụ: Template và mở rộng (template-mo-rong)

> Slug: `template-mo-rong` · Label: `nghiepvu:template-mo-rong`
> ALLOWED: internal/features/** cmd/newfeature/** cmd/templateinit/** cmd/app/features_gen.go scripts/init-template.sh Makefile docs/ADDING_A_FEATURE.md docs/nghiepvu/template-mo-rong.md

## Mô tả

Cơ chế làm repo này thành template tái dùng được: registry feature, generator sinh feature
mới, script khởi tạo template từ repo mẫu. Người dùng clone → đổi tên → chạy `make init` →
cắm feature mới mà không phải sửa `main.go`.

## Thư mục / file liên quan

- `internal/features/` — interface `Feature` (`ID`, `TrayItems`, `Commands`, `Start`),
  `Register`, `All`; feature mẫu `sample`, `healthcheck`.
- `cmd/app/features_gen.go` — blank-import các feature (file sinh, không sửa tay).
- `cmd/newfeature/` — generator sinh package + struct + test + registry.
- `cmd/templateinit/`, `scripts/init-template.sh` — khởi tạo repo mới từ template
  (`make init APP=... MODULE=...`).
- `docs/ADDING_A_FEATURE.md` — hướng dẫn thêm feature bằng generator.
- `Makefile` — target `init`, `new-feature`.

## Quy tắc bắt buộc

- Feature mới = tạo package trong `internal/features/<id>` + `features.Register(&X{})` trong
  `init()`; KHÔNG sửa `main.go`.
- Không đăng ký trùng ID (panic ở `Register`).
- Build mặc định headless; adapter tray phải qua build tag `tray`.
- File sửa đổi trong phạm vi: `internal/features/`, `cmd/newfeature/`, `cmd/templateinit/`,
  `cmd/app/features_gen.go`, `scripts/init-template.sh`, `Makefile`, `docs/ADDING_A_FEATURE.md`.
