# PROMPT GỐC — Triển khai "Issue-Driven Development" bằng opencode-ai

> File này lưu lại **nguyên văn spec** mà chủ repo đưa ra để xây dựng hệ thống
> Issue-Driven AI Development. Toàn bộ hạ tầng hiện tại (`docs/nghiepvu/`,
> `.github/ISSUE_TEMPLATE/`, `.github/workflows/opencode-issue-agent.yml`,
> `.github/workflows/refresh-free-models.yml`, `.opencode/`, `scripts/`) được
> triển khai theo đúng spec này.
>
> Mọi thay đổi lớn về vận hành nên đối chiếu với spec này trước.

## Yêu cầu tổng quát

Triển khai quy trình **Issue-Driven Development** bằng opencode-ai, miễn phí
100%, không cần API key, không cài thêm GitHub App:

- Người dùng tạo **issue** theo template → GitHub Actions tự chạy agent
  (`anomalyco/opencode/github@latest`) với model miễn phí của OpenCode Zen →
  agent nghiên cứu / lên kế hoạch / sửa code trong **phạm vi nghiệp vụ** → tạo PR.
- Chỉ dùng **OpenCode Zen free tier** (`opencode/<model-id>`), **KHÔNG bao giờ**
  dùng model trả phí, **KHÔNG có secret/API key** trong workflow.

## Phase 0 — Khảo sát codebase, khai báo nghiệp vụ

- Quét toàn bộ codebase, chia thành các **nghiệp vụ** (business domain), mỗi
  nghiệp vụ một file `docs/nghiepvu/<slug>.md`:
  - Slug (viết thường, gạch ngang), label `nghiepvu:<slug>`
  - Mô tả nghiệp vụ
  - Thư mục/file liên quan
  - **Quy tắc bắt buộc** (không thể thỏa hiệp khi sửa)
  - **Phạm vi file được phép sửa**
- Tạo `docs/nghiepvu/INDEX.md` — bảng chỉ mục các nghiệp vụ để người dùng chọn.

## Phase 1 — Issue templates

- Với mỗi nghiệp vụ × mỗi loại issue (`research` / `fix` / `feature`) → một file
  `.github/ISSUE_TEMPLATE/<slug>-<type>.yml`:
  - Dropdown chọn model AI (danh sách model free).
  - Tự gắn nhãn `nghiepvu:<slug>`, `type:<type>`, `status:new`.
- `.github/ISSUE_TEMPLATE/config.yml` — **tắt blank issue**
  (`blank_issues_enabled: false`).

## Phase 2 — Cấu hình opencode + tự refresh danh sách model free

- `.opencode/opencode.json`:
  - KHÔNG có `apiKey`.
  - `permission`: `edit` = `allow`, `bash` = `allow`, `webfetch` = `ask`.
  - Model mặc định: `opencode/big-pickle`.
- `scripts/refresh-free-models.mjs` (Node, chỉ dùng built-in fetch):
  - GET `https://opencode.ai/zen/v1/models`.
  - Lọc model free: `cost.input === 0 && cost.output === 0 && status !== "deprecated"`
    (nếu API không trả các field này thì fallback heuristic: id chứa `-free`
    hoặc `big-pickle`).
  - Ghi `docs/nghiepvu/free-models.json` (`{"fetched_at", "source", "free": [...]}`).
  - Danh sách model free **xoay vòng liên tục** → phải refresh định kỳ.
- `.github/workflows/refresh-free-models.yml`:
  - `schedule` hàng tuần + `workflow_dispatch`.
  - Tạo PR bằng `peter-evans/create-pull-request@v6`.
  - Nếu danh sách rỗng → báo lỗi, không mở PR.

## Phase 3 — Workflow chính `opencode-issue-agent.yml`

- Triggers:
  - `issues: [opened, labeled]` (mở issue → tự research)
  - `issue_comment: [created]` (đọc `/opencode ...`)
- Permissions: `contents`, `issues`, `pull-requests`, `id-token` = `write`.
- **KHÔNG khai báo env chứa API key/secret** — chỉ dùng `GITHUB_TOKEN`.
- `concurrency`: `group = opencode-issue-<issue.number>`,
  `cancel-in-progress: false`.
- `timeout-minutes: 30`.
- **Routing** bằng `actions/github-script`: đọc labels
  (`nghiepvu:<slug>`, `type:<type>`, `model:<id>`, `size:<size>`)
  và comment `/opencode ask | plan | approve | execute`:
  - Map `model:<id>` → `opencode/<id>`; model không nằm trong
    `free-models.json` → fallback `opencode/big-pickle` **+ comment cảnh báo**.
  - Không có model free nào → **dừng, `status:needs-human`** — tuyệt đối không
    tự chuyển sang model trả phí.
- **Prompt-context** = nội dung issue + toàn bộ comments + `docs/nghiepvu/<slug>.md`.
- **State machine** qua label:
  `status:new → analyzing → plan-ready → approved → executing → pr-created | done`
  cộng `status:error`, `status:needs-clarification`, `status:needs-human`.
- **Guardrail**:
  - `execute` chỉ chạy khi `status:approved` — trừ issue có `size:small` (opt-in).
  - Giới hạn file sửa trong phạm vi nghiệp vụ, **validate bằng `git diff --name-only`**;
    file ngoài phạm vi bị revert.
  - Retry tối đa **1 lần** khi gặp lỗi (vd HTTP 429), backoff ~30s; vẫn lỗi →
    `status:error`.

## Phase 5 — Tài liệu vận hành

- `docs/nghiepvu/README-opencode-workflow.md`: cách dùng, label, state machine,
  guardrail, cách thêm nghiệp vụ, xử lý khi hết model free.

## Ràng buộc cứng (bất biến)

1. **Chỉ model OpenCode Zen free** — không API key, không secret.
2. Mặc định `opencode/big-pickle`.
3. Danh sách model free **tự refresh** (list xoay vòng); script phải viết phòng thủ.
4. Không có model free → `status:needs-human`, **dừng**, không dùng model trả phí.
5. **Không push thẳng vào `main`** — nhánh `agent/<issue-number>-<slug>`,
   mở PR bằng `gh pr create`, **không bao giờ tự merge**.
6. Label taxonomy: `nghiepvu:<slug>`, `type:research|fix|feature`, `status:*`,
   `model:<id>` (không bao giờ `model:anthropic-*`), `size:small|large`.
7. `cancel-in-progress: false`, `timeout-minutes: 30`, retry tối đa 1 lần.
