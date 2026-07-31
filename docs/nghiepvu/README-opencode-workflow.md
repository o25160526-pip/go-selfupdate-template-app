# Issue-Driven AI Development với OpenCode Zen (miễn phí)

Hệ thống tự động: tạo issue theo template → GitHub Actions chạy agent
(`anomalyco/opencode/github@latest`) bằng **model miễn phí của OpenCode Zen** →
agent nghiên cứu / lên kế hoạch / sửa code trong phạm vi nghiệp vụ → tạo PR.

**Không cần API key, không cài OpenCode GitHub App, không tốn tiền.**

## 1. Cách dùng (vòng đời)

1. Mở `docs/nghiepvu/INDEX.md` → chọn nghiệp vụ phù hợp → tạo issue từ template
   (dạng `[Tự cập nhật] Yêu cầu sửa lỗi / fix` …). Nhãn `nghiepvu:<slug>`,
   `type:research|fix|feature`, `status:new` được gắn tự động.
2. Chọn model trong dropdown (chỉ model free). Nếu không chắc → để mặc định.
3. Agent tự chạy `research`: phân tích và trả lời trong issue, gắn `status:plan-ready`.
4. Điều khiển tiếp bằng comment:
   - `/opencode ask` — hỏi thêm (agent trả lời, không sửa code).
   - `/opencode plan` — agent đề xuất kế hoạch chi tiết (file nào, test nào).
   - `/opencode approve` — duyệt kế hoạch → gắn `status:approved`.
   - `/opencode execute` — agent thực thi. **Chỉ chạy khi đã `status:approved`**,
     trừ issue có nhãn `size:small` (opt-in chạy thẳng).
5. Nếu có thay đổi trong phạm vi → workflow commit lên nhánh
   `agent/<số-issue>-<slug>` và mở PR → gắn `status:pr-created`. Bạn tự review
   và merge PR (không bao giờ tự merge).

## 2. Label

| Nhãn | Ý nghĩa |
| --- | --- |
| `nghiepvu:<slug>` | Nghiệp vụ (bắt buộc, xác định phạm vi file) |
| `type:research` / `fix` / `feature` | Loại yêu cầu |
| `status:new` → `analyzing` → `plan-ready` → `approved` → `executing` → `pr-created` / `done` | State machine |
| `status:error` | Agent lỗi (vd rate limit 429, đã retry 1 lần) |
| `status:needs-clarification` | Thiếu nhãn / chưa đủ thông tin |
| `status:needs-human` | Không còn model free — cần con người |
| `model:<id>` | Chọn model (ghi đè dropdown). Phải nằm trong danh sách free, nếu không → fallback `big-pickle` + cảnh báo |
| `size:small` | Cho phép `/opencode execute` mà không cần duyệt (opt-in) |

## 3. Model miễn phí

- Nguồn sự thật: `docs/nghiepvu/free-models.json`, được refresh tự động mỗi tuần
  (workflow `refresh-free-models`, script `scripts/refresh-free-models.mjs`, nguồn
  `https://opencode.ai/zen/v1/models`).
- Workflow `opencode-issue-agent` chỉ dùng model có id trong file đó, tiền tố
  `opencode/<id>` (mặc định `opencode/big-pickle`).
- **Không có model free khả dụng → dừng ngay với `status:needs-human`** — hệ thống
  không bao giờ tự chuyển sang model trả phí.

## 4. Phạm vi file (guardrail)

- Mỗi nghiệp vụ khai báo phạm vi bằng dòng `ALLOWED:` trong
  `docs/nghiepvu/<slug>.md` (vd: `ALLOWED: internal/updater/** internal/version/**`).
- Sau khi agent chạy, workflow kiểm tra `git status --porcelain` bằng
  `scripts/check-diff-scope.mjs`; **mọi file ngoài phạm vi bị revert tự động**
  (trong phạm vi được giữ lại). Lệnh `ask`/`plan`/`research` phải không đổi file nào
  (nếu đổi → revert toàn bộ).
- `docs/nghiepvu/**` luôn được phép (agent được cập nhật tài liệu nghiệp vụ).

## 5. Thêm nghiệp vụ mới

1. Tạo `docs/nghiepvu/<slug>.md` theo mẫu các file hiện có — **bắt buộc có dòng**
   `ALLOWED:` liệt kê phạm vi file.
2. Cập nhật `docs/nghiepvu/INDEX.md`.
3. Tạo 3 template `.github/ISSUE_TEMPLATE/<slug>-research.yml`,
   `<slug>-fix.yml`, `<slug>-feature.yml` (copy từ nghiệp vụ khác, đổi label
   `nghiepvu:<slug>`).

## 6. An toàn / chống lạm dụng

- Chỉ `GITHUB_TOKEN` (mặc định, hết hạn sau run) — không secret tùy chỉnh.
- `concurrency: group=opencode-issue-<issue.number>`, `cancel-in-progress: false`
  (mỗi issue chạy tuần tự, không hủy giữa chừng).
- `timeout-minutes: 30` cho mỗi run.
- Retry tối đa 1 lần khi agent fail (backoff ~30s); lỗi tiếp → `status:error`.
- Agent bị cấm `git commit/push/gh pr` trong prompt; mọi PR do workflow tạo.
- Bot không bao giờ tự merge PR, không bao giờ push thẳng vào `main`.
- Workflow không kích hoạt lại khi bot tự comment (lọc comment của Bot /
  comment không chứa `/opencode`).

## 7. Cấu hình local (tùy chọn)

`.opencode/opencode.json` đặt model mặc định + quyền (edit/bash allow, webfetch ask).
Muốn chạy thử ngoài GitHub:
`opencode run "<nội dung>" --model opencode/big-pickle`. Sau khi sửa config, thoát
và mở lại opencode để áp dụng.
