# Đặc tả kỹ thuật

## 1. Phạm vi

Ứng dụng `app` tự tìm release phù hợp, xác minh artifact, áp dụng binary mới và giữ bản rollback. Code chính nằm trong `internal/updater`.

## 2. Luồng self-update

1. `internal/app` đọc cấu hình và tạo `updater.Engine`.
2. Engine tải manifest nếu `APP_MANIFEST_URL` có giá trị. Manifest được parse, kiểm tra schema/thời hạn và xác minh chữ ký bằng các public key cấu hình.
3. Source được lọc theo `channels[channel].sources`, sau đó GitHub và Azure được truy vấn đồng thời.
4. Resolver lọc channel, version và asset theo `GOOS/GOARCH`.
5. Engine lấy SHA-256 từ metadata asset hoặc `checksums.txt`, lấy chữ ký detached `<asset>.sig`, rồi downloader tải binary với retry và resume.
6. Binary được xác minh checksum và Ed25519, import vào cache.
7. `ApplyBinary` thay binary theo cách nguyên tử, lưu `<target>.rollback`, dùng `update.lock` để tránh chạy đồng thời.
8. Nếu apply lỗi, mã thoát thuộc nhóm apply/rollback; lệnh `rollback` khôi phục bản dự phòng.

`--dry-run` chỉ resolve và trả JSON, không tải/apply. `--silent` bỏ prompt xác nhận; forced update cũng bỏ prompt.

## 3. Version

`internal/version` dùng format display `1.YY.MMDD.HHmm`, ví dụ `1.26.0730.0139`. Tag được sinh thành `v1.YY.MDDHHmm`, ví dụ `v1.26.730139`. Version release do `cmd/versiongen` sinh từ thời điểm commit UTC; nếu tag đã tồn tại, generator tăng từng phút để tránh trùng.

Binary nhận `Display`, `Commit` và `BuildDate` qua Go ldflags. Lệnh kiểm tra:

```bash
./dist/app version
./dist/app version --json
```

## 4. Interface nội bộ

- `Source`: liệt kê release theo channel và token.
- `Resolver`: hợp nhất, sắp xếp release và chọn asset tương thích.
- `Engine.List` / `Engine.ListWithPolicy`: liệt kê release, có hoặc không áp manifest.
- `Engine.Update`: resolve, policy check, download, verify, cache và apply.
- `Manifest.Validate`, `Verify`, `Evaluate`: kiểm tra và áp chính sách.
- `Cache`: lưu metadata binary, LRU prune và prefetch.
- `ApplyBinary`, `Rollback`, `WithLock`: thay thế, khôi phục và đồng bộ hóa file.

## 5. Manifest

Manifest JSON schema 1 có dạng:

```json
{
  "schema": 1,
  "expires_at": "2027-07-29T00:00:00Z",
  "channels": {
    "stable": {
      "latest": "1.26.0730.0139",
      "min_supported": "1.26.0701.0000",
      "force_update": true,
      "rollout_percent": 100,
      "sources": ["github", "azure"]
    }
  },
  "blocked": ["1.26.0715.1200"],
  "key_id": "current",
  "signature": "BASE64_ED25519_SIGNATURE"
}
```

`latest` và `min_supported` phải parse được theo format version. `rollout_percent` nằm trong 0..100. Chữ ký được tạo trên JSON sau khi đặt `signature` rỗng. Hai public key hỗ trợ xoay khóa.

## 6. Chính sách và edge case

Manifest hết hạn, schema khác 1, chữ ký sai hoặc channel không tồn tại đều bị từ chối. Version bị block, thấp hơn `min_supported`, vượt `latest` ở channel không phải internal, hoặc nằm ngoài rollout đều không được phép. Thiếu checksum/chữ ký, sai checksum/chữ ký, không có asset phù hợp, source lỗi, timeout, cancel, apply lỗi và rollback lỗi được ánh xạ thành lỗi/mã thoát tương ứng.

Source có thể lỗi độc lập; kết quả thành công từ source khác vẫn được dùng. Metadata được cache 15 phút và revalidate bằng ETag. Download dùng file staging, retry tối đa 3 lần và cache binary sau khi xác minh.
