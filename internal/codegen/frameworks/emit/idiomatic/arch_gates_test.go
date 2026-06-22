package idiomatic

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testbedDirs returns the flagship framework package directories used as the
// conformance test-bed. They are the curated frameworks whose output the
// refactor's quality gates apply to (other frameworks carry pre-existing naming
// data — underscores, de-prefix collisions — orthogonal to this refactor).
func testbedDirs(t *testing.T) []string {
	t.Helper()
	root := filepath.Join("..", "..", "..", "..", "..", "opinionated", "idiomatic", "framework")
	var dirs []string
	for _, fw := range []string{"virtualization", "foundation", "appkit"} {
		d := filepath.Join(root, fw)
		if _, err := os.Stat(d); err != nil {
			t.Skipf("test-bed %s not present (%v); run `generate idiomatic`", fw, err)
		}
		dirs = append(dirs, d)
	}
	return dirs
}

// parseTestbed parses every generated .go file (excluding tests) in the test-bed.
func parseTestbed(t *testing.T) (*token.FileSet, []*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	var files []*ast.File
	for _, dir := range testbedDirs(t) {
		for _, path := range goFilesIn(t, dir) {
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			files = append(files, f)
		}
	}
	return fset, files
}

// goFilesIn returns the non-test .go files in dir (one package directory).
func goFilesIn(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	return files
}

// TestNoPackageStateKeyedByFramework enforces P6: the emitter holds no
// package-level mutable map keyed by *meta.FrameworkMeta. Per-framework derived
// data lives in frameworkContext for one EmitFrameworkWrappers call only. A
// regression (re-adding such a cache) fails here.
func TestNoPackageStateKeyedByFramework(t *testing.T) {
	fset := token.NewFileSet()
	for _, file := range goFilesIn(t, ".") {
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs := spec.(*ast.ValueSpec)
				typeStr := ""
				if vs.Type != nil {
					typeStr = exprString(vs.Type)
				}
				for _, v := range vs.Values {
					typeStr += exprString(v)
				}
				if strings.Contains(typeStr, "map[*meta.FrameworkMeta]") {
					t.Errorf("%s: package-level var keyed by *meta.FrameworkMeta is banned (P6); use frameworkContext", file)
				}
			}
		}
	}
}

// exprString renders an AST expression's leading structure enough to spot a map
// type literal; it is intentionally shallow (composite-literal and map-type
// nodes) rather than a full printer.
func exprString(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.MapType:
		return "map[" + exprString(x.Key) + "]" + exprString(x.Value)
	case *ast.StarExpr:
		return "*" + exprString(x.X)
	case *ast.SelectorExpr:
		return exprString(x.X) + "." + x.Sel.Name
	case *ast.Ident:
		return x.Name
	case *ast.CompositeLit:
		return exprString(x.Type)
	}
	return ""
}

// TestGofmtCleanTestbed enforces E1: every generated file in the test-bed is
// gofmt-clean (it passed through emit.WriteGoFile / go/format).
func TestGofmtCleanTestbed(t *testing.T) {
	for _, dir := range testbedDirs(t) {
		out, err := exec.Command("gofmt", "-l", dir).CombinedOutput()
		if err != nil {
			t.Fatalf("gofmt -l %s: %v\n%s", dir, err, out)
		}
		if strings.TrimSpace(string(out)) != "" {
			t.Errorf("gofmt found unformatted files:\n%s", out)
		}
	}
}

// TestNoUnderscoreExportsTestbed enforces E6 (MixedCaps) for the identifiers the
// refactor itself synthesises — the wrapper, provider, and mock type names. It
// deliberately does NOT cover enum constant members or a handful of raw
// class-method names: Apple's SDK encodes the OS version in constant names such
// as NSDateFormatterBehavior10_0 / NSPropertyListXMLFormat_v1_0, where the
// underscore is meaningful (macOS 10.0); stripping it would lose the version and
// break the one-to-one mapping to the SDK. That is a recorded divergence from a
// pure E6 reading, not a gap in the generator's naming.
func TestNoUnderscoreExportsTestbed(t *testing.T) {
	_, files := parseTestbed(t)
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				if strings.Contains(ts.Name.Name, "_") {
					t.Errorf("synthesised type name %s contains an underscore (E6)", ts.Name.Name)
				}
			}
		}
	}
}

// TestNamedReturnsTestbed enforces E7: a function with more than one return value
// uses named returns, so the godoc signature documents each value.
func TestNamedReturnsTestbed(t *testing.T) {
	_, files := parseTestbed(t)
	for _, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
				continue
			}
			// Count the values across fields (a single field can hold several
			// names of the same type, but generated returns are one value each).
			values := 0
			named := true
			for _, field := range fn.Type.Results.List {
				n := len(field.Names)
				if n == 0 {
					values++
					named = false
				} else {
					values += n
				}
			}
			if values > 1 && !named {
				t.Errorf("%s has %d return values but they are not named (E7)", fn.Name.Name, values)
			}
		}
	}
}

// TestInterfaceAssertionsTestbed enforces E11: every concrete wrapper type emits
// a var _ <Type>able = (*Type)(nil) assertion proving it satisfies its mock
// interface.
func TestInterfaceAssertionsTestbed(t *testing.T) {
	_, files := parseTestbed(t)
	for _, f := range files {
		// Collect the wrapper struct types and the mock assertions in this file.
		wrappers := map[string]bool{}
		asserts := map[string]bool{}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if ts.Name.IsExported() && isWrapperStruct(ts) {
						wrappers[ts.Name.Name] = true
					}
				}
			}
		}
		// Scan var declarations of the form var _ X = (*Y)(nil).
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "_" || len(vs.Values) != 1 {
					continue
				}
				if y := assertedConcreteType(vs.Values[0]); y != "" {
					asserts[y] = true
				}
			}
		}
		for w := range wrappers {
			if !asserts[w] {
				t.Errorf("wrapper type %s lacks a var _ %sable = (*%s)(nil) assertion (E11)", w, w, w)
			}
		}
	}
}

// isWrapperStruct reports whether a struct type is an Objective-C wrapper — it
// embeds a field (objref.Handle or a same-framework base) — as opposed to a plain
// value struct, which has only named fields.
func isWrapperStruct(ts *ast.TypeSpec) bool {
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return false
	}
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 { // embedded (anonymous) field
			return true
		}
	}
	return false
}

// assertedConcreteType extracts Y from an expression of the form (*Y)(nil).
func assertedConcreteType(e ast.Expr) string {
	call, ok := e.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return ""
	}
	paren, ok := call.Fun.(*ast.ParenExpr)
	if !ok {
		return ""
	}
	star, ok := paren.X.(*ast.StarExpr)
	if !ok {
		return ""
	}
	if id, ok := star.X.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// TestDocCommentsStartName enforces E2 on the func and wrapper-type API surface:
// every exported function/method and every exported wrapper struct type has a
// doc comment whose first word is the declared identifier. (Enum and value-struct
// declarations carry Apple prose and are out of scope for this gate.)
func TestDocCommentsStartName(t *testing.T) {
	_, files := parseTestbed(t)
	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				checkDocStarts(t, d.Doc, d.Name.Name)
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !ts.Name.IsExported() || !isWrapperStruct(ts) {
						continue // enums and plain value structs are out of scope
					}
					doc := ts.Doc
					if doc == nil {
						doc = d.Doc
					}
					checkDocStarts(t, doc, ts.Name.Name)
				}
			}
		}
	}
}

func checkDocStarts(t *testing.T, doc *ast.CommentGroup, name string) {
	t.Helper()
	if doc == nil {
		t.Errorf("%s has no doc comment (E2)", name)
		return
	}
	text := strings.TrimSpace(doc.Text())
	first, _, _ := strings.Cut(text, " ")
	if first != name {
		t.Errorf("%s doc starts with %q, want the identifier name (E2)", name, first)
	}
}

// TestNoFprintfOfGoInEmitter enforces P7: the emitter no longer assembles Go
// declarations by writing into a body/buffer with fmt.Fprintf — every construct
// is produced by a template.
func TestNoFprintfOfGoInEmitter(t *testing.T) {
	for _, path := range goFilesIn(t, ".") {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range []string{"Fprintf(&body", "Fprintf(&buf"} {
			if strings.Contains(string(data), bad) {
				t.Errorf("%s uses %s — Go must be rendered by templates, not string-built (P7)", path, bad)
			}
		}
	}
}

// TestRenderImportsClean enforces P1: the render package imports no metadata or
// type-mapping packages — every value it needs is already on the view. A render
// file that reaches for meta or typemap fails here.
func TestRenderImportsClean(t *testing.T) {
	fset := token.NewFileSet()
	for _, file := range goFilesIn(t, "render") {
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.HasSuffix(path, "/frameworks/meta") || strings.HasSuffix(path, "/frameworks/typemap") {
				t.Errorf("%s imports %s; render must not depend on meta/typemap (P1)", file, path)
			}
		}
	}
}
