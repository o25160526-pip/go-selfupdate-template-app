# Ví dụ sử dụng

## Kiểm tra version

```bash
make build
./dist/app version
# 1.26.0730.0139
./dist/app version --json
```

JSON gồm `version`, `tag`, `commit` và `build_date`. Với binary CI, các trường commit/build date được nhúng qua ldflags.

## Liệt kê release

```bash
APP_GITHUB_OWNER=your-org \
APP_GITHUB_REPO=your-app \
./dist/app update --list --channel stable
```

Output gồm version, source, trạng thái stable/beta/draft và asset phù hợp.

## Dry run

```bash
./dist/app update --latest --dry-run --channel beta
```

Output mẫu:

```json
{
  "from": "1.26.0729.1930",
  "to": "1.26.0730.0139",
  "source": "github",
  "asset": "app_linux_amd64",
  "dry_run": true,
  "forced": false
}
```

## Cập nhật im lặng và chọn version

```bash
APP_GITHUB_OWNER=your-org \
APP_GITHUB_REPO=your-app \
./dist/app update --silent --version 1.26.0730.0139 --timeout 5m
```

Khi thành công, output có dạng `updated <from> -> <to> via <source> (<asset>)`. Nếu đã ở version mới nhất, mã thoát là 10.

## Rollback

```bash
./dist/app rollback
# rollback completed
```

Trong test có thể chỉ định binary:

```bash
./dist/app rollback --target "$PWD/dist/app"
```

## Channel, cache và cấu hình

```bash
./dist/app channel set beta
./dist/app config show
./dist/app cache list
./dist/app cache prefetch --keep 3
./dist/app cache prune --keep 3
```

Kênh `internal` cần `APP_UPDATE_TOKEN`. Manifest có thể giới hạn source và áp dụng rollout/force update.

## E2E local

```bash
./scripts/e2e-local.sh
```

Script dựng binary local, chạy update server test và kiểm tra upgrade/rollback theo luồng dành cho phát triển.
