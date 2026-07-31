# Chỉ mục nghiệp vụ — go-selfupdate-template-app

> File này được dùng làm nguồn để người tạo issue chọn đúng nghiệp vụ (label `nghiepvu:<slug>`).
> Cách dùng: mở đúng file `<slug>.md` bên dưới để biết phạm vi file được phép sửa và quy tắc.
> Quy trình vận hành agent AI trên issue: xem [README-opencode-workflow.md](README-opencode-workflow.md).
> Spec gốc đã triển khai hệ thống này: [PROMPT-GOC.md](PROMPT-GOC.md).

| Slug | Nghiệp vụ | Mô tả | Thư mục chính |
| --- | --- | --- | --- |
| [`tu-cap-nhat`](tu-cap-nhat.md) | Tự cập nhật | Engine cập nhật: resolve, download, verify, apply, rollback | `internal/updater/`, `internal/version/` |
| [`phat-hanh-ci`](phat-hanh-ci.md) | Phát hành và CI/CD | Workflow build/release, smoke test, publish | `.github/workflows/`, `azure-pipelines.yml`, `scripts/` |
| [`an-toan-chu-ky`](an-toan-chu-ky.md) | Chữ ký và toàn vẹn | Ed25519/minisign, quản lý khoá, ký binary & manifest | `internal/signing/`, `cmd/*sign*/`, `cmd/keygen/` |
| [`cli-giao-dien`](cli-giao-dien.md) | CLI, cấu hình, giao diện | Lệnh, config, TUI, tray, exit code | `cmd/app/`, `internal/app/`, `internal/config/`, `internal/ui/`, `internal/tray/` |
| [`template-mo-rong`](template-mo-rong.md) | Template và mở rộng | Feature registry, generator, init template | `internal/features/`, `cmd/newfeature/`, `cmd/templateinit/` |
| [`khao-sat-nghiep-vu`](khao-sat-nghiep-vu.md) | Khảo sát và chuẩn hóa nghiệp vụ | Quét codebase tìm logic nghiệp vụ mới, chuẩn hóa docs/template | `docs/nghiepvu/`, `.github/ISSUE_TEMPLATE/` |

## Mẹo chọn nghiệp vụ

- Sửa **cơ chế tự cập nhật** (source, cache, apply, manifest policy) → `tu-cap-nhat`.
- Sửa **quy trình phát hành** (workflow, script CI, sinh version) → `phat-hanh-ci`.
- Sửa **khoá/chữ ký/verify** → `an-toan-chu-ky`.
- Sửa **lệnh, cấu hình, menu, tray** → `cli-giao-dien`.
- **Thêm feature mới / làm template** → `template-mo-rong`.
- Không chắc chắn → chọn nghiệp vụ gần nhất, agent sẽ hỏi lại nếu phạm vi không khớp.
