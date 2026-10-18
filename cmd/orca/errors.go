package main

import "errors"

var (
	ErrArgumentRequired = errors.New("argument required")
	ErrTooManyArguments = errors.New("too many arguments")
)
