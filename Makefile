SHELL := /bin/sh
.PHONY: test verify build build-matrix e2e init new-feature clean

test:
	go test ./...

verify:
	go test -race ./...
	go vet ./...
	sh ./scripts/e2e-local.sh

build:
	go build -trimpath -o dist/app ./cmd/app

build-matrix:
	sh ./scripts/build-matrix.sh

e2e:
	sh ./scripts/e2e-local.sh

init:
	@test -n "$(APP)" || (echo "APP required" >&2; exit 2)
	@test -n "$(MODULE)" || (echo "MODULE required" >&2; exit 2)
	APP="$(APP)" MODULE="$(MODULE)" sh ./scripts/init-template.sh

new-feature:
	@test -n "$(NAME)" || (echo "NAME required" >&2; exit 2)
	go run ./cmd/newfeature --name "$(NAME)"
	gofmt -w ./internal/features/"$(NAME)" ./cmd/app/features_gen.go

go-clean clean:
	rm -rf dist
