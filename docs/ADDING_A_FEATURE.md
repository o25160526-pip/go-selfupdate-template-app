# Adding a feature

A feature is a package under `internal/features/<name>` that registers itself through `init()`. The application entrypoint is never edited.

## Generate

```bash
make new-feature NAME=diagnostics
```

The name must match `^[a-z][a-z0-9_]*$`. The generator creates:

```text
internal/features/diagnostics/diagnostics.go
internal/features/diagnostics/diagnostics_test.go
```

It also updates the generated blank-import list in `cmd/app/features_gen.go`. That import causes `init()` to call `features.Register(&Feature{})`.

## Contract

Each feature implements:

```go
type Feature interface {
    ID() string
    TrayItems() []tray.Item
    Commands() []features.Command
    Start(context.Context) error
}
```

Rules:

1. `ID()` is stable and unique.
2. `Start()` must return quickly or start background work that stops with its context.
3. Commands must not call `os.Exit`; return errors to the root application.
4. Tray actions should map to an existing command string.
5. A feature package must contain at least one unit test.
6. Never add feature wiring to `main.go`; regenerate the import list instead.

## Verify

```bash
gofmt -w internal/features/diagnostics cmd/app/features_gen.go
go test ./...
./dist/app features
./dist/app diagnostics
```

The built-in `sample` and generated `healthcheck` packages are reference implementations.
