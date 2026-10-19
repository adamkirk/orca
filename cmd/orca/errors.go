package main

import "errors"

var (
	ErrArgumentRequired = errors.New("argument required")
	ErrUnknownCommand   = errors.New("unknown command")
	ErrUnneededDashes   = errors.New("unneeded '--'")
)
