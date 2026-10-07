package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Functions allowed to exit the process directly, see exit.go.
var allowedExitFuncs = map[string]bool{
	"main": true,
	"exit": true,
}

// Exiting directly skips stopping plugin processes, leaving them orphaned. This
// makes sure everything goes through exit/checkErr, or returns an error instead.
func Test_NoDirectExits(t *testing.T) {
	var violations []string

	for _, dir := range []string{".", "../../internal"} {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}

			violations = append(violations, findDirectExits(t, path)...)

			return nil
		})

		require.NoError(t, err)
	}

	assert.Empty(t, violations, "use exit/checkErr (cmd/orca/exit.go) or return an error instead")
}

func findDirectExits(t *testing.T, path string) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err)

	var violations []string

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)

		// Methods are never allowed, only the top level functions listed above
		if ok && fn.Recv == nil && allowedExitFuncs[fn.Name.Name] && filepath.Dir(path) == "." {
			continue
		}

		ast.Inspect(decl, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			if (pkg.Name == "os" && sel.Sel.Name == "Exit") ||
				(pkg.Name == "cobra" && sel.Sel.Name == "CheckErr") ||
				(pkg.Name == "log" && strings.HasPrefix(sel.Sel.Name, "Fatal")) {
				violations = append(violations, fset.Position(call.Pos()).String()+": "+pkg.Name+"."+sel.Sel.Name)
			}

			return true
		})
	}

	return violations
}
