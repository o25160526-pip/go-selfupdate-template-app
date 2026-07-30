# Báo cáo đối chiếu triển khai với kế hoạch

- Nguồn yêu cầu: `Kế hoạch triển khai_ go-selfupdate template app-20260729230922.md`
- Ngày kiểm định: 2026-07-29 (UTC+7)
- Commit mã nguồn trước báo cáo: `5907cf428d9285c48a13e2299e74c5288a71f1ae`
- Kết luận cổng kiểm định: **PASS** (`FINAL_GATE_PASS`)

## 1. Phương pháp thực hiện

Quy trình đã lặp theo đúng chu kỳ:

1. Dựng code theo roadmap và kiến trúc trong kế hoạch.
2. Chạy unit/integration/E2E/build/static validation.
3. Đối chiếu tự động bằng `scripts/audit-plan.sh` và đối chiếu thủ công từng yêu cầu.
4. Sửa sai lệch phát hiện được.
5. Chạy lại toàn bộ cổng kiểm định từ đầu.

Các sai lệch đã phát hiện và sửa trong các vòng lặp:

| Vòng | Sai lệch phát hiện | Điều chỉnh | Bằng chứng kiểm tra |
|---:|---|---|---|
| 1 | Import thừa làm lỗi biên dịch | Loại import, chạy lại toàn bộ test | `go test -race ./...` PASS |
| 2 | `.goreleaser.yml` có biểu thức template làm YAML không hợp lệ | Chuyển `ldflags` sang block scalar hợp lệ | YAML parser PASS |
| 3 | Resolver chỉ dự phòng khi `List` lỗi, chưa dự phòng khi tải asset từ nguồn nhanh nhất lỗi | Giữ toàn bộ candidate cùng version và thử tải lần lượt | `TestEngineFallsBackWhenPreferredDownloadFails` PASS |
| 4 | `force_update` vẫn có khả năng đi qua callback hỏi người dùng | Đánh giá policy trước prompt; bỏ qua confirm khi forced | `TestForcedPolicySkipsConfirmation` PASS |
| 5 | Cache mới chỉ lưu binary, thiếu TTL 15 phút và ETag cho metadata | Thêm `MetadataCache.Get`, TTL, `If-None-Match`, xử lý 304 | `TestMetadataCacheTTLAndETag` PASS |
| 6 | GitHub/Azure có thể sinh version khác nhau nếu dựa vào giờ runner | Sinh version từ UTC commit timestamp ở cả hai pipeline | test va chạm version PASS |
| 7 | Chữ ký binary ban đầu là Ed25519 base64 thô, chưa đúng minisign text | Thêm định dạng minisign bốn dòng và giữ tương thích cũ | CLI signing + `TestMinisignRoundTripAndRawCompatibility` PASS |
| 8 | Template đổi app/module có nguy cơ còn chuỗi cũ | Thêm script init và test clone thực tế | `TEMPLATE_INIT PASS ... targets=6` |

## 2. Đối chiếu theo phase

| Phase | Trạng thái | Phần đã triển khai | Bằng chứng chính |
|---|---|---|---|
| 0 — scaffold/version/CLI/ldflags | **Hoàn thành** | Cấu trúc repo; version hiển thị ↔ tag; so sánh; sinh version UTC từ commit; chống trùng; CLI command surface | `internal/version/version.go`; `cmd/versiongen`; `scripts/gen-version.sh`; test round-trip/fuzz |
| 1 — GitHub/apply/rollback | **Hoàn thành lõi** | GitHub Releases; chọn đúng OS/arch; tải, verify, atomic replace, backup và rollback; lock | `source_github.go`; `download.go`; `apply.go`; E2E binary thật |
| 2 — manifest/policy/signature | **Hoàn thành** | Schema 1; expiry; force/min/blocked/rollout; danh sách nguồn; hai khóa current+next; manifest Ed25519; binary minisign | `manifest.go`; `signing/minisign.go`; `buildinfo.go`; test policy/rotation |
| 3 — Azure/dual source/cache | **Hoàn thành** | Azure Blob `index.json`; list song song; ưu tiên latency; fallback cả list và download; cache SHA256/LRU; prefetch; metadata TTL/ETag | resolver, engine fallback, Azure source, cache tests |
| 4 — silent/exit codes/CI gate | **Hoàn thành** | Exit 0/10/20/30/40/50; `--timeout`; `--dry-run`; GitHub draft → smoke update → promote; Azure build/publish | `updater.go`; `internal/app/app.go`; workflows; YAML/shell checks |
| 5 — tray/TUI | **Hoàn thành giao diện lõi, adapter desktop một phần** | Menu terminal chọn version, nguồn, cache; tray build tag và lifecycle boundary | `internal/ui`; `internal/tray`; `go test -tags tray` PASS |
| 6 — template hóa | **Hoàn thành** | README; `make init`; `make new-feature`; registry; mẫu feature; tài liệu một trang | clone/init test và build sáu target PASS |

## 3. Ma trận yêu cầu chi tiết

| Yêu cầu kế hoạch | Trạng thái | Chứng minh |
|---|---|---|
| Interface `Source` nhỏ, tái sử dụng | Xong | `internal/updater/source.go:47-51` |
| Feature registry, không sửa `main.go` khi thêm feature | Xong | `internal/features/features.go`; `registry.go`; generator `cmd/newfeature` |
| Version `1.YY.MMDD.HHmm`, tag semver tương thích | Xong | `internal/version/version.go:29-136` |
| Sinh version UTC, tránh trùng phút | Xong | `cmd/versiongen`; test `VERSION_COLLISION PASS` |
| Stable/beta/internal; internal bắt buộc token | Xong | `source.go`; config validation; `TestInternalChannelRequiresToken` |
| Không nhúng token vào binary | Xong theo thiết kế | token chỉ lấy từ config/env runtime; CI truyền `${{ github.token }}` vào smoke job |
| Sáu target Windows/Linux/macOS × amd64/arm64 | Xong | build matrix thực tế PASS; CI/Azure matrix |
| Từ chối asset sai kiến trúc | Xong | tên asset chuẩn và `SelectAsset` exact match |
| Hai nguồn query song song, một nguồn chết vẫn hoạt động | Xong | `Resolver.Resolve`; resolver/integration tests |
| Nếu download nguồn ưu tiên lỗi thì thử nguồn còn lại | Xong | `updater.go:197-238`; fallback test |
| Manifest tách khỏi release | Xong | `APP_MANIFEST_URL`; metadata fetch riêng; workflow ký/đính manifest |
| Force/min/blocked/rollout/source policy | Xong | `manifest.go`; toàn bộ policy tests |
| Hai public key để xoay khóa | Xong | `internal/buildinfo`; `Manifest.Verify`; signing tests |
| Resume Range, retry 3, proxy, `.part`, state | Xong | `download.go`; resume/reset tests |
| SHA256 trước chữ ký | Xong | downloader verify pipeline; checksum rejection test |
| Binary signature minisign | Xong | `internal/signing/minisign.go`; `cmd/binarysign`; file test bốn dòng |
| Cache binary theo SHA256 + LRU | Xong | `cache.go`; cache test |
| Metadata TTL 15 phút + ETag | Xong | `metadata_cache.go`; ETag/304 test |
| Prefetch chỉ version mới hơn, giữ mặc định 3 | Xong | `internal/app/app.go:345-406` |
| Atomic swap + rollback | Xong lõi | MinIO-compatible apply subset; `apply.go`; E2E upgrade/rollback/upgrade |
| Lock tránh hai tiến trình update | Xong | active/stale lock tests |
| Silent exit code chuẩn | Xong | `updater.go:19-26`; command tests |
| `--timeout=5m`, `--dry-run` | Xong | engine timeout và CLI flags |
| TUI chọn version/xem nguồn/cache | Xong dạng terminal | `internal/ui/menu.go`; menu test |
| Tray thao tác desktop | Một phần | boundary + build tag có thể build/test; chưa là icon native production |
| GitHub build/test/draft/smoke/promote | Xong ở cấu hình | workflow parse PASS; local E2E PASS |
| Azure build 6 target + Blob index | Xong ở cấu hình | pipeline parse PASS; inline shell parse PASS |
| Real E2E không mock | Xong trên Linux local; cấu hình CI cho 3 OS | `scripts/e2e-local.sh`; GitHub smoke matrix |
| `make init`, `make new-feature` | Xong | clone template thực tế, đổi module/app, test + 6 build PASS |

## 4. Sai lệch còn lại và giới hạn triển khai

Các mục dưới đây được ghi rõ để không đánh đồng “code lõi hoàn thành” với “đã phát hành production trên mọi desktop”:

1. **CLI không dùng Cobra thực tế.** Command surface được triển khai bằng `flag` chuẩn thư viện Go để build offline. Hành vi lệnh, flags và exit contract đã có; muốn bám tuyệt đối stack chốt thì thay parser bằng Cobra tại boundary `internal/app`.
2. **TUI không dùng Bubble Tea thực tế.** Hiện là chooser terminal không phụ thuộc ngoài, có grouping nguồn/cache và test. Boundary `internal/ui` đã tách để thay renderer.
3. **Tray chưa dùng `fyne.io/systray` native.** Build tag `tray/notray` và adapter lifecycle đã có, nhưng chưa tạo icon/notification native. Việc này cần dependency GUI, CGO và runner native; hướng thay adapter nằm tại `docs/DESKTOP_ADAPTERS.md`.
4. **MinIO selfupdate được vendored dưới dạng subset tương thích source.** Nguyên nhân là môi trường kiểm định không có tải Go module; API atomic swap cần dùng đã được giữ và test. Khi có mạng module có thể bỏ `replace` để dùng upstream nguyên bản.
5. **macOS codesign/notarization chưa thể thực thi cục bộ** vì thiếu Developer ID/Apple credentials. Pipeline cần bổ sung secret và chạy `codesign/notarytool` trên macOS.
6. **Windows UAC/Program Files chưa được test cục bộ.** Binary Windows đã cross-build; smoke workflow có Windows runner, nhưng helper nâng quyền vẫn là bước triển khai sản phẩm tùy cách cài đặt.
7. **E2E local chỉ chạy Linux/amd64.** Workflow GitHub đã định nghĩa smoke trên Ubuntu, Windows và macOS; kết quả runner thật chỉ có sau khi repo được đẩy lên GitHub và cấu hình secrets.
8. **Tray auto-update nền/tạm hoãn 24 giờ chưa có scheduler lâu dài.** Config/menu boundary đã sẵn, nhưng service nền native không nằm trong artifact hiện tại.

Vì các điểm trên, kết luận chính xác là: **xương sống Phase 0–4 và template Phase 6 hoàn thành, kiểm thử được; Phase 5 hoàn thành boundary/TUI terminal nhưng adapter desktop native vẫn là hạng mục tích hợp trước production.**

## 5. Kết luận

- Toàn bộ cổng kiểm định cuối đã đạt.
- Không có test thất bại bị bỏ qua.
- Các phần chưa thể chứng minh trong sandbox đã được phân loại rõ, không đánh dấu hoàn thành giả.
- Mã nguồn phù hợp để dùng làm template headless/CLI ngay; trước khi phát hành desktop production cần hoàn thiện adapter tray native, ký/notarize macOS và kiểm thử UAC Windows.
