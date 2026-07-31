# .opencode/

Cấu hình của [opencode](https://opencode.ai) cho repo này — phục vụ quy trình
**Issue-Driven AI Development** (xem `docs/nghiepvu/README-opencode-workflow.md`).

## Quy tắc quan trọng

- **KHÔNG có API key hay secret trong repo.** Mọi phiên chạy (local hoặc CI)
  đều dùng OpenCode Zen — chỉ các model miễn phí có id dạng `opencode/<id>`.
- `opencode.json` chỉ khai báo model mặc định (`opencode/big-pickle`) và quyền:
  - `edit`/`bash`: **allow** — agent được sửa file và chạy lệnh khi thực thi.
  - `webfetch`: **ask** — hỏi trước khi tải trang web.
  - Model trong CI luôn được ghi đè bằng model đã chọn ở dropdown issue
    (xem `.github/workflows/opencode-issue-agent.yml`), KHÔNG dùng model ở đây.

## Danh sách model free

File `docs/nghiepvu/free-models.json` là nguồn sự thật về model free hiện tại,
được refresh định kỳ bởi workflow `.github/workflows/refresh-free-models.yml`
(script: `scripts/refresh-free-models.mjs`).

Chỉ dùng model có id nằm trong `free` của file đó, theo tiền tố `opencode/<id>`.
Không có model free khả dụng → dừng và nhờ con người, **không bao giờ** tự chuyển
sang model trả phí.

## Ghi chú

- Đổi config ở đây xong thì **thoát và khởi động lại opencode** để áp dụng.
- Muốn chạy thử cục bộ một yêu cầu từ issue:
  `opencode run "…nội dung…" --model opencode/<id> --permission default`
