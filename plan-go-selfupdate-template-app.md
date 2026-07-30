# Kế hoạch triển khai: go-selfupdate template app

> Kế hoạch cho prompt Private ([https://app.clickup.com/90182930008/docs/2kzn4kjr-78/2kzn4kjr-38](https://app.clickup.com/90182930008/docs/2kzn4kjr-78/2kzn4kjr-38)). Hướng: **một repo template Go** (`gh template repo`) có sẵn engine tự cập nhật + tray + CI/CD, feature mới cắm vào sau.
## 1\. Kiến trúc template

```plain
cmd/app/main.go              # entrypoint, wire mọi thứ
internal/version/            # version hiện tại (ldflags), parser + comparator
internal/updater/
  updater.go                 # orchestrator: check → resolve → download → apply
  source.go                  # interface Source
  source_github.go           # GitHub Releases (kể cả draft, chỉ khi có token)
  source_azure.go            # Azure DevOps Universal Packages / Blob
  manifest.go                # parse + verify policy manifest
  cache.go                   # cache metadata + binary theo SHA256
  apply.go                   # atomic swap + rollback
internal/features/           # <<< chỗ cắm chức năng mới, mỗi feature 1 package + Register()
internal/tray/               # system tray (build tag: tray / notray)
internal/ui/                 # menu tương tác (TUI)
configs/manifest.example.json
.github/workflows/build.yml
azure-pipelines.yml
```

**Interface trung tâm** (giữ nhỏ, đây là lý do template này tái dùng được):

```go
type Source interface {
    Name() string
    List(ctx context.Context, opt ListOptions) ([]Release, error)
    Fetch(ctx context.Context, r Release, dst io.Writer) error
}

type Feature interface {
    ID() string
    TrayItems() []tray.Item   // tự thêm vào tray menu
    Commands() []*cobra.Command
    Start(ctx context.Context) error
}
```

Feature mới = tạo package trong `internal/features/x`, gọi `features.Register(&X{})` trong `init()`. Không sửa `main.go`. Đó là toàn bộ điều kiện để gọi cái này là "template".
## 2\. Stack chốt sẵn

| Việc | Chọn | Lý do |
| ---| ---| --- |
| Self-update | `github.com/minio/selfupdate` | atomic replace, rollback, verify ed25519 sẵn. Đừng tự viết phần swap binary. |
| CLI/menu | `spf13/cobra` + `charmbracelet/bubbletea` | menu chạy được cả trên server SSH, không cần GUI toolchain |
| Tray | `fyne.io/systray` | fork còn được maintain của getlantern/systray |
| Build/release | `goreleaser` | cross-compile 3 OS + checksum + upload draft release trong 1 config |
| Chữ ký | `ed25519` (minisign format) | nhẹ, verify offline, không cần cosign infra |

## 3\. Ba chỗ tôi phản biện trong prompt
**1\.** **`1.YY.MMDD.HHmm`** **không phải semver → goreleaser, Go modules,** **`golang.org/x/mod/semver`** **đều vỡ.** Fix: tag git là `v1.26.7291930` (MMDDHHmm gộp, vẫn tăng đơn điệu trong năm, semver hợp lệ), còn hiển thị cho user vẫn là `1.26.0729.1930`. `internal/version` lo phần map 2 chiều.

**2\. "Client có cờ kiểm tra update từ draft release" là lỗ bảo mật.** GitHub API chỉ trả draft khi có token có quyền repo → nghĩa là phải nhét token vào binary phát hành. Đừng. Thay bằng: kênh `internal` chỉ dùng trong CI với `GITHUB_TOKEN` của job, hoặc dev tự set `APP_UPDATE_TOKEN` trong env. Client public không bao giờ thấy draft.

**3\. "Tool luôn version mới nhất" trong YAML = build không reproducible + rủi ro supply chain.** Pin version + để Renovate/Dependabot bump hàng tuần. Vẫn "mới nhất" nhưng có PR để review, không phải mới nhất kiểu roulette.

**Ngoài ra:** Azure DevOps không có concept "release" như GitHub. Cần chốt: **Universal Packages feed** (khuyên dùng, có versioning sẵn) hay Blob Storage + `manifest.json`.
## 4\. Manifest chính sách
Host tách rời khỏi release (blob/gh-pages) để đổi policy không cần build lại:

```json
{
  "schema": 1,
  "channels": {
    "stable": {
      "latest": "1.26.0729.1930",
      "min_supported": "1.26.0701.0000",
      "force_update": true,
      "rollout_percent": 100,
      "sources": ["github", "azure"]
    }
  },
  "blocked": ["1.26.0715.1200"],
  "signature": "<ed25519>"
}
```

*   `force_update: true` → client tự update im lặng, không hỏi.
*   `min_supported` → dưới mức này là bắt buộc, không cho skip.
*   `blocked` → kill switch cho bản lỗi. Cái này bạn sẽ cảm ơn nó lúc 2h sáng.
*   `rollout_percent` → hash machine-ID để chia nhóm, ổn định giữa các lần chạy.
## 5\. Logic 2 nguồn
*   `--version=X`: query song song cả 2 nguồn, nguồn nào có X thì lấy, ưu tiên nguồn nhanh hơn (đo latency, cache lại).
*   `--latest`: query song song, so sánh comparator, lấy max. Một nguồn chết → vẫn chạy với nguồn còn lại, không fail cả tiến trình.
*   Cache: `~/.cache/<app>/meta.json` (TTL 15 phút, có ETag) + `~/.cache/<app>/blobs/<sha256>`. Chỉ prefetch bản **mới hơn** bản hiện tại, giới hạn N bản gần nhất (default 3), có LRU dọn rác.
## 6\. CI/CD
**GitHub Actions** — cache: `actions/setup-go` với `cache: true` (lo `~/go/pkg/mod`) + `actions/cache` cho `GOCACHE`, key theo `go.sum` + OS. Job:

1. `test` → 2. `build` (goreleaser, matrix 3 OS) → 3. **`smoke-update`**: tải binary release _trước đó_, chạy `app update --silent --channel=internal`, assert version mới + binary chạy được + rollback test → 4. `promote`: draft → published + push manifest.

**Azure Pipelines** — `Cache@2` cho `$(GOMODCACHE)` và `$(GOCACHE)`, cùng key strategy. Chạy song song, publish lên Universal Packages.

Đây là phần hay nhất trong prompt của bạn: **không release nào được publish nếu chưa tự chứng minh là update được**. Đảm bảo luôn có ít nhất 1 test upgrade-path thật, không mock.
## 7\. Roadmap

| Phase | Nội dung | Xong khi |
| ---| ---| --- |
| 0 | Scaffold repo, version pkg + comparator, cobra skeleton, ldflags | `app version` in đúng `1.YY.MMDD.HHmm` |
| 1 | GitHub source + apply + rollback | update tay từ 1 nguồn chạy trên 3 OS |
| 2 | Manifest + policy + ed25519 verify | force update & blocked hoạt động |
| 3 | Azure source + logic 2 nguồn + cache/prefetch | 1 nguồn chết vẫn update được |
| 4 | `--silent`, exit code chuẩn, CI gate promote draft→publish | pipeline tự chặn bản lỗi |
| 5 | Tray + TUI menu (chọn version, xem changelog, check now) | user không cần CLI |
| 6 | Dọn thành template repo: README, `make init`, feature registry mẫu | clone → đổi tên → chạy |

Phase 0-4 là xương sống, làm tuần tự. Phase 5-6 làm song song được.
## 8\. Rủi ro cần xử lý sớm
*   **macOS**: binary không sign/notarize → Gatekeeper chặn. Cần Developer ID + `codesign`/`notarytool` trong pipeline, hoặc chấp nhận user phải chạy `xattr -d`.
*   **Windows**: không ghi đè được file .exe đang chạy → cần side-load + rename + restart. `selfupdate` xử lý được nhưng phải test kỹ, đặc biệt khi app cài ở `Program Files` (cần UAC elevate).
*   **Tray trên Linux** cần CGO + `libayatana-appindicator3-dev` → phá cross-compile thuần. Vì vậy phải có build tag `notray` cho bản headless.
*   **Update khi app đang chạy tray**: phải có lock file, tránh 2 tiến trình cùng update.
## 9\. Rà soát lại prompt: những chỗ bản kế hoạch trên còn hở
Đối chiếu từng gạch đầu dòng trong prompt gốc, đây là phần chưa được đặc tả đủ để code được. Bổ sung bên dưới.
### 9.1 Trigger build khi push commit (prompt có yêu cầu, mục 6 chưa nói)

```yaml
on:
  push:
    branches: [main, 'release/**']
    paths-ignore: ['**.md', 'docs/**', '.github/ISSUE_TEMPLATE/**']
concurrency:
  group: release-${{ github.ref }}
  cancel-in-progress: false   # KHÔNG cancel: mỗi commit phải ra 1 version riêng
```

*   `cancel-in-progress: false` là bắt buộc. Nếu cancel, bạn sẽ có tag đã tạo mà không có release đi kèm → client 404.
*   Push vào branch khác → chỉ `test` + `build`, không tag, không release.
*   Azure: `trigger: branches: include: [main]` + `batch: false` (cùng lý do).
### 9.2 Sinh version trong CI: 2 cái bẫy prompt không lường
**Bẫy timezone.** Runner GitHub chạy UTC, Azure agent tuỳ pool → cùng 1 commit ra 2 version khác nhau. **Chốt cứng** **`TZ=UTC`** cho bước sinh version ở cả 2 pipeline, ghi rõ trong README. `1.26.0729.1230` nghĩa là 12:30 UTC, không phải giờ VN.

**Bẫy trùng version.** 2 commit trong cùng 1 phút → cùng `HHmm` → tag đã tồn tại → pipeline fail giữa đường. Xử lý trong script sinh version:

```bash
# nếu tag đã tồn tại, cộng dồn 1 phút cho tới khi trống
while git rev-parse "v$VER" >/dev/null 2>&1; do
  MIN=$((MIN+1)); VER=$(printf "1.%s.%s%04d" "$YY" "$MMDD" "$MIN")
done
```

Đừng dùng semver build metadata (`+run.123`) để phân biệt: build metadata **bị bỏ qua** khi so sánh version, client sẽ không biết bản nào mới hơn.

**Kiểm chứng tính đơn điệu:** trong 1 năm `MMDDHHmm` luôn tăng (01010005 < 12312359 ✓). Qua năm mới minor tăng (26 → 27) nên tổng thể vẫn tăng. Format an toàn tới 2099.
### 9.3 "Đa môi trường" cần nói rõ là 6 target, không phải 3
Prompt viết "desktop, linux, macos". Thực tế bắt buộc:

| OS | Arch | Lưu ý |
| ---| ---| --- |
| windows | `amd64`, `arm64` | arm64 cho Surface/Snapdragon, đang tăng |
| linux | `amd64`, `arm64` | arm64 cho server ARM + Raspberry Pi |
| darwin | `amd64`, `arm64` | arm64 KHÔNG optional, Mac Intel đã EOL |

Updater phải tự chọn asset theo `runtime.GOOS`/`GOARCH` từ tên file chuẩn `app_<os>_<arch>[.exe]`. Và phải **từ chối** update nếu asset không khớp arch, thay vì tải bừa rồi crash.
### 9.4 Cờ kiểm tra draft: đặc tả cụ thể phần thay thế
Mục 3 chỉ nói "đừng nhét token", chưa nói làm thế nào. Cơ chế:
*   Client có `--channel` với 3 giá trị: `stable` (mặc định), `beta`, `internal`.
*   `internal` = có đọc draft/prerelease, **chỉ hoạt động khi** env `APP_UPDATE_TOKEN` tồn tại. Không có token → updater trả lỗi rõ ràng, không im lặng fallback về stable.
*   Trong pipeline, bước `smoke-update` set `APP_UPDATE_TOKEN=${{ github.token }}` → đúng nhu cầu "app chạy trong build kiểm tra được draft".
*   Dev nội bộ muốn test bản draft thì tự set PAT. Zero token trong binary phát hành.
### 9.5 Silent update: hợp đồng exit code (CI phụ thuộc vào cái này)
`app update --silent` phải không TTY, không tray, không prompt, log ra `stderr` + file:

| Code | Nghĩa | CI làm gì |
| ---| ---| --- |
| `0` | đã update xong | promote draft → publish |
| `10` | đã ở bản mới nhất | pass (no-op hợp lệ) |
| `20` | không tìm thấy version | fail, giữ draft |
| `30` | verify chữ ký/checksum sai | fail cứng + alert |
| `40` | apply lỗi, đã rollback | fail, giữ draft |
| `50` | mọi nguồn không tới được | retry 1 lần rồi fail |

Kèm `--timeout=5m` bắt buộc, để job không treo. Và `--dry-run` để test resolve mà không ghi đè binary.
### 9.6 Bề mặt lệnh + menu (prompt yêu cầu "menu chọn phiên bản")

```plain
app update                    # tương tác, mặc định latest ở channel hiện tại
app update --version=1.26.0729.1930
app update --latest --silent
app update --list             # liệt kê version khả dụng kèm nguồn nào có
app rollback                  # về bản trước, dùng binary backup local
app channel set beta
app cache list | prune | prefetch --keep=3
app version --json
```

**TUI menu** (bubbletea): danh sách version × nguồn nào có × đã cache chưa (✓ tải sẵn) × changelog, cho phép chọn **cả bản cũ hơn** (downgrade, chặn nếu dưới `min_supported`), toggle auto-update, tạm hoãn 24h.

**Tray menu**: trạng thái icon 4 màu (up-to-date / có bản mới / đang tải / lỗi), notification khi có bản mới, "Check now", "Update now", "Mở menu", "Tạm hoãn", "Quit". Prompt chỉ nói "có tray icon để thao tác khi cần" → đây là danh sách thao tác đó.
### 9.7 Config + thứ tự ưu tiên (chưa có trong bản trên)
`CLI flag > env (APP_*) > config file > manifest policy > default`. Ngoại lệ: `force_update` và `blocked` trong manifest **luôn thắng** config local, nếu không kill switch vô nghĩa. Config ở `~/.config/<app>/config.yaml`, có `app config show` để debug.
### 9.8 Tải xuống bền bỉ (mục 5 nói cache nhưng không nói mạng lỗi)
Resume bằng HTTP Range, retry exponential backoff 3 lần, tôn trọng `HTTP_PROXY`/`HTTPS_PROXY`, verify SHA256 từ `checksums.txt` **trước** khi verify chữ ký ed25519, tải vào `.part` rồi rename. Ghi `download.state.json` để lần chạy sau tiếp tục từ chỗ dứt.
### 9.9 Xoay khoá chữ ký
Nhúng **2 public key** (current + next) trong binary từ ngày đầu. Không có cái này, mất private key = toàn bộ client cài trước đó thành brick, phải cài tay lại. Rẻ để làm bây giờ, không thể thêm sau.
### 9.10 Ma trận test tối thiểu
*   Unit: version parse/compare (fuzz), manifest verify (chữ ký sai, hết hạn, schema lạ), resolve 2 nguồn (1 chết / cả 2 chết / version chỉ có ở 1 nguồn).
*   Integration: fake GitHub + Azure server (httptest), assert đúng nguồn thắng.
*   E2E trên runner thật cả 3 OS: `vN-1` → `vN` → rollback → `vN`. Đây là gate publish, không được mock.
*   Test đặc thù: Windows update khi binary đang chạy; macOS quarantine attribute; Linux không có systray lib (build `notray`).
### 9.11 Scaffold cho "thêm chức năng khi cần"
Đây là điều kiện để nó là template, không phải app: `make new-feature NAME=xyz` sinh sẵn package + struct implement `Feature` + file test + tự append vào `features/registry.go`. Kèm `docs/ADDING_A_FEATURE.md` dài đúng 1 trang. Không có generator, 3 tháng sau chính bạn cũng sẽ copy-paste sai.

**Cập nhật roadmap:** 9.1-9.2 nhét vào Phase 0 (không có version sinh đúng thì mọi thứ sau đều sai), 9.4-9.5 vào Phase 4, 9.6 vào Phase 5, 9.9 phải làm **ngay Phase 1** cùng lúc với verify, 9.11 vào Phase 6.