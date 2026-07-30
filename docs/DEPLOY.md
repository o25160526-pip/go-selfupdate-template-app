# Triển khai và phát hành

## Build local

Yêu cầu Go 1.23 trở lên, Git và Make tùy chọn.

```bash
go test ./...
go vet ./...
go test -race ./...
make build
make build-matrix
```

Binary local nằm trong `dist/`. Build mặc định headless; dùng `-tags tray` nếu cần adapter tray.

## CI/CD

Workflow `.github/workflows/build.yml` chạy khi push vào `main` hoặc `release/**`, pull request và thủ công:

1. Sinh display version và release tag theo timestamp commit UTC.
2. Chạy race test, vet và E2E local.
3. Build sáu target: Windows/Linux/macOS x amd64/arm64.
4. Ký binary, upload artifact và tạo GitHub draft release.
5. Smoke test upgrade, rollback, upgrade lại trên Linux, Windows và macOS.
6. Sinh và ký manifest, upload manifest rồi publish draft release.

Thay đổi chỉ Markdown hoặc `docs/**` được bỏ qua bởi workflow release.

## Secrets và variables

Secrets cần thiết: `APP_BINARY_PRIVATE_KEY`, `APP_MANIFEST_PRIVATE_KEY`, `APP_CURRENT_PUBLIC_KEY`, `APP_NEXT_PUBLIC_KEY`. Variable cần thiết: `APP_MANIFEST_URL`. Azure còn cần `AZURE_PUBLIC_BASE_URL`, `AZURE_SERVICE_CONNECTION`, `AZURE_STORAGE_ACCOUNT`, `AZURE_CONTAINER` trong pipeline Azure.

Không commit private key thật. Script `scripts/ci-keys.sh` có fallback dành cho local/CI không production; production phải cấu hình secret riêng.

## Quy trình phát hành

```bash
git checkout main
git pull --ff-only
# sửa code hoặc tài liệu
go test ./...
git add .
git commit -m "chore: ..."
git push origin main
```

Sau push, theo dõi workflow trên GitHub Actions. Khi workflow xanh, kiểm tra draft/release có đủ binary, `.sig`, `checksums.txt` và manifest đã ký. Không tự tạo version bằng file mới: version được workflow sinh từ commit timestamp. Nếu timestamp tạo tag đã tồn tại, generator tự tăng phút.

## Kiểm tra artifact

```bash
./dist/app version
sha256sum dist/*
```

Dùng `gh release view <tag>` để kiểm tra asset và trạng thái publish. Chỉ publish khi smoke test và bước ký manifest đều pass.
