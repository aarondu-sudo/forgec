package writer

import (
    "bytes"
    "fmt"
    "go/format"
    "os"
    "sort"
    "strings"

    "example.com/forgec/internal/scanner"
)

// WriteExportsGo generates exports.go with cgo exports, panic recovery, errno, and helpers.
func WriteExportsGo(path, modPath, cPrefix string, funcs []scanner.Func) error {
    // Sort for stability
    sort.Slice(funcs, func(i, j int) bool { return funcs[i].Name < funcs[j].Name })

    var b bytes.Buffer
    b.WriteString("package main\n\n")
    b.WriteString("/*\n#include <stdlib.h>\n#include <stdint.h>\n*/\n")
    b.WriteString("import \"C\"\n\n")
    b.WriteString("import (\n")
    fmt.Fprintf(&b, "    p \"%s/internal\"\n", modPath)
    fmt.Fprintf(&b, "    \"%s/sentrywrap\"\n", modPath)
    b.WriteString("    \"unsafe\"\n")
    b.WriteString(")\n\n")

    // Helpers
    b.WriteString("//export capi_free\n")
    b.WriteString("func capi_free(p unsafe.Pointer) { C.free(p) }\n\n")

    b.WriteString("//export capi_last_error_json\n")
    b.WriteString("func capi_last_error_json() *C.char {\n")
    b.WriteString("    s := sentrywrap.LastErrorJSON()\n")
    b.WriteString("    return C.CString(s)\n")
    b.WriteString("}\n\n")

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
        b.WriteString("    sentrywrap.RecoverAndReport(func() {\n")
        if f.HasValue {
            fmt.Fprintf(&b, "        res, err := p.%s(%s)\n", f.Name, strings.Join(goArgs, ", "))
        } else {
            fmt.Fprintf(&b, "        err := p.%s(%s)\n", f.Name, strings.Join(goArgs, ", "))
        }
        b.WriteString("        if err != nil {\n")
        b.WriteString("            errno = 1\n")
        b.WriteString("            sentrywrap.SetLastError(err)\n")
        b.WriteString("            return\n")
        b.WriteString("        }\n")
        if f.HasValue {
            if f.RetType == "int64" {
                b.WriteString("        if out != nil { *out = C.int64_t(res) }\n")
            } else {
                b.WriteString("        if out != nil { *out = C.int32_t(res) }\n")
            }
        }
        b.WriteString("    })\n")
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
