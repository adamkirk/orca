package bus

import (
	"context"
	"log/slog"
	"time"
)

// Next calls the rest of the middleware chain, and eventually the handler.
type Next func(ctx context.Context) (any, error)

// Middleware wraps the dispatch of every message. The message and result are
// untyped, as interfaces can't declare generic methods.
type Middleware func(ctx context.Context, msg Named, next Next) (any, error)

// Use adds middleware to the bus. Middleware runs in the order it's added, and
// wraps the hooks as well as the handler.
func (b *Bus) Use(mw ...Middleware) {
	b.middleware = append(b.middleware, mw...)
}

// LoggingMiddleware logs each dispatch at debug level.
func LoggingMiddleware(ctx context.Context, msg Named, next Next) (any, error) {
	start := time.Now()

	slog.Debug("dispatching message", "message", msg.MessageName())

	res, err := next(ctx)

	slog.Debug(
		"dispatched message",
		"message", msg.MessageName(),
		"duration", time.Since(start),
		"err", err,
	)

	return res, err
}
