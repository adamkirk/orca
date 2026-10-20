package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
)

// ExecutionTarget is the workspace and project a message runs against, as given
// by the user. Either may be empty, in which case they're resolved as usual
// (see Resolver.Resolve).
type ExecutionTarget struct {
	Workspace string
	Project   string

	// When the workspace can't be determined, carry on with only the working
	// directory, instead of failing. Used by commands that are still useful
	// outside of orca, like the git commands in any repository.
	WorkspaceOptional bool

	// The command only applies to projects with a compose stack, so fails for a
	// library project, whether it's given or worked out from the directory.
	ServicesOnly bool
}

type resolver interface {
	Resolve(ws string, project string) (ExecutionContext, error)
}

// ExecutionContextMiddleware resolves the ExecutionContext for a message before
// it's handled, and passes it along in the context.Context, to be read with
// ExecutionContextFrom.
//
// Only messages with a target registered (see Target) are resolved, others are
// passed through as they are.
type ExecutionContextMiddleware struct {
	resolver resolver
	targets  map[reflect.Type]func(msg any) ExecutionTarget
}

func NewExecutionContextMiddleware(resolver resolver) *ExecutionContextMiddleware {
	return &ExecutionContextMiddleware{
		resolver: resolver,
		targets:  map[reflect.Type]func(msg any) ExecutionTarget{},
	}
}

// Target registers how to get the target for messages of type M. Registering a
// second target for the same type is a programming error, so panics.
func (m *ExecutionContextMiddleware) Target[M bus.Named](fn func(msg M) ExecutionTarget) {
	t := reflect.TypeFor[M]()

	if _, exists := m.targets[t]; exists {
		panic(fmt.Sprintf("execution target already registered for %s", t))
	}

	m.targets[t] = func(msg any) ExecutionTarget {
		return fn(msg.(M))
	}
}

// TargetedTypes returns the type of every message that has a target, e.g.
// "controller.UpCommand".
func (m *ExecutionContextMiddleware) TargetedTypes() []string {
	types := make([]string, 0, len(m.targets))

	for t := range m.targets {
		types = append(types, t.String())
	}

	slices.Sort(types)

	return types
}

// Handle is the bus.Middleware.
func (m *ExecutionContextMiddleware) Handle(ctx context.Context, msg bus.Named, next bus.Next) (any, error) {
	targetFn, ok := m.targets[reflect.TypeOf(msg)]

	if !ok {
		return next(ctx)
	}

	target := targetFn(msg)
	execCtx, err := m.resolve(target)

	if err != nil {
		return nil, err
	}

	if target.ServicesOnly && execCtx.Project != nil && execCtx.Project.IsLibrary() {
		name := execCtx.Project.Name

		return nil, usererr.Wrap(
			fmt.Errorf("%w: %s", ErrLibraryProject, name),
			fmt.Sprintf("%s is a library project, so it has no compose stack for this command to act on!", name),
		)
	}

	return next(WithExecutionContext(ctx, execCtx))
}

func (m *ExecutionContextMiddleware) resolve(target ExecutionTarget) (ExecutionContext, error) {
	execCtx, err := m.resolver.Resolve(target.Workspace, target.Project)

	if err == nil {
		return execCtx, nil
	}

	if !errors.Is(err, ErrCouldNotDetermineWorkspace) {
		return ExecutionContext{}, usererr.Wrap(err, "Failed to determine the workspace or project!")
	}

	// A specific project can't be found without its workspace
	if !target.WorkspaceOptional || target.Project != "" {
		return ExecutionContext{}, usererr.Wrap(
			err,
			"Could not determine the workspace; run this from within a project, pass --workspace, or choose one with 'orca ws switch'.",
		)
	}

	wd, err := os.Getwd()

	if err != nil {
		return ExecutionContext{}, err
	}

	return ExecutionContext{
		WorkingDirectory: wd,
	}, nil
}

type executionContextKey struct{}

func WithExecutionContext(ctx context.Context, execCtx ExecutionContext) context.Context {
	return context.WithValue(ctx, executionContextKey{}, execCtx)
}

// ExecutionContextFrom returns the ExecutionContext resolved for the message
// being handled. It should only be called from handlers, whose message has a
// target registered with the ExecutionContextMiddleware.
func ExecutionContextFrom(ctx context.Context) (ExecutionContext, error) {
	execCtx, ok := ctx.Value(executionContextKey{}).(ExecutionContext)

	if !ok {
		return ExecutionContext{}, ErrNoExecutionContext
	}

	return execCtx, nil
}

// ErrNoExecutionContext means a handler needed an ExecutionContext, but its
// message has no target registered, so it was never resolved. This is a bug.
var ErrNoExecutionContext = errors.New("no execution context was resolved for this message, it needs a target registering")
