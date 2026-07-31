# Nghiệp vụ: Khảo sát và chuẩn hóa nghiệp vụ (khao-sat-nghiep-vu)

> Slug: `khao-sat-nghiep-vu` · Label: `nghiepvu:khao-sat-nghiep-vu`
> ALLOWED: docs/nghiepvu/** .github/ISSUE_TEMPLATE/** .github/ISSUE_TEMPLATE/config.yml

## Mô tả

"Siêu nghiệp vụ" quản lý chính hệ thống nghiệp vụ: quét codebase để **phát hiện
logic nghiệp vụ mới**, đối chiếu với các file `docs/nghiepvu/<slug>.md` hiện có,
và **chuẩn hóa lại** các nghiệp vụ (nội dung, phạm vi file, quy tắc, template
issue) cho khớp với codebase thực tế.

Dùng khi:
- Codebase thay đổi nhiều mà `docs/nghiepvu/` chưa phản ánh.
- Nghi ngờ có logic nghiệp vụ chưa được khai báo (phạm vi file chưa bao phủ).
- Cần rà soát lại phạm vi `ALLOWED:` sau khi refactor/đổi cấu trúc thư mục.

## Thư mục / file liên quan

- `docs/nghiepvu/` — toàn bộ tài liệu nghiệp vụ (`<slug>.md`, `INDEX.md`,
  `README-opencode-workflow.md`, `PROMPT-GOC.md`, `free-models.json`).
- `.github/ISSUE_TEMPLATE/` — template issue cho từng nghiệp vụ.
- Toàn codebase (đọc để khảo sát — nghiệp vụ này KHÔNG sửa code, chỉ sửa docs).

## Quy tắc bắt buộc

- KHÔNG sửa code nguồn — kết quả của nghiệp vụ này chỉ là **tài liệu + template**.
- Mọi nghiệp vụ mới phải có đủ: slug, label `nghiepvu:<slug>`, mô tả, thư mục
  liên quan, quy tắc bắt buộc, và dòng `ALLOWED:` (phạm vi file máy đọc được).
- Đổi phạm vi file của nghiệp vụ hiện có phải đối chiếu cấu trúc thư mục thật
  (bằng `git ls-tree` / đọc code), không đoán.
- Cập nhật `docs/nghiepvu/INDEX.md` và tạo đủ 3 template
  `<slug>-{research,fix,feature}.yml` cho nghiệp vụ mới.
- Giữ nguyên `docs/nghiepvu/PROMPT-GOC.md` (spec gốc) — chỉ thêm, không sửa
  các ràng buộc cứng đã có.
- Không xóa nghiệp vụ đang được nhắc tới trong issue/PR; muốn gộp/xóa → đề xuất
  rõ trong kế hoạch để người duyệt quyết.
