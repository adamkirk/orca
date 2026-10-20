package tui

import "errors"

var (
	ErrUserAbortedExecution = errors.New("user aborted execution")

	// ErrNotInteractive means orca needed to ask something, but there's no
	// terminal to ask it in, e.g. in a script or pipeline.
	ErrNotInteractive = errors.New("not running in an interactive terminal")
)
