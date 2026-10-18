package bus_test

import (
	"context"
	"errors"
	"testing"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countQuery struct {
	bus.Returns[int]
	N int
}

func (countQuery) MessageName() string { return "count" }

type doCommand struct {
	bus.Returns[bus.Void]
}

func (doCommand) MessageName() string { return "do" }

func handleCount(ctx context.Context, q countQuery) (int, error) {
	return q.N * 2, nil
}

func Test_Dispatch_InfersResultType(t *testing.T) {
	b := bus.New()
	b.Handle(handleCount)

	// No type arguments needed, and res is statically an int
	res, err := b.Dispatch(context.Background(), countQuery{N: 21})

	require.NoError(t, err)
	assert.Equal(t, 42, res)
}

func Test_Dispatch_NoHandler(t *testing.T) {
	b := bus.New()

	_, err := b.Dispatch(context.Background(), doCommand{})

	assert.Equal(t, bus.ErrNoHandler{Name: "do"}, err)
}

func Test_Handle_PanicsOnDuplicate(t *testing.T) {
	b := bus.New()
	b.Handle(handleCount)

	assert.Panics(t, func() {
		b.Handle(handleCount)
	})
}

func Test_HandledTypes(t *testing.T) {
	b := bus.New()
	b.Handle(handleCount)
	b.HandleVoid(func(ctx context.Context, c doCommand) error {
		return nil
	})

	assert.Equal(t, []string{"bus_test.countQuery", "bus_test.doCommand"}, b.HandledTypes())
}

func Test_Dispatch_ReturnsHandlerError(t *testing.T) {
	b := bus.New()
	handlerErr := errors.New("boom")

	b.Handle(func(ctx context.Context, c doCommand) (bus.Void, error) {
		return bus.Void{}, handlerErr
	})

	_, err := b.Dispatch(context.Background(), doCommand{})

	assert.Same(t, handlerErr, err)
}

func Test_HandleVoid(t *testing.T) {
	b := bus.New()
	handlerErr := errors.New("boom")

	b.HandleVoid(func(ctx context.Context, c doCommand) error {
		return handlerErr
	})

	res, err := b.Dispatch(context.Background(), doCommand{})

	assert.Equal(t, bus.Void{}, res)
	assert.Same(t, handlerErr, err)
}

func Test_Middleware_Order(t *testing.T) {
	b := bus.New()
	calls := []string{}

	record := func(name string) bus.Middleware {
		return func(ctx context.Context, msg bus.Named, next bus.Next) (any, error) {
			calls = append(calls, name+":in:"+msg.MessageName())
			res, err := next(ctx)
			calls = append(calls, name+":out")

			return res, err
		}
	}

	b.Use(record("first"), record("second"))
	b.Before(func(ctx context.Context, c doCommand) error {
		calls = append(calls, "hook")

		return nil
	})
	b.Handle(func(ctx context.Context, c doCommand) (bus.Void, error) {
		calls = append(calls, "handler")

		return bus.Void{}, nil
	})

	_, err := b.Dispatch(context.Background(), doCommand{})

	require.NoError(t, err)
	assert.Equal(t, []string{
		"first:in:do",
		"second:in:do",
		"hook",
		"handler",
		"second:out",
		"first:out",
	}, calls)
}

func Test_Hooks_BeforeErrorStopsDispatch(t *testing.T) {
	b := bus.New()
	hookErr := errors.New("not allowed")
	handled := false
	afterRan := false

	b.Before(func(ctx context.Context, c doCommand) error {
		return hookErr
	})
	b.After(func(ctx context.Context, c doCommand, res bus.Void, err error) error {
		afterRan = true

		return nil
	})
	b.Handle(func(ctx context.Context, c doCommand) (bus.Void, error) {
		handled = true

		return bus.Void{}, nil
	})

	_, err := b.Dispatch(context.Background(), doCommand{})

	assert.ErrorIs(t, err, hookErr)

	var hookFailed bus.ErrHookFailed
	require.ErrorAs(t, err, &hookFailed)
	assert.Equal(t, bus.PhaseBefore, hookFailed.Phase)
	assert.Equal(t, "do", hookFailed.Name)

	assert.False(t, handled)
	assert.False(t, afterRan)
}

func Test_Hooks_AfterReceivesResultAndError(t *testing.T) {
	b := bus.New()
	handlerErr := errors.New("boom")

	var gotRes int
	var gotErr error

	b.Handle(func(ctx context.Context, q countQuery) (int, error) {
		return 7, handlerErr
	})
	b.After(func(ctx context.Context, q countQuery, res int, err error) error {
		gotRes, gotErr = res, err

		return nil
	})

	res, err := b.Dispatch(context.Background(), countQuery{})

	assert.Equal(t, 7, res)
	assert.Same(t, handlerErr, err)
	assert.Equal(t, 7, gotRes)
	assert.Same(t, handlerErr, gotErr)
}

func Test_Hooks_AfterErrorsAreJoined(t *testing.T) {
	b := bus.New()
	handlerErr := errors.New("handler")
	hookErr := errors.New("hook")

	b.Handle(func(ctx context.Context, c doCommand) (bus.Void, error) {
		return bus.Void{}, handlerErr
	})
	b.After(func(ctx context.Context, c doCommand, res bus.Void, err error) error {
		return hookErr
	})

	_, err := b.Dispatch(context.Background(), doCommand{})

	assert.ErrorIs(t, err, handlerErr)
	assert.ErrorIs(t, err, hookErr)
}

func Test_Hooks_TypedHooksOnlyFireForTheirMessage(t *testing.T) {
	b := bus.New()
	countHooks := 0
	events := []bus.Event{}

	b.Handle(handleCount)
	b.Handle(func(ctx context.Context, c doCommand) (bus.Void, error) {
		return bus.Void{}, nil
	})
	b.Before(func(ctx context.Context, q countQuery) error {
		countHooks++

		return nil
	})
	b.After(func(ctx context.Context, q countQuery, res int, err error) error {
		countHooks++

		return nil
	})
	b.AddHook(func(ctx context.Context, ev bus.Event) error {
		events = append(events, ev)

		return nil
	})

	_, err := b.Dispatch(context.Background(), doCommand{})
	require.NoError(t, err)

	assert.Equal(t, 0, countHooks)

	_, err = b.Dispatch(context.Background(), countQuery{N: 1})
	require.NoError(t, err)

	assert.Equal(t, 2, countHooks)

	// Untyped hooks see every message
	require.Len(t, events, 4)
	assert.Equal(t, bus.Event{Phase: bus.PhaseBefore, Name: "do", Message: doCommand{}}, events[0])
	assert.Equal(t, bus.Event{Phase: bus.PhaseAfter, Name: "do", Message: doCommand{}, Result: bus.Void{}}, events[1])
	assert.Equal(t, "count", events[2].Name)
	assert.Equal(t, 2, events[3].Result)
}
