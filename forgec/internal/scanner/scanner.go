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
    ParamTypes []string // Go types (int32|int64)
    HasValue   bool     // true if function returns a value before error
    RetType    string   // value type ("int32"|"int64") when HasValue=true
}

// Struct represents a struct to export to C.
type Struct struct {
    Name   string
    Fields []Field
}

type Field struct {
    Name       string // original Go field name
    GoType     string // e.g., string, int32, int64, time.Time, map[string]int64
    CType      string // e.g., const char*, int32_t, int64_t, double
    ExportName string // C field name (may add suffix like JSON/Unix)
}

// ScanExported scans a package directory for top-level functions annotated with `capi:export`.
// Enforces signature: func(...int32) (int32, error)
func ScanExported(pkgDir string) ([]Func, []Struct, error) {
    info, err := os.Stat(pkgDir)
    if err != nil {
        return nil, nil, err
    }
    if !info.IsDir() {
        return nil, nil, fmt.Errorf("pkg path is not a directory: %s", pkgDir)
    }

    fset := token.NewFileSet()
    pkgs, err := parser.ParseDir(fset, pkgDir, nil, parser.ParseComments)
    if err != nil {
        return nil, nil, err
    }

    var out []Func
    var structs []Struct
    for _, pkg := range pkgs {
        for _, f := range pkg.Files {
            // Only consider files within the provided dir (avoid vendor, etc.)
            if !strings.HasPrefix(fset.Position(f.Package).Filename, filepath.Clean(pkgDir)) {
                continue
            }
            for _, decl := range f.Decls {
                switch d := decl.(type) {
                case *ast.FuncDecl:
                    fn := d
                    if fn.Recv != nil || fn.Name == nil {
                        continue
                    }
                    if fn.Doc == nil || !hasExportTag(fn.Doc.List) {
                        continue
                    }
                    hasVal, retType, err := validateSignature(fn.Type)
                    if err != nil {
                        return nil, nil, fmt.Errorf("%s: %w", fn.Name.Name, err)
                    }
                    pnames, ptypes := collectParams(fn.Type)
                    out = append(out, Func{
                        Name:       fn.Name.Name,
                        CName:      fn.Name.Name,
                        Params:     pnames,
                        ParamTypes: ptypes,
                        HasValue:   hasVal,
                        RetType:    retType,
                    })
                case *ast.GenDecl:
                    if d.Tok != token.TYPE {
                        continue
                    }
                    if d.Doc == nil || !hasExportTag(d.Doc.List) {
                        // allow per-spec doc too
                        // we will also check TypeSpec.Doc below
                    }
                    for _, spec := range d.Specs {
                        ts, ok := spec.(*ast.TypeSpec)
                        if !ok {
                            continue
                        }
                        var hasTag bool
                        if d.Doc != nil && hasExportTag(d.Doc.List) {
                            hasTag = true
                        }
                        if ts.Doc != nil && hasExportTag(ts.Doc.List) {
                            hasTag = true
                        }
                        st, ok := ts.Type.(*ast.StructType)
                        if !ok || !hasTag {
                            continue
                        }
                        s, err := collectStruct(ts.Name.Name, st)
                        if err != nil {
                            return nil, nil, fmt.Errorf("struct %s: %w", ts.Name.Name, err)
                        }
                        structs = append(structs, s)
                    }
                }
            }
        }
    }
    return out, structs, nil
}

func hasExportTag(list []*ast.Comment) bool {
    for _, c := range list {
        if strings.Contains(c.Text, "capi:export") {
            return true
        }
    }
    return false
}

// validateSignature now supports:
// - params: any number, each int32 or int64
// - results: either `error` only, or `(int32|int64, error)`
// returns (hasValue, retType, error)
func validateSignature(t *ast.FuncType) (bool, string, error) {
    if t.Params != nil {
        for _, f := range t.Params.List {
            if !(isIdentType(f.Type, "int32") || isIdentType(f.Type, "int64")) {
                return false, "", fmt.Errorf("param must be int32 or int64: %s", exprString(f.Type))
            }
        }
    }
    if t.Results == nil || len(t.Results.List) == 0 || len(t.Results.List) > 2 {
        return false, "", errors.New("result must be error or (int32|int64, error)")
    }
    if len(t.Results.List) == 1 {
        if !isIdentType(t.Results.List[0].Type, "error") {
            return false, "", errors.New("single result must be error")
        }
        return false, "", nil
    }
    // two results
    // first result must be int32 or int64
    rt := t.Results.List[0].Type
    if !(isIdentType(rt, "int32") || isIdentType(rt, "int64")) {
        return false, "", fmt.Errorf("first result must be int32 or int64: %s", exprString(rt))
    }
    if !isIdentType(t.Results.List[1].Type, "error") {
        return false, "", fmt.Errorf("second result must be error: %s", exprString(t.Results.List[1].Type))
    }
    r := "int32"
    if isIdentType(rt, "int64") { r = "int64" }
    return true, r, nil
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
    case *ast.SelectorExpr:
        // e.g., time.Time
        if pkg, ok := x.X.(*ast.Ident); ok {
            return pkg.Name + "." + x.Sel.Name
        }
        return x.Sel.Name
    case *ast.MapType:
        return "map"
    default:
        return fmt.Sprintf("%T", e)
    }
}

func collectStruct(name string, st *ast.StructType) (Struct, error) {
    var fields []Field
    for _, f := range st.Fields.List {
        // skip embedded/anonymous
        if len(f.Names) == 0 {
            continue
        }
        gt := exprString(f.Type)
        ctype, exportName, ok := mapGoToCField(f.Names[0].Name, f.Type)
        if !ok {
            return Struct{}, fmt.Errorf("unsupported field type: %s", gt)
        }
        fields = append(fields, Field{ Name: f.Names[0].Name, GoType: gt, CType: ctype, ExportName: exportName })
    }
    return Struct{ Name: name, Fields: fields }, nil
}

func mapGoToCField(base string, t ast.Expr) (ctype string, exportName string, ok bool) {
    exportName = base
    switch tt := t.(type) {
    case *ast.Ident:
        switch tt.Name {
        case "string":
            return "const char*", exportName, true
        case "int32":
            return "int32_t", exportName, true
        case "int64":
            return "int64_t", exportName, true
        case "bool":
            return "int32_t", exportName, true
        case "float64":
            return "double", exportName, true
        default:
            return "", "", false
        }
    case *ast.SelectorExpr:
        // e.g., time.Time -> int64 unix
        if id, ok := tt.X.(*ast.Ident); ok && id.Name == "time" && tt.Sel.Name == "Time" {
            return "int64_t", base + "Unix", true
        }
        return "", "", false
    case *ast.MapType:
        // map[string]int64 -> JSON string
        if k, ok := tt.Key.(*ast.Ident); ok && k.Name == "string" {
            if v, ok := tt.Value.(*ast.Ident); ok && v.Name == "int64" {
                return "const char*", base + "JSON", true
            }
        }
        return "", "", false
    default:
        return "", "", false
    }
}
