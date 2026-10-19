package logging

import (
	"log/slog"
	"os"
)

const LogLevelNone = 10
const LogLevelNoneName = "none"

type config interface {
	GetLoggingLevel() string
	GetLoggingFormat() string
}

func NewSlogHandler(cfg config) (slog.Handler, error) {
	l := slog.Level(LogLevelNone)

	if cfg.GetLoggingLevel() != LogLevelNoneName {
		err := l.UnmarshalText([]byte(cfg.GetLoggingLevel()))

		if err != nil {
			return nil, err
		}
	}

	opts := &slog.HandlerOptions{
		Level:     l,
		AddSource: true,
	}

	var handler slog.Handler

	if cfg.GetLoggingFormat() == "text" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	return handler, nil
}
