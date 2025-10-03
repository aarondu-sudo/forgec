# Repository Guidelines

## Project Structure & Module Organization
- Generator module at repo root:
  - `cmd/forgec/`: CLI entrypoint (`main.go`)
  - `internal/`: generator internals (`scanner`, `writer`, `version`)
  - `template/`: text/template files used by writer (embedded)
- Example consumer modules are not included in this repo by default. When present, a typical layout is:
  - `<module>/internal/`: API functions (annotated with `capi:export`)
  - `<module>/sentrywrap/`: panic/error capture helpers
  - `<module>/dist/`: build outputs (`.so/.dll/.dylib`)
- Root: `go.mod`, `README.md`, `LICENSE`.

## Build, Test, and Development Commands
- Build generator: `go build ./cmd/forgec`
- Install CLI globally (published): `go install github.com/aarondu-sudo/forgec/forgec/cmd/forgec@latest`
  - Pin version: `@v0.1.1`
  - Local dev alternative: `go install ./cmd/forgec` (use `GOWORK=off` if your workspace interferes)
- Check CLI version: `forgec --version`
- Run via go: `go run ./cmd/forgec -h`
- Generate bindings in your module (installed CLI): `forgec -pkg ./internal -o ./exports.go -hout ./forgec.h [-sentry]`
- Build shared library (macOS/Linux, example): `go build -buildmode=c-shared -o ./dist/lib<name>.so ./`
- C smoke test (macOS/Linux, example): `cc c_smoke.c -I. -L./dist -l<name> -Wl,-rpath,@loader_path/dist -o /tmp/smoke && /tmp/smoke`

### Project Init
- Initialize a standard DLL project layout: `forgec -init gamedl`
  - Creates `./gamedl/` with `./gamedl/internal/`
  - Writes a starter `internal/calc.go` with `capi:export` examples (only if missing)
  - Generates template files next to `internal/` (always overwritten on re-init):
    - `generate.go` (no sentry) and `generate_sentry.go` (with sentry), each with a `//go:generate forgec ...` command
    - `build.sh` and `build.ps1` (DLL build scripts)
  - Idempotent: never deletes or overwrites user code under `internal/`; template files are refreshed every time

## Coding Style & Naming Conventions
- Go version: 1.22; use idiomatic Go (gofmt). Run `go fmt ./...`.
- Generated files are formatted via `go/format`; do not hand-edit generated `exports.go` or `forgec.h` in consumer modules.
- C export names default to `PM_<GoName>`; return value via `int32_t* out`; function returns errno (`0` ok, `1` error`).
- Keep packages small; filenames lowercase with underscores only if needed.

## Testing Guidelines
- Framework: standard `testing`. Add `_test.go` files with `TestXxx` functions.
- Run tests for the generator: `go test ./internal/...` (or `go test ./...`).
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
  - Module path: pass via `-mod` or omit to auto-detect from nearest `go.mod`.

### Templates
- Some fixed code is rendered via `text/template` and embedded with `go:embed` under `template/`:
  - `sentrywrap.go.tmpl` → `<module>/sentrywrap/sentrywrap.go`
  - `build.sh.tmpl` → `<module>/build.sh`
  - `build.ps1.tmpl` → `<module>/build.ps1`
  - `init_calc.go.tmpl` → `<project>/internal/calc.go` for `-init`
  - `generate.go.tmpl` → `<project>/generate.go` (no sentry)
  - `generate_sentry.go.tmpl` → `<project>/generate_sentry.go` (with sentry)
- To modify output, edit templates and rebuild `forgec`.

## Versioning
- The CLI exposes a version via `forgec --version` (see `internal/version`).
- Bump the version on any functional change to the CLI, templates, or output format.

## Usage Examples
- Generate using installed CLI (auto-detect module path):
  - `forgec -pkg ./internal -o ./exports.go -hout ./forgec.h`
  - With sentry: `forgec -pkg ./internal -o ./exports.go -hout ./forgec.h -sentry`
- `go generate` integration (created by `-init`): run `go generate ./...` in your module root.

## Contributor Tips
- When extending signatures or prefixes, update both `scanner` and `writer` and refresh example via `go generate`.
