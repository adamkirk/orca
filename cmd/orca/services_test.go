package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A message without a handler only fails when it's dispatched, so this makes
// sure every message the commands dispatch has one registered in GetBus.
func Test_AllDispatchedMessagesHaveHandlers(t *testing.T) {
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)

	dispatched := map[string]bool{}

	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		for _, msg := range findDispatchedMessages(t, path) {
			dispatched[msg] = true
		}
	}

	require.NotEmpty(t, dispatched)

	// Handlers are registered lazily, so this doesn't build any services
	handled := (&services{}).GetBus().HandledTypes()

	for msg := range dispatched {
		assert.Contains(t, handled, msg, "no handler registered in GetBus")
	}
}

// findDispatchedMessages returns the type of each message literal passed to
// Dispatch, e.g. "controller.UpCommand".
func findDispatchedMessages(t *testing.T, path string) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err)

	var msgs []string

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Dispatch" {
			return true
		}

		lit, ok := call.Args[1].(*ast.CompositeLit)
		if !ok {
			t.Errorf("%s: messages should be dispatched as literals, so they can be checked here", fset.Position(call.Pos()))
			return true
		}

		typ, ok := lit.Type.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		pkg, ok := typ.X.(*ast.Ident)
		if !ok {
			return true
		}

		msgs = append(msgs, pkg.Name+"."+typ.Sel.Name)

		return true
	})

	return msgs
}
