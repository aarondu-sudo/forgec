# forgec
Minimal Go→C export codegen. It scans `internal/` for functions annotated with `capi:export`, validates signature `func(...int32) (int32, error)`, and generates:

- `exports.go` in package `main` with `//export` symbols (errno-style return, panic capture, last-error helpers)
- `forgec.h` header with C prototypes and helpers (`capi_last_error_json`, `capi_free`)

Quick start:

- Install CLI globally (published): `go install github.com/aarondu-sudo/forgec/forgec/cmd/forgec@latest`
  - For reproducible installs: `go install github.com/aarondu-sudo/forgec/forgec/cmd/forgec@v0.1.1`
  - Local development (no pre-build needed):
    - From repo root: `go install ./cmd/forgec`
    - If your workspace causes module issues: `GOWORK=off go install ./cmd/forgec`
- Check version: `forgec --version`

Example build (your module):

- Generate: `forgec -pkg ./internal -o ./exports.go -hout ./forgec.h [-sentry]`
- Linux/macOS: `go build -buildmode=c-shared -o ./dist/lib<name>.so ./`
- Windows: `go build -buildmode=c-shared -o ./dist/<name>.dll ./`
- Optional smoke (Linux/macOS): `cc path/to/c_smoke.c -I. -L./dist -l<name> -Wl,-rpath,@loader_path/dist -o /tmp/smoke && /tmp/smoke`

Notes:

- Exported C symbol names default to `PM_<GoName>`; return value is via `int32_t* out`, function returns errno (`0` success, `1` error`).
- Panic-safe exports: with `-sentry` enabled, uses `sentrywrap.RecoverAndReport` and `LastErrorJSON`; otherwise uses a built-in lightweight recorder.
- Generated files are idempotent and `gofmt` formatted.

Sentry integration (optional):

- Enable via `-sentry` (alias: `-withsentry`). When enabled, `forgec` writes `<module>/sentrywrap/sentrywrap.go` and imports it from `exports.go`.
- When not enabled, `exports.go` includes a tiny recorder and does not import `sentrywrap`.

Direct usage (installed CLI):

```
# From your module root (auto-detect module path via go.mod)
forgec -pkg ./internal -o ./exports.go -hout ./forgec.h

# With sentry error capture
forgec -pkg ./internal -o ./exports.go -hout ./forgec.h -sentry

# If running outside a module or custom path, pass -mod
forgec -pkg ./internal -o ./exports.go -hout ./forgec.h -mod example.com/myapi
```

Alternative (no install):

```
go run ./cmd/forgec -pkg ./internal -o ./exports.go -hout ./forgec.h -mod example.com/myapi [-sentry]
```

Build scripts:

- `build.sh` (macOS/Linux) and `build.ps1` (Windows) are generated in the module root and build a c-shared library into `./dist/`.

Project init:

- Scaffold: `forgec -init gamedl`
  - Creates `./gamedl/internal/` and a starter `internal/calc.go` if it doesn’t exist.
  - Generates template files next to `internal/` (always refreshed on re-run):
    - `generate.go` (no sentry) and `generate_sentry.go` (with sentry); both call `forgec` via `//go:generate`.
    - `build.sh` and `build.ps1` for building DLLs to `./dist/`.
  - Idempotent: does not delete or overwrite user code under `internal/`.

Tips:

- Run `go generate ./...` in your module root to regenerate `exports.go`/`forgec.h` using the generated `generate.go` files.
- When `-sentry` is used, `sentrywrap/sentrywrap.go` is generated and imported by `exports.go`.
- Omit `-mod` to let `forgec` auto-detect your module path from the nearest `go.mod`.
