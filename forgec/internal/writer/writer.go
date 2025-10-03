package writer

import (
    "bytes"
    "fmt"
    "go/format"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "text/template"

    tpl "example.com/forgec/template"
    "example.com/forgec/internal/scanner"
)

func renderTemplate(name string, data any) (string, error) {
    t, err := template.New(name).ParseFS(tpl.FS, name)
    if err != nil {
        return "", fmt.Errorf("parse template %s: %w", name, err)
    }
    var b bytes.Buffer
    if err := t.Execute(&b, data); err != nil {
        return "", fmt.Errorf("execute template %s: %w", name, err)
    }
    return b.String(), nil
}

// WriteExportsGo generates exports.go with cgo exports, panic recovery, errno, and helpers.
func WriteExportsGo(path, modPath, cPrefix string, funcs []scanner.Func, withSentry bool) error {
    // Sort for stability
    sort.Slice(funcs, func(i, j int) bool { return funcs[i].Name < funcs[j].Name })

    var b bytes.Buffer
    b.WriteString("package main\n\n")
    b.WriteString("/*\n#include <stdlib.h>\n#include <stdint.h>\n*/\n")
    b.WriteString("import \"C\"\n\n")
    b.WriteString("import (\n")
    fmt.Fprintf(&b, "    p \"%s/internal\"\n", modPath)
    if withSentry {
        fmt.Fprintf(&b, "    \"%s/sentrywrap\"\n", modPath)
    } else {
        b.WriteString("    \"encoding/json\"\n")
        b.WriteString("    \"sync\"\n")
    }
    b.WriteString("    \"unsafe\"\n")
    b.WriteString(")\n\n")

    // Helpers
    b.WriteString("//export capi_free\n")
    b.WriteString("func capi_free(p unsafe.Pointer) { C.free(p) }\n\n")

    if withSentry {
        b.WriteString("//export capi_last_error_json\n")
        b.WriteString("func capi_last_error_json() *C.char {\n")
        b.WriteString("    s := sentrywrap.LastErrorJSON()\n")
        b.WriteString("    return C.CString(s)\n")
        b.WriteString("}\n\n")
    } else {
        b.WriteString("var (\n")
        b.WriteString("    lastErrMu sync.Mutex\n")
        b.WriteString("    lastErr   string\n")
        b.WriteString(")\n\n")
        b.WriteString("func setLastError(err error) {\n")
        b.WriteString("    lastErrMu.Lock()\n")
        b.WriteString("    defer lastErrMu.Unlock()\n")
        b.WriteString("    if err == nil { lastErr = \"\"; return }\n")
        b.WriteString("    b, _ := json.Marshal(map[string]any{\"error\": err.Error()})\n")
        b.WriteString("    lastErr = string(b)\n")
        b.WriteString("}\n\n")
        b.WriteString("func lastErrorJSON() string {\n")
        b.WriteString("    lastErrMu.Lock()\n")
        b.WriteString("    defer lastErrMu.Unlock()\n")
        b.WriteString("    if lastErr == \"\" { return \"{}\" }\n")
        b.WriteString("    return lastErr\n")
        b.WriteString("}\n\n")
        b.WriteString("type simpleError string\n")
        b.WriteString("func (e simpleError) Error() string { return string(e) }\n")
        b.WriteString("func errFromRecover(r any) error {\n")
        b.WriteString("    switch x := r.(type) {\n")
        b.WriteString("    case error:\n        return x\n")
        b.WriteString("    case string:\n        return simpleError(x)\n")
        b.WriteString("    default:\n        return simpleError(\"panic\")\n")
        b.WriteString("    }\n")
        b.WriteString("}\n\n")
        b.WriteString("//export capi_last_error_json\n")
        b.WriteString("func capi_last_error_json() *C.char {\n")
        b.WriteString("    s := lastErrorJSON()\n")
        b.WriteString("    return C.CString(s)\n")
        b.WriteString("}\n\n")
    }

    for _, f := range funcs {
        cname := cPrefix + f.CName
        // C param list includes all params + out pointer
        b.WriteString("//export " + cname + "\n")
        fmt.Fprintf(&b, "func %s(", cname)
        // params as C.int32_t or C.int64_t
        var goArgs []string
        for i, pn := range f.Params {
            if i > 0 { b.WriteString(", ") }
            cpt := "C.int32_t"
            gCast := "int32"
            if i < len(f.ParamTypes) && f.ParamTypes[i] == "int64" {
                cpt = "C.int64_t"
                gCast = "int64"
            }
            fmt.Fprintf(&b, "%s %s", pn, cpt)
            goArgs = append(goArgs, fmt.Sprintf("%s(%s)", gCast, pn))
        }
        if f.HasValue {
            if len(f.Params) > 0 { b.WriteString(", ") }
            outType := "*C.int32_t"
            if f.RetType == "int64" { outType = "*C.int64_t" }
            b.WriteString("out " + outType)
        }
        b.WriteString(") C.int32_t {\n")
        b.WriteString("    var errno C.int32_t = 0\n")
        if withSentry {
            b.WriteString("    sentrywrap.RecoverAndReport(func() {\n")
        } else {
            b.WriteString("    func() {\n")
            b.WriteString("        defer func() { if r := recover(); r != nil { setLastError(errFromRecover(r)) } }()\n")
        }
        if f.HasValue {
            fmt.Fprintf(&b, "        res, err := p.%s(%s)\n", f.Name, strings.Join(goArgs, ", "))
        } else {
            fmt.Fprintf(&b, "        err := p.%s(%s)\n", f.Name, strings.Join(goArgs, ", "))
        }
        b.WriteString("        if err != nil {\n")
        b.WriteString("            errno = 1\n")
        if withSentry {
            b.WriteString("            sentrywrap.SetLastError(err)\n")
        } else {
            b.WriteString("            setLastError(err)\n")
        }
        b.WriteString("            return\n")
        b.WriteString("        }\n")
        if f.HasValue {
            if f.RetType == "int64" {
                b.WriteString("        if out != nil { *out = C.int64_t(res) }\n")
            } else {
                b.WriteString("        if out != nil { *out = C.int32_t(res) }\n")
            }
        }
        if withSentry {
            b.WriteString("    })\n")
        } else {
            b.WriteString("    }()\n")
        }
        b.WriteString("    return errno\n")
        b.WriteString("}\n\n")
    }

    // Provide a dummy main to satisfy c-shared build requirements.
    b.WriteString("func main() {}\n")

    src := b.Bytes()
    fmted, err := format.Source(src)
    if err != nil {
        // write raw to help debugging
        if werr := os.WriteFile(path, src, 0o644); werr != nil {
            return fmt.Errorf("write (unformatted) %s: %v; format error: %v", path, werr, err)
        }
        return fmt.Errorf("format generated code: %w", err)
    }
    if err := os.WriteFile(path, fmted, 0o644); err != nil {
        return fmt.Errorf("write %s: %w", path, err)
    }
    return nil
}

// WriteSentryWrap generates the sentrywrap package under the provided module root directory.
// It writes to <modRoot>/sentrywrap/sentrywrap.go, overwriting if it exists.
func WriteSentryWrap(modRoot string) error {
    dir := filepath.Join(modRoot, "sentrywrap")
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return fmt.Errorf("mkdir sentrywrap: %w", err)
    }
    content, err := renderTemplate("sentrywrap.go.tmpl", nil)
    if err != nil {
        return err
    }
    out := filepath.Join(dir, "sentrywrap.go")
    if err := os.WriteFile(out, []byte(content), 0o644); err != nil {
        return fmt.Errorf("write sentrywrap.go: %w", err)
    }
    return nil
}

// WriteBuildScripts writes simple build scripts into the target module root:
// - build.sh for macOS/Linux
// - build.ps1 for Windows
// They build a c-shared library into the dist directory.
func WriteBuildScripts(modRoot, modName string) error {
    if err := os.MkdirAll(filepath.Join(modRoot, "dist"), 0o755); err != nil {
        return fmt.Errorf("mkdir dist: %w", err)
    }

    data := map[string]any{"ModName": modName}
    sh, err := renderTemplate("build.sh.tmpl", data)
    if err != nil { return err }
    if err := os.WriteFile(filepath.Join(modRoot, "build.sh"), []byte(sh), 0o755); err != nil {
        return fmt.Errorf("write build.sh: %w", err)
    }

    ps1, err := renderTemplate("build.ps1.tmpl", data)
    if err != nil { return err }
    if err := os.WriteFile(filepath.Join(modRoot, "build.ps1"), []byte(ps1), 0o644); err != nil {
        return fmt.Errorf("write build.ps1: %w", err)
    }
    return nil
}

// WriteHeader generates forgec.h with C prototypes.
func WriteHeader(path, cPrefix string, funcs []scanner.Func, structs []scanner.Struct) error {
    sort.Slice(funcs, func(i, j int) bool { return funcs[i].Name < funcs[j].Name })

    var b bytes.Buffer
    b.WriteString("#pragma once\n\n")
    b.WriteString("#include <stdint.h>\n")
    b.WriteString("#include <stddef.h>\n\n")
    b.WriteString("#ifdef __cplusplus\nextern \"C\" {\n#endif\n\n")

    for _, f := range funcs {
        // int32_t PM_Name(int32_t a, ..., int32_t* out);
        b.WriteString("int32_t ")
        b.WriteString(cPrefix)
        b.WriteString(f.CName)
        b.WriteString("(")
        for i := range f.Params {
            if i > 0 { b.WriteString(", ") }
            if i < len(f.ParamTypes) && f.ParamTypes[i] == "int64" {
                b.WriteString("int64_t ")
            } else {
                b.WriteString("int32_t ")
            }
            b.WriteString(f.Params[i])
        }
        if f.HasValue {
            if len(f.Params) > 0 { b.WriteString(", ") }
            if f.RetType == "int64" {
                b.WriteString("int64_t* out")
            } else {
                b.WriteString("int32_t* out")
            }
        }
        b.WriteString(");\n")
    }

    b.WriteString("\nconst char* capi_last_error_json(void);\n")
    b.WriteString("void capi_free(void* p);\n\n")

    // Struct typedefs
    sort.Slice(structs, func(i, j int) bool { return structs[i].Name < structs[j].Name })
    for _, s := range structs {
        fmt.Fprintf(&b, "typedef struct %s {\n", s.Name)
        for _, f := range s.Fields {
            fmt.Fprintf(&b, "    %s %s;\n", f.CType, f.ExportName)
        }
        fmt.Fprintf(&b, "} %s;\n\n", s.Name)
    }
    b.WriteString("#ifdef __cplusplus\n}\n#endif\n")

    if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
        return fmt.Errorf("write %s: %w", path, err)
    }
    return nil
}

// InitProject scaffolds a new DLL project directory with standard layout and a sample calc.go.
func InitProject(name string) error {
    root := filepath.Clean(name)
    internalDir := filepath.Join(root, "internal")
    if err := os.MkdirAll(internalDir, 0o755); err != nil {
        return fmt.Errorf("mkdir project: %w", err)
    }

    // Do not overwrite user's internal logic. Only create calc.go if missing.
    calcPath := filepath.Join(internalDir, "calc.go")
    if _, err := os.Stat(calcPath); os.IsNotExist(err) {
        calc, err := renderTemplate("init_calc.go.tmpl", map[string]any{"Package": "internal"})
        if err != nil { return err }
        if err := os.WriteFile(calcPath, []byte(calc), 0o644); err != nil {
            return fmt.Errorf("write calc.go: %w", err)
        }
    }

    // Always (re)generate template-based files next to internal/: build scripts and go:generate helpers.
    modName := filepath.Base(root)
    if err := WriteBuildScripts(root, modName); err != nil {
        return err
    }

    // Generate two go:generate helpers: one with sentry, one without. Always overwrite for freshness.
    genNoSentry, err := renderTemplate("generate.go.tmpl", map[string]any{"WithSentry": false})
    if err != nil { return err }
    if err := os.WriteFile(filepath.Join(root, "generate.go"), []byte(genNoSentry), 0o644); err != nil {
        return fmt.Errorf("write generate.go: %w", err)
    }

    genSentry, err := renderTemplate("generate_sentry.go.tmpl", map[string]any{"WithSentry": true})
    if err != nil { return err }
    if err := os.WriteFile(filepath.Join(root, "generate_sentry.go"), []byte(genSentry), 0o644); err != nil {
        return fmt.Errorf("write generate_sentry.go: %w", err)
    }
    return nil
}
