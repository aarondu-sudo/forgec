# forgec
Minimal Go→C export codegen. It scans `internal/` for functions annotated with `capi:export`, validates signature `func(...int32) (int32, error)`, and generates:

- `exports.go` in package `main` with `//export` symbols (errno-style return, panic capture, last-error helpers)
- `forgec.h` header with C prototypes and helpers (`capi_last_error_json`, `capi_free`)

Quick start:

- Build generator: `go build ./forgec/cmd/forgec`
- Example: `examples/myapi`
  - `go generate ./examples/myapi`
  - Linux/macOS: `go build -buildmode=c-shared -o examples/myapi/dist/libmyapi.so ./examples/myapi`
  - Windows: `go build -buildmode=c-shared -o examples/myapi/dist/myapi.dll ./examples/myapi`
  - C smoke (Linux/macOS): `cc examples/myapi/c_smoke.c -Iexamples/myapi -Lexamples/myapi/dist -lmyapi -Wl,-rpath,@loader_path/dist -o /tmp/smoke && /tmp/smoke`

Notes:

- Exported C symbol names default to `PM_<GoName>`; return value is via `int32_t* out`, function returns errno (`0` success, `1` error`).
- Panic-safe via `sentrywrap.RecoverAndReport`; last error retrievable as JSON string.
- Generated files are idempotent and `gofmt` formatted.
