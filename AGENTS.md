# Repository Guidelines

## Project Structure & Module Organization
- `forgec/`: Go code generator module
  - `cmd/forgec/`: CLI entrypoint (`main.go`)
  - `internal/`: generator internals (`scanner`, `writer`)
  - `template/`: text/template files used by writer (embedded)
- `examples/myapi/`: Sample consumer module
  - `internal/`: example API functions (annotated with `capi:export`)
  - `sentrywrap/`: panic/error capture helpers
  - `dist/`: build outputs (`.so/.dll/.dylib`)
- Root: `go.work`, `README.md`, `LICENSE`.

## Build, Test, and Development Commands
- Build generator: `go build ./forgec/cmd/forgec`
- Run via go: `go run ./forgec/cmd/forgec -h`
- Generate example bindings: `go generate ./examples/myapi`
- Build shared library (macOS/Linux): `go build -buildmode=c-shared -o examples/myapi/dist/libmyapi.so ./examples/myapi`
- C smoke test (macOS/Linux): `cc examples/myapi/c_smoke.c -Iexamples/myapi -Lexamples/myapi/dist -lmyapi -Wl,-rpath,@loader_path/dist -o /tmp/smoke && /tmp/smoke`

### Project Init
- Initialize a standard DLL project layout: `forgec -init gamedl`
  - Creates `./gamedl/` with `./gamedl/internal/`
  - Writes a starter `internal/calc.go` with `capi:export` examples

## Coding Style & Naming Conventions
- Go version: 1.22; use idiomatic Go (gofmt). Run `go fmt ./...`.
- Generated files are formatted via `go/format`; do not hand-edit `examples/myapi/exports.go` or `forgec.h`.
- C export names default to `PM_<GoName>`; return value via `int32_t* out`; function returns errno (`0` ok, `1` error`).
- Keep packages small; filenames lowercase with underscores only if needed.

## Testing Guidelines
- Framework: standard `testing`. Add `_test.go` files with `TestXxx` functions.
- Run tests: `go test ./forgec/...` and `go test ./examples/myapi/...`.
- Prefer table-driven tests for `scanner` and `writer`; include edge cases (invalid signatures, no tags).

## Commit & Pull Request Guidelines
- Commits: imperative, concise subject (<= 72 chars), focused scope. Example: `writer: generate errno-style wrappers`.
- PRs: clear description, rationale, before/after behavior, and any CLI/output changes. Link issues when applicable.
- Include minimal reproducible steps (commands) for reviewers.

## Architecture Overview
- CLI (`cmd/forgec`) scans target package for `capi:export` annotations, enforces signature `func(...int32) (int32, error)`, then renders:
  - `exports.go`: cgo `//export` wrappers with panic capture and last-error helpers.
  - `forgec.h`: matching C prototypes and helpers (`capi_last_error_json`, `capi_free`).
  - Build scripts: `build.sh` and `build.ps1` in module root.

### Templates
- Some fixed code is rendered via `text/template` and embedded with `go:embed` under `forgec/template/`:
  - `sentrywrap.go.tmpl` → `<module>/sentrywrap/sentrywrap.go`
  - `build.sh.tmpl` → `<module>/build.sh`
  - `build.ps1.tmpl` → `<module>/build.ps1`
  - `init_calc.go.tmpl` → `<project>/internal/calc.go` for `-init`
- To modify output, edit templates and rebuild `forgec`.

## Contributor Tips
- When extending signatures or prefixes, update both `scanner` and `writer` and refresh example via `go generate`.
