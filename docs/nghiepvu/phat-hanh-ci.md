# Nghiệp vụ: Phát hành và CI/CD (phat-hanh-ci)

> Slug: `phat-hanh-ci` · Label: `nghiepvu:phat-hanh-ci`
> ALLOWED: .github/workflows/** azure-pipelines.yml scripts/** Makefile cmd/versiongen/** .release-trigger renovate.json docs/nghiepvu/phat-hanh-ci.md

## Mô tả

Toàn bộ quy trình build → test → ký → tạo draft release → smoke test upgrade/rollback →
publish trên GitHub Actions và Azure Pipelines. Version được sinh từ commit timestamp UTC,
không tự đặt bằng tay.

## Thư mục / file liên quan

- `.github/workflows/build.yml` — pipeline chính: `version → test → build → draft → smoke-update → promote → post-verify`.
- `azure-pipelines.yml` — pipeline Azure song song (Test, Build, SmokeUpdate, Publish).
- `cmd/versiongen/` — sinh display version + git tag từ commit time (kèm xử lý trùng phút).
- `scripts/audit-plan.sh`, `scripts/e2e-local.sh`, `scripts/build-matrix.sh`, `scripts/ci-keys.sh`.
- `Makefile` — lệnh `test`, `verify`, `build`, `build-matrix`, `e2e`, `init`, `new-feature`.
- `.release-trigger`, `renovate.json` — cơ chế trigger/xoay vòng phụ trợ.

## Quy tắc bắt buộc

- Version/tag sinh trong CI bằng `cmd/versiongen` với `TZ=UTC`; KHÔNG tự tạo tag tay.
- Tag tạo xong phải có release đi kèm (không cancel pipeline giữa chừng —
  `cancel-in-progress: false`).
- Smoke test upgrade-path thật (vN-1 → vN → rollback → vN) là gate publish, không mock.
- Push vào `main` mới kích hoạt publish; nhánh khác chỉ test + build.
- File sửa đổi trong phạm vi: workflow YAML, `scripts/`, `Makefile`, `cmd/versiongen/`,
  `.release-trigger`, `renovate.json`.
