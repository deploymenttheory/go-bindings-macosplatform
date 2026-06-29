//go:build darwin

package idiomatic

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// scanHandAuthored parses every non-generated .go file already present in the
// package directory and returns the methods (keyed by receiver type name) and
// the package-level function names a human has written by hand. The emitter
// uses these to avoid colliding with hand-crafted refinements.
func scanHandAuthored(outDir string) (map[string]map[string]bool, map[string]bool, error) {
	methods := map[string]map[string]bool{}
	funcs := map[string]bool{}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		if os.IsNotExist(err) {
			return methods, funcs, nil
		}
		return nil, nil, err
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_generated.go") {
			continue
		}
		f, perr := parser.ParseFile(
			fset,
			filepath.Join(outDir, name),
			nil,
			parser.SkipObjectResolution,
		)
		if perr != nil {
			// A stale or half-written file must not abort generation; skip it.
			continue
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Recv == nil {
				funcs[fn.Name.Name] = true
				continue
			}
			recv := receiverTypeName(fn.Recv)
			if recv == "" {
				continue
			}
			if methods[recv] == nil {
				methods[recv] = map[string]bool{}
			}
			methods[recv][fn.Name.Name] = true
		}
	}
	return methods, funcs, nil
}

// removeGeneratedFiles deletes the package's previously generated *_generated.go
// files so a regeneration never leaves stale output behind. Hand-authored files
// (which never carry the _generated.go suffix) and doc.go are kept; doc.go is
// rewritten in place by the emitter.
func removeGeneratedFiles(outDir string) error {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_generated.go") {
			continue
		}
		if err := os.Remove(filepath.Join(outDir, e.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}
