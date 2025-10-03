package main

import (
    "bufio"
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"

    "github.com/aarondu-sudo/forgec/forgec/internal/scanner"
    "github.com/aarondu-sudo/forgec/forgec/internal/writer"
    "github.com/aarondu-sudo/forgec/forgec/internal/version"
)

func main() {
    var (
        initName string
        pkgPath string
        outGo   string
        outH    string
        modPath string
        cPrefix string
        withSentryFlag bool
        withSentryLong bool
        showVersion bool
    )

    flag.StringVar(&initName, "init", "", "initialize a new DLL project (e.g., -init gamedl)")
    flag.StringVar(&pkgPath, "pkg", "./internal", "path to the Go package to scan (e.g., ./internal)")
    flag.StringVar(&outGo, "o", "./exports.go", "output path for generated exports.go")
    flag.StringVar(&outH, "hout", "./forgec.h", "output path for generated C header")
    flag.StringVar(&modPath, "mod", "", "Go module path of the target project (e.g., example.com/myapi)")
    flag.StringVar(&cPrefix, "cprefix", "PM_", "C export symbol prefix (e.g., PM_)")
    // Sentry integration toggle (short and long forms)
    flag.BoolVar(&withSentryFlag, "sentry", false, "include sentrywrap helpers and reporting")
    flag.BoolVar(&withSentryLong, "withsentry", false, "include sentrywrap helpers and reporting")
    flag.BoolVar(&showVersion, "version", false, "print forgec version and exit")
    flag.Parse()

    withSentry := withSentryFlag || withSentryLong

    if showVersion {
        fmt.Println(version.Version)
        return
    }

    // Handle project initialization and exit
    if initName != "" {
        if err := writer.InitProject(initName); err != nil {
            log.Fatalf("init project: %v", err)
        }
        fmt.Printf("Initialized project at ./%s (idempotent). Templates regenerated.\n", initName)
        return
    }

    // Determine module path: use -mod if provided, otherwise detect from go.mod near outputs.
    if modPath == "" {
        detected, derr := detectModulePath(filepath.Dir(outGo))
        if derr != nil || detected == "" {
            log.Fatal("module path not provided and go.mod not found; pass -mod or run within a module")
        }
        modPath = detected
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

// detectModulePath tries to find a go.mod (starting from startDir and up) and parse its module path.
func detectModulePath(startDir string) (string, error) {
    dir := startDir
    if dir == "" {
        var err error
        dir, err = os.Getwd()
        if err != nil {
            return "", err
        }
    }
    for {
        gm := filepath.Join(dir, "go.mod")
        if fi, err := os.Stat(gm); err == nil && !fi.IsDir() {
            f, err := os.Open(gm)
            if err != nil {
                return "", err
            }
            defer f.Close()
            s := bufio.NewScanner(f)
            for s.Scan() {
                line := strings.TrimSpace(s.Text())
                if strings.HasPrefix(line, "module ") {
                    return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
                }
            }
            if err := s.Err(); err != nil {
                return "", err
            }
            break
        }
        parent := filepath.Dir(dir)
        if parent == dir {
            break
        }
        dir = parent
    }
    return "", fmt.Errorf("go.mod not found from %s", startDir)
}
