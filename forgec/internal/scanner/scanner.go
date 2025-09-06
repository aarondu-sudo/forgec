package scanner

import (
    "errors"
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
    "strings"
)

// Func describes a function to be exported.
type Func struct {
    Name       string   // Go name, e.g., Add
    CName      string   // C name without prefix, same as Name
    Params     []string // parameter names
    ParamTypes []string // Go types (should be int32)
}

// ScanExported scans a package directory for top-level functions annotated with `capi:export`.
// Enforces signature: func(...int32) (int32, error)
func ScanExported(pkgDir string) ([]Func, error) {
    info, err := os.Stat(pkgDir)
    if err != nil {
        return nil, err
    }
    if !info.IsDir() {
        return nil, fmt.Errorf("pkg path is not a directory: %s", pkgDir)
    }

    fset := token.NewFileSet()
    pkgs, err := parser.ParseDir(fset, pkgDir, nil, parser.ParseComments)
    if err != nil {
        return nil, err
    }

    var out []Func
    for _, pkg := range pkgs {
        for _, f := range pkg.Files {
            // Only consider files within the provided dir (avoid vendor, etc.)
            if !strings.HasPrefix(fset.Position(f.Package).Filename, filepath.Clean(pkgDir)) {
                continue
            }
            for _, decl := range f.Decls {
                fn, ok := decl.(*ast.FuncDecl)
                if !ok || fn.Recv != nil || fn.Name == nil {
                    continue
                }
                if fn.Doc == nil || !hasExportTag(fn.Doc.List) {
                    continue
                }
                if err := validateSignature(fn.Type); err != nil {
                    return nil, fmt.Errorf("%s: %w", fn.Name.Name, err)
                }
                pnames, ptypes := collectParams(fn.Type)
                out = append(out, Func{
                    Name:       fn.Name.Name,
                    CName:      fn.Name.Name,
                    Params:     pnames,
                    ParamTypes: ptypes,
                })
            }
        }
    }
    return out, nil
}

func hasExportTag(list []*ast.Comment) bool {
    for _, c := range list {
        if strings.Contains(c.Text, "capi:export") {
            return true
        }
    }
    return false
}

func validateSignature(t *ast.FuncType) error {
    // Params: any number, all int32
    if t.Params != nil {
        for _, f := range t.Params.List {
            if !isIdentType(f.Type, "int32") {
                return fmt.Errorf("param must be int32: %s", exprString(f.Type))
            }
        }
    }
    // Results: exactly 2 -> (int32, error)
    if t.Results == nil || len(t.Results.List) != 2 {
        return errors.New("result must be (int32, error)")
    }
    // First result: int32
    if !isIdentType(t.Results.List[0].Type, "int32") {
        return fmt.Errorf("first result must be int32: %s", exprString(t.Results.List[0].Type))
    }
    // Second result: error
    if !isIdentType(t.Results.List[1].Type, "error") {
        return fmt.Errorf("second result must be error: %s", exprString(t.Results.List[1].Type))
    }
    return nil
}

func collectParams(t *ast.FuncType) ([]string, []string) {
    var names []string
    var types []string
    if t.Params == nil {
        return names, types
    }
    idx := 0
    for _, f := range t.Params.List {
        tname := exprString(f.Type)
        if len(f.Names) == 0 {
            names = append(names, fmt.Sprintf("p%d", idx))
            types = append(types, tname)
            idx++
            continue
        }
        for _, n := range f.Names {
            names = append(names, n.Name)
            types = append(types, tname)
            idx++
        }
    }
    return names, types
}

func isIdentType(e ast.Expr, want string) bool {
    id, ok := e.(*ast.Ident)
    return ok && id.Name == want
}

func exprString(e ast.Expr) string {
    switch x := e.(type) {
    case *ast.Ident:
        return x.Name
    default:
        return fmt.Sprintf("%T", e)
    }
}

