# Test Case Report

## 1. Tổng quan

| Thuộc tính | Giá trị |
|---|---|
| Ngày chạy | 2026-07-29 |
| Môi trường local | Linux/amd64, Go toolchain hiện có trong sandbox |
| Cổng cuối | `FINAL_GATE_PASS` |
| Race test | PASS |
| Vet | PASS |
| Coverage toàn repo | 53.7% statement |
| Coverage updater lõi | 76.7% |
| E2E binary thật | PASS |
| Cross-build | 6/6 target PASS |
| Tray build tag | PASS |
| Template clone/init | PASS |

Log nguyên bản: `reports/final-test-output.log`  
Coverage profile: `reports/coverage.out`  
Danh sách binary kiểm tra: `reports/build-artifacts.txt`

## 2. Test case chức năng

| ID | Test case | Cách kiểm tra | Kết quả |
|---|---|---|---|
| VER-01 | Display version ↔ Git tag round-trip | `TestDisplayTagRoundTrip` | PASS |
| VER-02 | Comparator tăng đúng theo phút/ngày/năm | `TestCompare` | PASS |
| VER-03 | Từ chối ngày/giờ/version sai | `TestRejectInvalid` | PASS |
| VER-04 | Fuzz parser/round-trip | fuzz 2 giây, 104.938 executions, 125 interesting inputs | PASS |
| VER-05 | Hai version cùng phút không trùng tag | sinh tag khi tag đầu đã tồn tại | PASS: `...1656` → `...1657` |
| CFG-01 | Ưu tiên CLI > env > file > default | `TestPrecedence` | PASS |
| CFG-02 | Channel internal thiếu token phải lỗi rõ | `TestInternalRequiresTokenForUpdate` | PASS |
| SRC-01 | GitHub stable/beta/internal lọc draft/prerelease đúng | `TestGitHubSourceChannelsAndAssetSelection` | PASS |
| SRC-02 | Azure Blob index parse đúng | `TestAzureBlobSource` | PASS |
| RES-01 | Một nguồn list lỗi, nguồn kia vẫn resolve | `TestResolveLatestAndFallback` | PASS |
| RES-02 | Cùng version chọn nguồn list nhanh hơn | `TestResolveFastestForSameVersion` | PASS |
| RES-03 | Cả hai nguồn chết trả lỗi 50 mapping | `TestResolveAllDead` | PASS |
| RES-04 | Download nguồn ưu tiên lỗi, fallback nguồn thứ hai | `TestEngineFallsBackWhenPreferredDownloadFails` | PASS |
| RES-05 | Policy chỉ cho nguồn cụ thể | `TestListWithPolicyFiltersSources` | PASS |
| AST-01 | Chỉ chọn asset đúng GOOS/GOARCH | source integration + resolver tests | PASS |
| AST-02 | Bổ sung SHA/signature từ checksums và `.sig` | `TestEnrichAssetFromChecksumsAndSignature` | PASS |
| AST-03 | Thiếu checksum phải fail cứng | `TestEnrichAssetRequiresChecksum` | PASS |
| DL-01 | Resume bằng Range và verify thành công | `TestDownloadResumeVerify` | PASS |
| DL-02 | Checksum sai bị từ chối | `TestDownloadRejectsChecksum` | PASS |
| DL-03 | URL thay đổi phải xóa partial cũ | `TestDownloadResetsPartialWhenURLChanges` | PASS |
| SIG-01 | Minisign bốn dòng sign/verify | `TestMinisignRoundTripAndRawCompatibility` + CLI fixture | PASS |
| SIG-02 | Tương thích raw Ed25519 cũ | cùng test | PASS |
| MAN-01 | Manifest ký đúng và xoay current/next key | `TestManifestVerifyAndRotation` | PASS |
| MAN-02 | Manifest hết hạn bị chặn | `TestManifestExpired` | PASS |
| MAN-03 | min/force/rollout policy | `TestPolicy` | PASS |
| MAN-04 | Current bị blocked buộc nâng; target blocked bị cấm | `TestBlockedCurrentForcesUpgradeButBlockedTargetFails` | PASS |
| MAN-05 | Stable không vượt `latest` policy | `TestStableCannotExceedManifestLatest` | PASS |
| POL-01 | Forced update không gọi confirm | `TestForcedPolicySkipsConfirmation` | PASS |
| POL-02 | Non-forced bị từ chối không ghi target | `TestNonForcedCancellationDoesNotWriteTarget` | PASS |
| CCH-01 | Import blob theo SHA và LRU prune | `TestCacheImportAndLRUPrune` | PASS |
| CCH-02 | Metadata trong TTL không gọi mạng | `TestMetadataCacheTTLAndETag` | PASS |
| CCH-03 | Hết TTL gửi ETag và dùng 304 cache | cùng test | PASS |
| APP-01 | Atomic apply và rollback | `TestApplyRollback` | PASS |
| APP-02 | Lock loại trừ tiến trình đồng thời | `TestLock` | PASS |
| APP-03 | Active lock bị từ chối, stale lock được phục hồi | `TestWithLockRejectsActiveAndRecoversStale` | PASS |
| CLI-01 | `version --json` đúng cấu trúc | `TestVersionJSON` + CLI run | PASS |
| CLI-02 | `update --dry-run` không ghi binary | `TestUpdateDryRunFromGitHub` | PASS |
| CLI-03 | `--latest` và `--version` loại trừ nhau | `TestUpdateRejectsLatestAndVersionTogether` | PASS |
| UI-01 | Menu gom nguồn cùng version và hiện cache | `TestChooseGroupsSourcesAndShowsCache` | PASS |
| FEA-01 | Feature generator sinh ID/test đúng | generated healthcheck test | PASS |

## 3. Test case hệ thống và pipeline

| ID | Hạng mục | Lệnh/kiểm tra | Kết quả |
|---|---|---|---|
| SYS-01 | Race detector | `go test -race ./...` | PASS |
| SYS-02 | Static analysis | `go vet ./...` | PASS |
| SYS-03 | Coverage | `go test -coverprofile=reports/coverage.out ./...` | PASS, 53.7% |
| SYS-04 | E2E real binary | `./scripts/e2e-local.sh` | PASS |
| SYS-05 | Upgrade path | `1.26.0729.1900 → 1.26.0729.2000` | PASS |
| SYS-06 | Rollback path | new → old | PASS |
| SYS-07 | Re-upgrade after rollback | old → new | PASS |
| BLD-01 | Windows amd64 | `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build` | PASS |
| BLD-02 | Windows arm64 | tương tự | PASS |
| BLD-03 | Linux amd64 | tương tự | PASS |
| BLD-04 | Linux arm64 | tương tự | PASS |
| BLD-05 | Darwin amd64 | tương tự | PASS |
| BLD-06 | Darwin arm64 | tương tự | PASS |
| BLD-07 | Tray-tag package tests | `go test -tags tray ./...` | PASS |
| BLD-08 | Tray-tag Linux binary | `go build -tags tray` | PASS |
| CI-01 | GitHub workflow YAML | parse YAML | PASS |
| CI-02 | Azure pipeline YAML | parse YAML | PASS |
| CI-03 | GoReleaser YAML | parse YAML | PASS |
| CI-04 | GitHub embedded shell | trích `run:` rồi `bash -n` | PASS |
| CI-05 | Azure inline shell | trích script rồi `bash -n` | PASS |
| CI-06 | Mọi script shell repo | `sh -n scripts/*.sh` | PASS |
| AUD-01 | Kiểm tra cấu trúc/yêu cầu kế hoạch | `./scripts/audit-plan.sh` | PASS tất cả rule |
| TPL-01 | Copy template và đổi app/module | `scripts/init-template.sh selfdemo example.com/acme/selfdemo` | PASS |
| TPL-02 | Test repo sau đổi tên | `go test ./...` trong clone | PASS |
| TPL-03 | Build sáu target sau đổi tên | build matrix trong clone | PASS 6/6 |
| TPL-04 | Không còn chuỗi app/module cũ | grep kiểm tra | PASS |

## 4. E2E chứng minh binary tự cập nhật

Kịch bản dùng hai binary được build thật, không mock thao tác file executable:

```text
1. Chạy binary version 1.26.0729.1900.
2. Serve release 1.26.0729.2000 qua HTTP test server.
3. Binary cũ tải, verify và thay chính target.
4. Chạy target, xác nhận version 1.26.0729.2000.
5. Gọi rollback, xác nhận trở về 1.26.0729.1900.
6. Update lần nữa, xác nhận 1.26.0729.2000.
```

Kết quả log:

```text
E2E PASS [linux/amd64]: 1.26.0729.1900 -> 1.26.0729.2000 -> rollback -> 1.26.0729.1900 -> 1.26.0729.2000
```

## 5. Coverage

| Package | Statement coverage |
|---|---:|
| `internal/signing` | 86.2% |
| `internal/ui` | 84.2% |
| `internal/updater` | 76.7% |
| `internal/version` | 66.7% |
| `internal/config` | 53.0% |
| Toàn repo | 53.7% |

Coverage tổng bị kéo xuống bởi các command entrypoint và branch phụ thuộc external CI/network. Coverage lõi updater, signing và UI đều đủ cao để chứng minh các nhánh chính đã được thực thi, nhưng đây không được tuyên bố là coverage production tuyệt đối.

## 6. Các test chưa thể chạy trong sandbox

| Hạng mục | Lý do | Đã chuẩn bị |
|---|---|---|
| Smoke update thật trên Windows runner | Sandbox hiện tại là Linux | GitHub matrix `windows-latest` |
| Smoke update thật trên macOS runner | Sandbox hiện tại là Linux | GitHub matrix `macos-14` |
| macOS Gatekeeper/notarization | Không có Developer ID/Apple credentials | boundary và tài liệu deployment |
| Windows Program Files/UAC | Không có Windows/elevated installer | Windows build + smoke workflow |
| Tray icon native Linux/Windows/macOS | Adapter native chưa tích hợp | build-tag boundary + compile/lifecycle tests |
| Publish thật GitHub/Azure | Không có repo target/secrets/storage | workflow/pipeline đã parse và shell-check |

Không có mục nào trong bảng này được tính là test PASS local.
