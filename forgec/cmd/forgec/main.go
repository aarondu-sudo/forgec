package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"

    "example.com/forgec/internal/scanner"
    "example.com/forgec/internal/writer"
)

func main() {
    var (
        pkgPath string
        outGo   string
        outH    string
        modPath string
        cPrefix string
        withSentryFlag bool
        withSentryLong bool
    )

    flag.StringVar(&pkgPath, "pkg", "./internal", "path to the Go package to scan (e.g., ./internal)")
    flag.StringVar(&outGo, "o", "./exports.go", "output path for generated exports.go")
    flag.StringVar(&outH, "hout", "./forgec.h", "output path for generated C header")
    flag.StringVar(&modPath, "mod", "", "Go module path of the target project (e.g., example.com/myapi)")
    flag.StringVar(&cPrefix, "cprefix", "PM_", "C export symbol prefix (e.g., PM_)")
    // Sentry integration toggle (short and long forms)
    flag.BoolVar(&withSentryFlag, "sentry", false, "include sentrywrap helpers and reporting")
    flag.BoolVar(&withSentryLong, "withsentry", false, "include sentrywrap helpers and reporting")
    flag.Parse()

    withSentry := withSentryFlag || withSentryLong

    if modPath == "" {
        log.Fatal("-mod is required (module path of the target project, e.g., example.com/myapi)")
    }

    absPkg, err := filepath.Abs(pkgPath)
    if err != nil {
        log.Fatalf("resolve pkg path: %v", err)
    }

    funcs, structs, err := scanner.ScanExported(absPkg)
    if err != nil {
        log.Fatalf("scan failed: %v", err)
    }

    if len(funcs) == 0 {
        log.Println("no capi:export functions found; nothing to generate")
    }

    // Ensure output directories exist
    for _, p := range []string{outGo, outH} {
        dir := filepath.Dir(p)
        if dir != "." && dir != "" {
            if err := os.MkdirAll(dir, 0o755); err != nil {
                log.Fatalf("mkdir %s: %v", dir, err)
            }
        }
    }

    if err := writer.WriteExportsGo(outGo, modPath, cPrefix, funcs, withSentry); err != nil {
        log.Fatalf("write exports.go: %v", err)
    }
    if err := writer.WriteHeader(outH, cPrefix, funcs, structs); err != nil {
        log.Fatalf("write header: %v", err)
    }

    // Optionally generate sentrywrap package into the target module directory
    if withSentry {
        // Use the directory of exports.go as the module root for generation
        modRoot := filepath.Dir(outGo)
        if err := writer.WriteSentryWrap(modRoot); err != nil {
            log.Fatalf("write sentrywrap: %v", err)
        }
    }

    // Always generate build scripts into the target module dir
    {
        modRoot := filepath.Dir(outGo)
        modName := filepath.Base(modPath)
        if err := writer.WriteBuildScripts(modRoot, modName); err != nil {
            log.Fatalf("write build scripts: %v", err)
        }
    }

    if withSentry {
        fmt.Printf("Generated %s, %s, sentrywrap/, and build scripts (functions: %d, structs: %d)\n", outGo, outH, len(funcs), len(structs))
    } else {
        fmt.Printf("Generated %s, %s, and build scripts (functions: %d, structs: %d)\n", outGo, outH, len(funcs), len(structs))
    }
}
