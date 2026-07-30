# Code Proof — các điểm đã xong và mã chứng minh

Tài liệu này chỉ tới vị trí mã nguồn có thể kiểm tra trực tiếp. Số dòng áp dụng cho commit đi kèm gói bàn giao.

## 1. Version và tính đơn điệu

- Parse hai định dạng, validate ngày/giờ: `internal/version/version.go:29-97`.
- Hiển thị `1.YY.MMDD.HHmm`: `internal/version/version.go:106-108`.
- Tag `v1.YY.MDDHHmm`: `internal/version/version.go:109-113`.
- Comparator theo từng component: `internal/version/version.go:114-136`.
- Pipeline dùng UTC commit timestamp thay vì clock runner: `.github/workflows/build.yml` phần job `version`; `azure-pipelines.yml:54-59,87-90`.

Đoạn cốt lõi:

```go
func (v Version) String() string {
    return fmt.Sprintf("%d.%02d.%02d%02d.%02d%02d", v.Major, v.Year, v.Month, v.Day, v.Hour, v.Minute)
}
func (v Version) Tag() string {
    compact := fmt.Sprintf("%02d%02d%02d%02d", v.Month, v.Day, v.Hour, v.Minute)
    n, _ := strconv.Atoi(compact)
    return fmt.Sprintf("v%d.%d.%d", v.Major, v.Year, n)
}
```

## 2. Interface nguồn và asset chính xác

- Interface nguồn: `internal/updater/source.go:47-51`.
- Tên asset chuẩn theo OS/arch và `.exe`: `internal/updater/source.go:67-85`.

```go
type Source interface {
    Name() string
    List(context.Context, ListOptions) ([]Release, error)
    Fetch(context.Context, Release, io.Writer) error
}
```

## 3. Resolve song song và fallback hai nguồn

- Mỗi source được list trong goroutine: `internal/updater/resolver.go:27-57`.
- Một nguồn lỗi không làm mất kết quả nguồn còn lại: `resolver.go:59-72`.
- Max version thắng, cùng version ưu tiên latency thấp: `resolver.go:89-99`.
- Giữ mọi candidate cùng version để fallback download: `resolver.go:104-121`.
- Thử từng candidate; lỗi mạng thì chuyển nguồn tiếp theo: `internal/updater/updater.go:197-238`.
- Integrity metadata sai thì fail cứng, không “né” sang mirror: `updater.go:203-215`.

## 4. Manifest policy và kill switch

- Schema/channel/blocked/key/signature: `internal/updater/manifest.go:26-49`.
- Validate schema, expiry và version fields: `manifest.go:51-91`.
- Verify bằng danh sách public key current/next: `manifest.go` hàm `Verify`.
- Evaluate `blocked`, `min_supported`, `force_update`, rollout: `manifest.go` hàm `Evaluate`.
- Manifest được tải riêng bằng metadata cache rồi verify trước resolve: `internal/updater/updater.go:147-166,255-269`.
- Manifest source allow-list được áp dụng cho list/update: `updater.go:76-98,160-165,241-253`.

## 5. Forced update và exit contract

- Exit code chuẩn: `internal/updater/updater.go:19-26`.
- Default timeout 5 phút: `updater.go:136-142`.
- Policy đánh giá trước prompt: `updater.go:175-195`.
- Forced update bỏ qua callback confirm:

```go
if !result.Forced && opt.Confirm != nil && !opt.Confirm(result) {
    return result, ErrCancelled
}
```

- CLI không prompt khi `--silent`; prompt chỉ gắn khi không silent/dry-run: `internal/app/app.go:155-240`.

## 6. Resume, checksum và minisign

- Downloader `.part`, retry/backoff, Range/state, SHA256 và signature: `internal/updater/download.go`.
- URL đổi thì reset partial: logic state URL trong `download.go`.
- Metadata checksum/signature từ `checksums.txt` và `.sig`: `internal/updater/updater.go:297-342`.
- Parser/encoder minisign bốn dòng, EdDSA key ID và trusted comment: `internal/signing/minisign.go`.
- Raw signature compatibility: cùng file, nhánh decode base64 signature cũ.
- Công cụ phát hành: `cmd/keygen`, `cmd/binarysign`, `cmd/manifestsign`.

Signature tạo trong test fixture có đúng cấu trúc:

```text
untrusted comment: ...
<base64 signature packet>
trusted comment: ...
<base64 trusted-comment signature>
```

## 7. Cache metadata và binary

- Binary cache content-addressed theo SHA256, index và LRU: `internal/updater/cache.go`.
- Metadata cache TTL mặc định 15 phút: `internal/updater/metadata_cache.go`.
- Revalidation ETag bằng `If-None-Match`; HTTP 304 trả body cache: cùng file.
- Manifest/checksum/signature đều đi qua metadata cache: `updater.go:255-269,345-347`.
- Prefetch chỉ nhận version lớn hơn current và giữ `--keep` mặc định 3: `internal/app/app.go:345-406`.

## 8. Atomic apply, rollback và lock

- Wrapper apply giữ backup `.rollback`: `internal/updater/apply.go`.
- Atomic replacement sử dụng API tương thích MinIO selfupdate: `third_party/minio-selfupdate/apply.go`.
- Rollback khôi phục backup về target: `internal/updater/apply.go`.
- Lock có active/stale handling: cùng file; test ở `internal/updater/apply_test.go`.
- Engine chỉ apply bên trong `WithLock`: `internal/updater/updater.go:227-231`.

## 9. Feature template

- Interface feature nhỏ và không phụ thuộc `main`: `internal/features/features.go`.
- Registry: `internal/features/registry.go`.
- Generator package + test + registry entry: `cmd/newfeature/main.go`.
- Lệnh template: `make new-feature NAME=xyz`.
- Clone/init đổi tên app/module: `scripts/init-template.sh`, `cmd/templateinit`.

## 10. TUI/tray boundary

- Menu gom release cùng version, liệt kê nguồn và trạng thái cache: `internal/ui/menu.go`.
- Default headless: `internal/tray/tray_notray.go`.
- Build tag tray/lifecycle adapter: `internal/tray/tray_tray.go`.
- Hướng thay native systray/Bubble Tea: `docs/DESKTOP_ADAPTERS.md`.

Lưu ý: đây là boundary đã compile/test; không phải bằng chứng icon native production.

## 11. CI/CD và sáu target

- GitHub build matrix 3 OS × 2 arch: `.github/workflows/build.yml:56-100`.
- Draft bất biến, đủ signature/checksum: `.github/workflows/build.yml:102-131`.
- Smoke update real path trên Ubuntu/Windows/macOS, rollback rồi update lại: `.github/workflows/build.yml:133-209`.
- Chỉ promote sau smoke: `.github/workflows/build.yml:211-239`.
- Azure build matrix: `azure-pipelines.yml:36-66`.
- Azure Blob upload/index tối đa 50 release: `azure-pipelines.yml:68-135`.
- Local matrix: `scripts/build-matrix.sh`.

## 12. Bằng chứng test gắn với mã

- Danh sách toàn bộ test: xem `reports/TEST_CASE_REPORT.md`.
- Output nguyên bản: `reports/final-test-output.log`.
- Cổng đối chiếu kế hoạch tự động: `scripts/audit-plan.sh`.
- Kết thúc log phải có: `FINAL_GATE_PASS`.
