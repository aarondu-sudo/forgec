# forgec
Minimal Go→C export codegen. It scans `internal/` for functions annotated with `capi:export`, validates signature `func(...int32) (int32, error)`, and generates:

- `exports.go` in package `main` with `//export` symbols (errno-style return, panic capture, last-error helpers)
- `forgec.h` header with C prototypes and helpers (`capi_last_error_json`, `capi_free`)

Quick start:

- Build generator: `go build ./forgec/cmd/forgec`
- Example: `examples/myapi`
  - `go generate ./examples/myapi`
  - Linux/macOS: `bash examples/myapi/build.sh` (or `go build -buildmode=c-shared -o examples/myapi/dist/libmyapi.so ./examples/myapi`)
  - Windows: `pwsh -File examples/myapi/build.ps1` (or `go build -buildmode=c-shared -o examples/myapi/dist/myapi.dll ./examples/myapi`)
  - C smoke (Linux/macOS): `cc examples/myapi/c_smoke.c -Iexamples/myapi -Lexamples/myapi/dist -lmyapi -Wl,-rpath,@loader_path/dist -o /tmp/smoke && /tmp/smoke`

Notes:

- Exported C symbol names default to `PM_<GoName>`; return value is via `int32_t* out`, function returns errno (`0` success, `1` error`).
- Panic-safe exports: with `-sentry` enabled, uses `sentrywrap.RecoverAndReport` and `LastErrorJSON`; otherwise uses a built-in lightweight recorder.
- Generated files are idempotent and `gofmt` formatted.

Sentry integration (optional):

- Enable via `-sentry` (alias: `-withsentry`). When enabled, `forgec` writes `<module>/sentrywrap/sentrywrap.go` and imports it from `exports.go`.
- When not enabled, `exports.go` includes a tiny recorder and does not import `sentrywrap`.

Example invocation for the sample module:

```
go run ./forgec/cmd/forgec \
  -pkg ./examples/myapi/internal \
  -o ./examples/myapi/exports.go \
  -hout ./examples/myapi/forgec.h \
  -mod example.com/myapi \
  -sentry
```

Build scripts:

- `build.sh` (macOS/Linux) and `build.ps1` (Windows) are generated in the module root and build a c-shared library into `./dist/`.
