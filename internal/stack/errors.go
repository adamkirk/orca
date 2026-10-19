package stack

import "errors"

var (
	ErrUnknownService = errors.New("unknown service")
	ErrNoProject      = errors.New("no project")
)
