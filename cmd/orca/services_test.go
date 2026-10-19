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

// A handler that reads the ExecutionContext only gets one if its message has a
// target registered, otherwise it fails when dispatched. This makes sure every
// one has, in GetExecutionContextMiddleware.
func Test_ExecutionContextHandlersHaveTargets(t *testing.T) {
	var needed []string

	err := filepath.WalkDir("../../internal", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}

		needed = append(needed, findExecutionContextMessages(t, path)...)

		return nil
	})

	require.NoError(t, err)
	require.NotEmpty(t, needed)

	targeted := (&services{}).GetExecutionContextMiddleware().TargetedTypes()

	for _, msg := range needed {
		assert.Contains(t, targeted, msg, "no execution target registered in GetExecutionContextMiddleware")
	}
}

// findExecutionContextMessages returns the message type, e.g.
// "stack.UpCommand", of each handler that calls workspace.ExecutionContextFrom.
func findExecutionContextMessages(t *testing.T, path string) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err)

	var msgs []string

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !callsExecutionContextFrom(fn) {
			continue
		}

		// Handlers take (ctx context.Context, msg M)
		var paramTypes []ast.Expr
		for _, field := range fn.Type.Params.List {
			for range max(len(field.Names), 1) {
				paramTypes = append(paramTypes, field.Type)
			}
		}

		var msgType *ast.Ident
		if len(paramTypes) == 2 {
			msgType, _ = paramTypes[1].(*ast.Ident)
		}

		if msgType == nil {
			t.Errorf("%s: ExecutionContextFrom should only be called from handlers, taking (ctx, msg)", fset.Position(fn.Pos()))
			continue
		}

		msgs = append(msgs, file.Name.Name+"."+msgType.Name)
	}

	return msgs
}

func callsExecutionContextFrom(fn *ast.FuncDecl) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "workspace" && sel.Sel.Name == "ExecutionContextFrom" {
			found = true
		}

		return !found
	})

	return found
}

// findDispatchedMessages returns the type of each message literal passed to
// Dispatch, e.g. "stack.UpCommand".
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
