// Package bus provides a command bus, through which all of orca's commands and
// queries are dispatched. It's the single place for behaviour that applies to
// every command, such as logging (middleware) and lifecycle hooks.
//
// Handlers, middleware and hooks are expected to be registered while wiring up
// the application, before anything is dispatched, so the bus isn't safe for
// concurrent registration.
package bus

import (
	"context"
	"fmt"
	"reflect"
	"slices"
)

// Void is the result type for messages that don't return anything.
type Void = struct{}

// Named is implemented by every message. The name is a stable identifier for
// the message (e.g. "up", "ws.init"), used for logging and matching hooks.
type Named interface {
	MessageName() string
}

// Returns is embedded in a message to declare the type it returns, e.g.
//
//	type UpCommand struct {
//		bus.Returns[bus.Void]
//		...
//	}
//
// It lets the compiler check that the handler for a message and the callers
// dispatching it agree on the result type.
type Returns[R any] struct{}

func (Returns[R]) result(R) {}

// Message is a command or query that returns R when dispatched.
type Message[R any] interface {
	Named
	result(R)
}

// HandlerFunc handles a single type of message.
type HandlerFunc[M Message[R], R any] func(ctx context.Context, msg M) (R, error)

type erasedHandler func(ctx context.Context, msg any) (any, error)

type Bus struct {
	handlers   map[reflect.Type]erasedHandler
	middleware []Middleware
	hooks      []Hook
}

func New() *Bus {
	return &Bus{
		handlers: map[reflect.Type]erasedHandler{},
	}
}

// Handle registers the handler for messages of type M. Registering a second
// handler for the same type is a programming error, so panics.
func (b *Bus) Handle[M Message[R], R any](h HandlerFunc[M, R]) {
	t := reflect.TypeFor[M]()

	if _, exists := b.handlers[t]; exists {
		panic(fmt.Sprintf("bus: handler already registered for %s", t))
	}

	b.handlers[t] = func(ctx context.Context, msg any) (any, error) {
		return h(ctx, msg.(M))
	}
}

// HandleVoid registers the handler for messages of type M, which don't return
// anything, so the handler only needs to return an error.
func (b *Bus) HandleVoid[M Message[Void]](h func(ctx context.Context, msg M) error) {
	b.Handle(func(ctx context.Context, msg M) (Void, error) {
		return Void{}, h(ctx, msg)
	})
}

// Dispatch sends the message to its handler, through any middleware and hooks.
func (b *Bus) Dispatch[R any](ctx context.Context, msg Message[R]) (R, error) {
	var zero R

	h, ok := b.handlers[reflect.TypeOf(msg)]

	if !ok {
		return zero, ErrNoHandler{Name: msg.MessageName()}
	}

	next := func(ctx context.Context) (any, error) {
		return b.runWithHooks(ctx, msg, h)
	}

	// Wrapped in reverse, so the first middleware registered is the outermost
	for i := len(b.middleware) - 1; i >= 0; i-- {
		mw, inner := b.middleware[i], next

		next = func(ctx context.Context) (any, error) {
			return mw(ctx, msg, inner)
		}
	}

	res, err := next(ctx)

	// Result may be nil when there's an error
	r, _ := res.(R)

	return r, err
}

// HandledTypes returns the type of every message that has a handler, e.g.
// "controller.UpCommand".
func (b *Bus) HandledTypes() []string {
	types := make([]string, 0, len(b.handlers))

	for t := range b.handlers {
		types = append(types, t.String())
	}

	slices.Sort(types)

	return types
}

type ErrNoHandler struct {
	Name string
}

func (e ErrNoHandler) Error() string {
	return fmt.Sprintf("no handler registered for message '%s'", e.Name)
}
