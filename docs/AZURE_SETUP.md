# Thiết lập Azure DevOps một lần (Artifacts feed cho pipeline)

> Áp dụng cho nghiệp vụ: publish Universal Package lên Azure Artifacts và promote `@Release` trong `azure-pipelines.yml` (stage **Publish**). Việc này chỉ cần làm **một lần** trên Azure DevOps; sau đó pipeline tự chạy.

## 1. Tổng quan

Stage **Publish** trong [`azure-pipelines.yml`](../azure-pipelines.yml) khi chạy trên `main` sẽ:

1. Lắp ráp `release/` (6 binary + 6 `.sig` + `checksums.txt` + `index.json`).
2. Publish package Universal (`app-release@<version>`) lên feed bằng task `UniversalPackages@0`.
3. Promote version vừa publish lên view `@Release` qua REST API để **giữ vĩnh viễn** (miễn trừ retention).

Để 3 bước này chạy được, cần một feed Artifacts sẵn sàng trước. Dưới đây là 2 cách: làm tay trên web (UI) hoặc tự động hóa bằng REST API.

Quy ước trong doc này:

| Tham số | Ví dụ |
| --- | --- |
| `{org}` | `o25160526-pip` |
| `{project}` | `myproj` |
| `{feed}` | `releases` |
| `AZURE_ARTIFACTS_FEED` | `myproj/releases` (feed project-scope) hoặc `releases` (feed org-scope) |

> Feed trong ví dụ là **project-scoped**: biến `AZURE_ARTIFACTS_FEED` có dạng `project/feed`. Nếu dùng feed org-scoped thì chỉ cần tên feed, không có phần project (xem cách pipeline parse ở `azure-pipelines.yml:214-220`).

## 2. Cách A — Làm tay trên web (UI)

### Bước 1: Tạo feed và set biến pipeline

1. Vào **Artifacts** → **+ Create feed**.
   - **Name**: `releases` (hoặc tên bạn thích).
   - **Scope**: chọn **Project** (`myproj`) — khớp với `AZURE_ARTIFACTS_FEED=myproj/releases`. Chọn **Organization** nếu muốn feed dùng chung nhiều project.
   - **Visibility/Upstream**: tắt upstream nếu không cần; universal package không dùng upstream.
2. Trong pipeline (UI: **Pipelines** → chọn pipeline → **Edit** → **Variables** → **New variable**) set:
   - `AZURE_ARTIFACTS_FEED` = `myproj/releases`

   Các biến Azure khác dùng trong pipeline (đều optional, chỉ bật khi cần kênh Blob):
   - `AZURE_PUBLIC_BASE_URL`, `AZURE_SERVICE_CONNECTION`, `AZURE_STORAGE_ACCOUNT`, `AZURE_CONTAINER` — xem mục 4.

### Bước 2: (Khuyến nghị) Feed Settings → Retention Policy

1. Vào **Artifacts** → chọn feed `releases` → gear icon (Feed Settings) → **Retention Policies**.
2. Bật **Enable package retention**, đặt **tối đa** để dự phòng:
   - **Maximum number of versions per package**: `5000` (giá trị tối đa cho phép).
   - **Days to keep recently downloaded packages**: `365` (tối đa).

Lý do: retention chỉ xóa version cũ khi **vượt** số lượng giới hạn **và** không được tải trong số ngày quy định. Đặt max giúp feed giữ được nhiều nhất có thể; quan trọng hơn, version nào đã **promote lên view `@Release` sẽ được miễn trừ retention và giữ vĩnh viễn** (pipeline tự làm ở bước promote).

### Bước 3: Grant quyền Build Service (Contributor)

1. Vào feed `releases` → gear icon → **Permissions** → **Add users/groups**.
2. Thêm **cả hai** identity sau, gán role **Feed Publisher (Contributor)**:
   - `Project Collection Build Service Accounts ({org})`
   - `{project} Build Service ({org})`

> Mặc định Build Service chỉ có **Feed and Upstream Reader (Collaborator)** — không publish được. Phải nâng lên **Contributor** (Feed Publisher) thì task `UniversalPackages@0` mới pass. Nếu bỏ qua, lỗi điển hình là `403 Forbidden / user does not have permission to publish`.

## 3. Cách B — Tự động hóa bằng REST API

Tất cả bước ở trên đều làm được qua REST API (đã xác nhận với tài liệu mới nhất, `api-version=7.1`, 2026). Khuyến nghị dùng script một lần (idempotent) thay vì thao tác tay.

### Chuẩn bị

- PAT với scope **Packaging (Read & Write)** (`vso.packaging_manage`) — đủ cho cả 3 thao tác.
- `curl` + `jq`. Đã cài `az devops`/`azure-devops` extension thì thay `curl` bằng `az devops` được, nhưng curl là đủ.
- Script chạy bằng **Basic auth**: header `Authorization: Basic $(printf ':%s' "$PAT" | base64)`.

```bash
ORG=o25160526-pip
PROJECT=myproj
FEED=releases
PAT=xxxx   # packaging:manage
AUTH="$(printf ':%s' "$PAT" | base64 | tr -d '\n')"
BASE="https://feeds.dev.azure.com/$ORG/$PROJECT"
```

### 3.1 Tạo feed (nếu chưa có)

```bash
curl -sf -X POST "$BASE/_apis/packaging/feeds?api-version=7.1" \
  -H "Authorization: Basic $AUTH" \
  -H "Content-Type: application/json" \
  -d '{"name":"releases","description":"Go selfupdate releases","hideDeletedPackageVersions":true,"upstreamEnabled":false}'
# HTTP 201. Nếu đã tồn tại, trả 409 — bỏ qua là được (idempotent).
```

- Feed org-scoped: bỏ `/$PROJECT` trong URL.
- Để set biến `AZURE_ARTIFACTS_FEED` cho pipeline qua API: dùng `GET/PUT /{org}/{project}/_apis/build/definitions/{id}` (khóa `variables`), hoặc bật bằng UI nhanh hơn.

### 3.2 Set retention policy tối đa

```bash
curl -sf -X PUT "$BASE/_apis/packaging/Feeds/$FEED/retentionpolicies?api-version=7.1" \
  -H "Authorization: Basic $AUTH" \
  -H "Content-Type: application/json" \
  -d '{"countLimit":5000,"daysToKeepRecentlyDownloadedPackages":365}'
# HTTP 200
```

> `countLimit` tối đa 5000, `daysToKeepRecentlyDownloadedPackages` tối đa 365 (khớp giới hạn UI). Version đã promote `@Release` vẫn được miễn trừ dù policy set gì.

### 3.3 Grant Build Service (Contributor)

Cần `identityDescriptor` của build service. Hai descriptor thông dụng (cấu trúc chuẩn, có thể kiểm tra bằng `GET .../permissions` của một feed khác để lấy đúng chuỗi):

```bash
# Project Collection Build Service (org-scoped)
COLLECTION_DESC="Microsoft.TeamFoundation.ServiceIdentity;Build:$ORG\\Project Collection Build Service Accounts"
# Project Build Service (project-scoped)
PROJECT_DESC="Microsoft.TeamFoundation.ServiceIdentity;Build:$ORG\\$PROJECT Build Service ($ORG)"

curl -sf -X PATCH "$BASE/_apis/packaging/Feeds/$FEED/permissions?api-version=7.1" \
  -H "Authorization: Basic $AUTH" \
  -H "Content-Type: application/json" \
  -d "[
    {\"identityDescriptor\":\"$COLLECTION_DESC\",\"role\":\"contributor\"},
    {\"identityDescriptor\":\"$PROJECT_DESC\",\"role\":\"contributor\"}
  ]"
# HTTP 200
```

Vai trò hợp lệ: `reader` | `collaborator` | `contributor` | `administrator`. Cần **`contributor`** (Feed Publisher) để publish; `collaborator` mặc định không publish được.

### Tài liệu API tham khảo

- Feed Management: <https://learn.microsoft.com/en-us/rest/api/azure/devops/artifacts/feed-management?view=azure-devops-rest-7.1>
- Retention Policies (Set): <https://learn.microsoft.com/en-us/rest/api/azure/devops/artifacts/retention-policies/set-retention-policy?view=azure-devops-rest-7.1>
- Feed Permissions (Set): <https://learn.microsoft.com/en-us/rest/api/azure/devops/artifacts/feed-management/set-feed-permissions?view=azure-devops-rest-7.1>
- Miễn trừ retention cho package trong view: <https://learn.microsoft.com/en-us/azure/devops/artifacts/concepts/views>

## 4. Những lưu ý khi pipeline chạy

- **Stage Publish chỉ chạy trên `main`** — điều kiện `eq(variables['Build.SourceBranch'], 'refs/heads/main')` ở `azure-pipelines.yml:146`.
- **Publish Universal Package luôn bật** khi `AZURE_ARTIFACTS_FEED` khác rỗng (`azure-pipelines.yml:201`). Bước **Promote `@Release`** chạy kèm (`azure-pipelines.yml:228`) — đây là bước khiến version được giữ vĩnh viễn.
- **Step Blob Storage là tùy chọn**: chỉ chạy khi có `AZURE_STORAGE_ACCOUNT` (`azure-pipelines.yml:233`) — cần tài khoản Azure có subscription. Đây là kênh mà **client** đọc `index.json` qua `APP_AZURE_INDEX_URL` (`internal/updater/source_azure.go`).
- **Không có Azure vẫn chạy được**: client mặc định chỉ dùng GitHub Releases; khi Azure thiếu/thất bại, resolver fallback về GitHub không lỗi (đã test). Xem logic 2 nguồn ở `internal/updater`.
- **Universal Package KHÔNG phải kênh cho client công khai**: `AzureBlobSource` cố tình chọn Blob `index.json` thay vì Universal Packages vì client không cần credential Azure DevOps (`source_azure.go:11-13`). Feed Artifacts chỉ là kênh backup/trace của team.
