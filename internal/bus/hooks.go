package bus

import (
	"context"
	"errors"
	"fmt"
)

type Phase string

const (
	PhaseBefore Phase = "before"
	PhaseAfter  Phase = "after"
)

// Event describes a point in the lifecycle of a dispatch. Result and Err are
// only set for PhaseAfter.
type Event struct {
	Phase   Phase
	Name    string
	Message any
	Result  any
	Err     error
}

// Hook is called for every message at each phase of its dispatch.
//
// An error from a PhaseBefore hook stops the dispatch, so the handler and any
// later hooks don't run. PhaseAfter hooks always run, whether or not the
// handler failed, and their errors are joined with the handler's.
type Hook func(ctx context.Context, ev Event) error

// AddHook adds hooks to the bus, which run in the order they're added.
func (b *Bus) AddHook(hooks ...Hook) {
	b.hooks = append(b.hooks, hooks...)
}

// Before adds a hook that runs before messages of type M are handled.
func (b *Bus) Before[M Message[R], R any](fn func(ctx context.Context, msg M) error) {
	b.AddHook(func(ctx context.Context, ev Event) error {
		msg, ok := ev.Message.(M)

		if ev.Phase != PhaseBefore || !ok {
			return nil
		}

		return fn(ctx, msg)
	})
}

// After adds a hook that runs after messages of type M are handled, with the
// handler's result and error.
func (b *Bus) After[M Message[R], R any](fn func(ctx context.Context, msg M, res R, err error) error) {
	b.AddHook(func(ctx context.Context, ev Event) error {
		msg, ok := ev.Message.(M)

		if ev.Phase != PhaseAfter || !ok {
			return nil
		}

		res, _ := ev.Result.(R)

		return fn(ctx, msg, res, ev.Err)
	})
}

func (b *Bus) runWithHooks(ctx context.Context, msg Named, h erasedHandler) (any, error) {
	name := msg.MessageName()

	for _, hook := range b.hooks {
		ev := Event{Phase: PhaseBefore, Name: name, Message: msg}

		if err := hook(ctx, ev); err != nil {
			return nil, HookFailedError{Phase: PhaseBefore, Name: name, Err: err}
		}
	}

	res, err := h(ctx, msg)

	var hookErrs []error

	for _, hook := range b.hooks {
		ev := Event{Phase: PhaseAfter, Name: name, Message: msg, Result: res, Err: err}

		if hookErr := hook(ctx, ev); hookErr != nil {
			hookErrs = append(hookErrs, HookFailedError{Phase: PhaseAfter, Name: name, Err: hookErr})
		}
	}

	// Leave the handler's error as is unless there's something to join it with
	if len(hookErrs) == 0 {
		return res, err
	}

	return res, errors.Join(append([]error{err}, hookErrs...)...)
}

type HookFailedError struct {
	Phase Phase
	Name  string
	Err   error
}

func (e HookFailedError) Error() string {
	return fmt.Sprintf("%s hook for '%s' failed: %s", e.Phase, e.Name, e.Err)
}

func (e HookFailedError) Unwrap() error {
	return e.Err
}
