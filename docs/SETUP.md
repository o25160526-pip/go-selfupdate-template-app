# Thiết lập môi trường phát triển

## Yêu cầu

- Go 1.23 trở lên.
- Git.
- Make nếu muốn dùng các target trong Makefile.
- Bash/sh cho các script E2E và build matrix.

CI dùng Go `1.23.2` và UTC.

## Clone và kiểm tra

```bash
git clone https://github.com/o25160526-pip/go-selfupdate-template-app.git
cd go-selfupdate-template-app
go test ./...
go vet ./...
```

Chạy kiểm tra đầy đủ:

```bash
make verify
```

`make verify` chạy race test, vet và `scripts/e2e-local.sh`.

## Build và chạy

```bash
make build
./dist/app version
./dist/app version --json
./dist/app features
```

Tạo bản sao template:

```bash
make init APP=myapp MODULE=github.com/example/myapp
```

Tạo feature mới:

```bash
make new-feature NAME=diagnostics
go test ./...
```

## Cấu hình local

Copy `configs/config.example.yaml` thành file cấu hình riêng hoặc dùng biến `APP_*`. Có thể truyền file bằng `--config path` hoặc `APP_CONFIG`. Các source update cần GitHub owner/repo hoặc Azure index URL; manifest và public key được cấu hình khi cần policy/signature.

Không đưa token, private key hoặc thông tin production vào repository.
